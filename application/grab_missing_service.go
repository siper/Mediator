package application

import (
	"context"
	"log/slog"
	"sort"

	"stersh.ru/mediator/domain"
)

type GrabMissingService struct {
	grabSvc     *GrabService
	partSvc     *PartService
	mediaSvc    *MediaService
	releaseSvc  *ReleaseService
	qualityRepo domain.QualityProfileRepository
	groupRepo   domain.PartGroupRepository
}

func NewGrabMissingService(
	grabSvc *GrabService,
	partSvc *PartService,
	mediaSvc *MediaService,
	releaseSvc *ReleaseService,
	qualityRepo domain.QualityProfileRepository,
	groupRepo domain.PartGroupRepository,
) *GrabMissingService {
	return &GrabMissingService{
		grabSvc:     grabSvc,
		partSvc:     partSvc,
		mediaSvc:    mediaSvc,
		releaseSvc:  releaseSvc,
		qualityRepo: qualityRepo,
		groupRepo:   groupRepo,
	}
}

func (s *GrabMissingService) Run(ctx context.Context) error {
	wanted, err := s.collectWanted()
	if err != nil {
		return err
	}
	if len(wanted) == 0 {
		return nil
	}

	byMedia := make(map[domain.ID][]domain.Part)
	for _, p := range wanted {
		byMedia[p.MediaId] = append(byMedia[p.MediaId], p)
	}

	var grabbed int
	for mediaID, parts := range byMedia {
		media, err := s.mediaSvc.GetByID(mediaID)
		if err != nil {
			slog.Warn("grab-missing: media not found", "id", mediaID, "err", err)
			continue
		}
		count, err := s.processMedia(ctx, *media, parts)
		if err != nil {
			slog.Warn("grab-missing: media failed", "id", mediaID, "err", err)
			continue
		}
		grabbed += count
	}
	if grabbed > 0 {
		slog.Info("grab-missing: submitted grabs", "count", grabbed)
	}
	return nil
}

func (s *GrabMissingService) ProcessMedia(ctx context.Context, mediaID domain.ID) (int, error) {
	media, err := s.mediaSvc.GetByID(mediaID)
	if err != nil {
		return 0, err
	}
	parts, err := s.partSvc.GetByMediaID(mediaID)
	if err != nil {
		return 0, err
	}
	wanted := filterWantedParts(parts)
	return s.processMedia(ctx, *media, wanted)
}

func (s *GrabMissingService) collectWanted() ([]domain.Part, error) {
	page, limit := 1, 100
	var all []domain.Part
	for {
		parts, err := s.partSvc.Wanted(page, limit)
		if err != nil {
			return nil, err
		}
		all = append(all, parts...)
		if len(parts) < limit {
			return all, nil
		}
		page++
	}
}

func (s *GrabMissingService) processMedia(ctx context.Context, media domain.Media, parts []domain.Part) (int, error) {
	if media.ProviderID == string(domain.SourceAuthorToday) {
		count, err := s.grabAuthorToday(ctx, media, parts)
		if err != nil {
			return 0, err
		}
		if count > 0 {
			return count, nil
		}
	}
	switch media.Type {
	case domain.MediaTypeMovie, domain.MediaTypeBook:
		return s.grabSingleType(ctx, media, parts, media.Type)
	case domain.MediaTypeMusicAlbum:
		return s.grabAlbum(ctx, media, parts)
	case domain.MediaTypeSeries:
		return s.grabSeries(ctx, media, parts)
	default:
		slog.Debug("grab-missing: unsupported media type", "id", media.Id, "type", media.Type)
		return 0, nil
	}
}

func (s *GrabMissingService) grabAuthorToday(ctx context.Context, media domain.Media, parts []domain.Part) (int, error) {
	if !s.grabSvc.SupportsProvider(media.ProviderID) {
		return 0, nil
	}
	for _, p := range parts {
		if s.grabSvc.HasActiveGrab(media.Id, p.Id) {
			continue
		}
		if _, err := s.grabSvc.Grab(ctx, domain.GrabTarget{
			MediaID:      media.Id,
			PartIds:      []domain.ID{p.Id},
			ProviderName: media.ProviderID,
			Release:      &domain.Release{Title: media.Name},
		}); err != nil {
			slog.Warn("grab-missing: author_today grab failed", "id", media.Id, "err", err)
			continue
		}
		return 1, nil
	}
	return 0, nil
}

func (s *GrabMissingService) grabSingleType(ctx context.Context, media domain.Media, parts []domain.Part, mt domain.MediaType) (int, error) {
	profile, ok := s.profileForMedia(media)
	if !ok {
		return 0, nil
	}
	titles := MediaMatchTitles(media)
	for _, p := range parts {
		if s.grabSvc.HasActiveGrab(media.Id, p.Id) {
			continue
		}
		scored, err := s.releaseSvc.Search(ctx, media.Name, mt, profile, nil, titles...)
		if err != nil {
			slog.Warn("grab-missing: search failed", "id", media.Id, "type", mt, "err", err)
			return 0, nil
		}
		rel, ok := firstSeededRelease(scored)
		if !ok {
			slog.Debug("grab-missing: no releases", "id", media.Id, "type", mt)
			return 0, nil
		}
		if _, err := s.grabSvc.Grab(ctx, domain.GrabTarget{
			MediaID: media.Id,
			PartIds: []domain.ID{p.Id},
			Release: &rel,
		}); err != nil {
			slog.Warn("grab-missing: grab failed", "id", media.Id, "type", mt, "err", err)
			continue
		}
		return 1, nil
	}
	return 0, nil
}

func (s *GrabMissingService) grabAlbum(ctx context.Context, media domain.Media, parts []domain.Part) (int, error) {
	profile, ok := s.profileForMedia(media)
	if !ok {
		return 0, nil
	}
	if len(parts) == 0 {
		return 0, nil
	}
	for _, p := range parts {
		if s.grabSvc.HasActiveGrab(media.Id, p.Id) {
			slog.Debug("grab-missing: album has active grab, skipping", "id", media.Id)
			return 0, nil
		}
	}
	partIds := make([]domain.ID, 0, len(parts))
	for _, p := range parts {
		partIds = append(partIds, p.Id)
	}
	scored, err := s.releaseSvc.Search(ctx, media.Name, domain.MediaTypeMusicAlbum, profile, nil, MediaMatchTitles(media)...)
	if err != nil {
		slog.Warn("grab-missing: album search failed", "id", media.Id, "err", err)
		return 0, nil
	}
	rel, ok := firstSeededRelease(scored)
	if !ok {
		slog.Debug("grab-missing: no releases for album", "id", media.Id)
		return 0, nil
	}
	if _, err := s.grabSvc.Grab(ctx, domain.GrabTarget{
		MediaID: media.Id,
		PartIds: partIds,
		Release: &rel,
	}); err != nil {
		slog.Warn("grab-missing: album grab failed", "id", media.Id, "err", err)
		return 0, nil
	}
	return 1, nil
}

func (s *GrabMissingService) grabSeries(ctx context.Context, media domain.Media, parts []domain.Part) (int, error) {
	profile, ok := s.profileForMedia(media)
	if !ok {
		return 0, nil
	}
	groups, err := s.groupRepo.GetByMediaID(media.Id)
	if err != nil {
		return 0, err
	}
	seasonOf := make(map[domain.ID]int, len(groups))
	for _, g := range groups {
		seasonOf[g.Id] = g.Order
	}

	taken := make(map[domain.ID]bool)
	var remaining []domain.Part
	bySeason := make(map[int][]domain.Part)
	for _, p := range parts {
		if p.GroupId == nil {
			continue
		}
		season, ok := seasonOf[*p.GroupId]
		if !ok {
			continue
		}
		if s.grabSvc.HasActiveGrab(media.Id, p.Id) {
			taken[p.Id] = true
			continue
		}
		remaining = append(remaining, p)
		bySeason[season] = append(bySeason[season], p)
	}
	if len(remaining) == 0 {
		return 0, nil
	}

	candidates := s.collectSeriesReleases(ctx, media, profile, bySeason)
	if len(candidates) == 0 {
		slog.Debug("grab-missing: no releases for series", "id", media.Id)
		return 0, nil
	}

	seen := make(map[string]bool)
	var grabbed int
	for {
		var leftover []domain.Part
		for _, p := range remaining {
			if !taken[p.Id] {
				leftover = append(leftover, p)
			}
		}
		if len(leftover) == 0 {
			return grabbed, nil
		}
		ranked := rankByPartCoverage(candidates, leftover, seasonOf)
		picked := false
		for _, candidate := range ranked {
			key := releaseIdentity(candidate.Release)
			if seen[key] || s.grabSvc.HasActiveReleaseTitle(media.Id, candidate.Release.Title) {
				continue
			}
			covered := coveredWantedParts(candidate.Parsed, leftover, seasonOf)
			if len(covered) == 0 {
				continue
			}
			partIds := make([]domain.ID, len(covered))
			for i, p := range covered {
				partIds[i] = p.Id
			}
			rel := candidate.Release
			if _, err := s.grabSvc.Grab(ctx, domain.GrabTarget{
				MediaID: media.Id,
				PartIds: partIds,
				Release: &rel,
			}); err != nil {
				slog.Warn("grab-missing: series grab failed", "id", media.Id, "err", err)
				continue
			}
			seen[key] = true
			for _, id := range partIds {
				taken[id] = true
			}
			grabbed++
			picked = true
			break
		}
		if !picked {
			return grabbed, nil
		}
	}
}

func (s *GrabMissingService) collectSeriesReleases(ctx context.Context, media domain.Media, profile *domain.QualityProfile, bySeason map[int][]domain.Part) []ScoredRelease {
	titles := MediaMatchTitles(media)
	merged := make([]ScoredRelease, 0)
	seen := make(map[string]int)
	add := func(batch []ScoredRelease) {
		for _, item := range batch {
			if item.Release.Seeders <= 0 {
				continue
			}
			id := releaseIdentity(item.Release)
			if prev, ok := seen[id]; ok {
				if item.Release.Seeders > merged[prev].Release.Seeders {
					merged[prev] = item
				}
				continue
			}
			seen[id] = len(merged)
			merged = append(merged, item)
		}
	}

	unscoped, err := s.releaseSvc.Search(ctx, media.Name, domain.MediaTypeSeries, profile, nil, titles...)
	if err != nil {
		slog.Warn("grab-missing: series search failed", "id", media.Id, "err", err)
	} else {
		add(unscoped)
	}

	seasons := make([]int, 0, len(bySeason))
	for season := range bySeason {
		seasons = append(seasons, season)
	}
	sort.Ints(seasons)
	for _, season := range seasons {
		scored, err := s.releaseSvc.Search(ctx, media.Name, domain.MediaTypeSeries, profile, &SeriesTarget{Season: season}, titles...)
		if err != nil {
			slog.Warn("grab-missing: series search failed", "id", media.Id, "season", season, "err", err)
			continue
		}
		add(scored)
	}
	return merged
}

func (s *GrabMissingService) profileForMedia(media domain.Media) (*domain.QualityProfile, bool) {
	if media.QualityProfileID == nil {
		slog.Debug("grab-missing: no quality profile, skipping", "id", media.Id)
		return nil, false
	}
	profile, err := s.qualityRepo.GetById(*media.QualityProfileID)
	if err != nil {
		slog.Warn("grab-missing: quality profile not found", "id", media.Id, "profile_id", *media.QualityProfileID, "err", err)
		return nil, false
	}
	return profile, true
}

func filterWantedParts(parts []domain.Part) []domain.Part {
	var out []domain.Part
	for _, p := range parts {
		if p.Monitored && p.Path == nil {
			out = append(out, p)
		}
	}
	return out
}

func firstSeededRelease(scored []ScoredRelease) (domain.Release, bool) {
	for _, s := range scored {
		if s.Release.Seeders > 0 {
			return s.Release, true
		}
	}
	return domain.Release{}, false
}

func coveredPartIDs(seasonParts []domain.Part, parsed domain.ParsedRelease) []domain.ID {
	parts := coveredWantedParts(parsed, seasonParts, nil)
	ids := make([]domain.ID, len(parts))
	for i, p := range parts {
		ids[i] = p.Id
	}
	return ids
}

func coveredWantedParts(parsed domain.ParsedRelease, parts []domain.Part, seasonOf map[domain.ID]int) []domain.Part {
	coveredSeasons := make(map[int]bool)
	if parsed.Complete {
		for _, p := range parts {
			if p.GroupId == nil {
				continue
			}
			if seasonOf != nil {
				if season, ok := seasonOf[*p.GroupId]; ok {
					coveredSeasons[season] = true
				}
				continue
			}
			coveredSeasons[0] = true
		}
	} else if len(parsed.Seasons) > 0 {
		for _, season := range parsed.Seasons {
			coveredSeasons[season] = true
		}
	} else if parsed.Season != nil {
		coveredSeasons[*parsed.Season] = true
	} else {
		coveredSeasons[0] = true
	}

	var epSet map[int]bool
	if len(parsed.Episodes) > 0 {
		epSet = make(map[int]bool, len(parsed.Episodes))
		for _, ep := range parsed.Episodes {
			epSet[ep] = true
		}
	}

	var out []domain.Part
	for _, p := range parts {
		if seasonOf != nil {
			if p.GroupId == nil {
				continue
			}
			season, ok := seasonOf[*p.GroupId]
			if !ok || !coveredSeasons[season] {
				continue
			}
		}
		if epSet != nil {
			if p.GroupOrder == nil || !epSet[*p.GroupOrder] {
				continue
			}
		}
		out = append(out, p)
	}
	return out
}

func rankByPartCoverage(scored []ScoredRelease, remaining []domain.Part, seasonOf map[domain.ID]int) []ScoredRelease {
	if len(scored) == 0 {
		return scored
	}
	type entry struct {
		idx      int
		coverage int
	}
	entries := make([]entry, 0, len(scored))
	for i, s := range scored {
		cov := len(coveredWantedParts(s.Parsed, remaining, seasonOf))
		if cov == 0 {
			continue
		}
		entries = append(entries, entry{idx: i, coverage: cov})
	}
	sort.SliceStable(entries, func(a, b int) bool {
		if entries[a].coverage != entries[b].coverage {
			return entries[a].coverage > entries[b].coverage
		}
		ra := scored[entries[a].idx].Parsed.Quality.Rank()
		rb := scored[entries[b].idx].Parsed.Quality.Rank()
		if ra != rb {
			return ra > rb
		}
		return scored[entries[a].idx].Release.Seeders > scored[entries[b].idx].Release.Seeders
	})
	out := make([]ScoredRelease, len(entries))
	for i, e := range entries {
		out[i] = scored[e.idx]
	}
	return out
}
