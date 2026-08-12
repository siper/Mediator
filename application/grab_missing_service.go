package application

import (
	"context"
	"log/slog"

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
	query := mediaReleaseQuery(media)
	titles := mediaMatchTitles(media)
	for _, p := range parts {
		if s.grabSvc.HasActiveGrab(media.Id, p.Id) {
			continue
		}
		scored, err := s.releaseSvc.Search(ctx, query, mt, profile, nil, titles...)
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
	scored, err := s.releaseSvc.Search(ctx, mediaReleaseQuery(media), domain.MediaTypeMusicAlbum, profile, nil, mediaMatchTitles(media)...)
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

	bySeason := make(map[int][]domain.Part)
	for _, p := range parts {
		if p.GroupId == nil {
			continue
		}
		season, ok := seasonOf[*p.GroupId]
		if !ok {
			continue
		}
		bySeason[season] = append(bySeason[season], p)
	}

	seen := make(map[string]bool)
	query := mediaReleaseQuery(media)
	titles := mediaMatchTitles(media)
	var grabbed int
	for season, seasonParts := range bySeason {
		anyActive := false
		wantedEps := make([]int, 0, len(seasonParts))
		for _, p := range seasonParts {
			if s.grabSvc.HasActiveGrab(media.Id, p.Id) {
				anyActive = true
			}
			if p.GroupOrder != nil {
				wantedEps = append(wantedEps, *p.GroupOrder)
			}
		}
		if anyActive {
			continue
		}

		scored, err := s.releaseSvc.Search(ctx, query, domain.MediaTypeSeries, profile, &SeriesTarget{Season: season}, titles...)
		if err != nil {
			slog.Warn("grab-missing: series search failed", "id", media.Id, "season", season, "err", err)
			continue
		}
		ranked := rankByCoverage(scored, wantedEps)
		grabbedOne := false
		for _, candidate := range ranked {
			if candidate.Release.Seeders <= 0 {
				continue
			}
			key := releaseIdentity(candidate.Release)
			if seen[key] || s.grabSvc.HasActiveReleaseTitle(media.Id, candidate.Release.Title) {
				continue
			}
			partIds := coveredPartIDs(seasonParts, candidate.Parsed)
			if len(partIds) == 0 {
				continue
			}
			rel := candidate.Release
			if _, err := s.grabSvc.Grab(ctx, domain.GrabTarget{
				MediaID: media.Id,
				PartIds: partIds,
				Release: &rel,
			}); err != nil {
				slog.Warn("grab-missing: series grab failed", "id", media.Id, "season", season, "err", err)
				continue
			}
			seen[key] = true
			grabbed++
			grabbedOne = true
			break
		}
		if !grabbedOne {
			slog.Debug("grab-missing: no releases for season", "id", media.Id, "season", season)
		}
	}
	return grabbed, nil
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
	if len(parsed.Episodes) == 0 {
		ids := make([]domain.ID, 0, len(seasonParts))
		for _, p := range seasonParts {
			ids = append(ids, p.Id)
		}
		return ids
	}
	wanted := make(map[int]bool, len(parsed.Episodes))
	for _, ep := range parsed.Episodes {
		wanted[ep] = true
	}
	ids := make([]domain.ID, 0, len(seasonParts))
	for _, p := range seasonParts {
		if p.GroupOrder != nil && wanted[*p.GroupOrder] {
			ids = append(ids, p.Id)
		}
	}
	return ids
}
