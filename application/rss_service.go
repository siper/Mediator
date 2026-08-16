package application

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"stersh.ru/mediator/domain"
)

const rssDefaultInterval = "15m"
const rssSeenLimit = 10000

type RssService struct {
	grabSvc     *GrabService
	mediaRepo   domain.MediaRepository
	partRepo    domain.PartRepository
	groupRepo   domain.PartGroupRepository
	qualityRepo domain.QualityProfileRepository
	parser      *ReleaseParser
	source      domain.IndexerSource

	mu   sync.Mutex
	seen map[string]bool
}

func NewRssService(
	grabSvc *GrabService,
	mediaRepo domain.MediaRepository,
	partRepo domain.PartRepository,
	groupRepo domain.PartGroupRepository,
	qualityRepo domain.QualityProfileRepository,
	parser *ReleaseParser,
	source domain.IndexerSource,
) *RssService {
	return &RssService{
		grabSvc:     grabSvc,
		mediaRepo:   mediaRepo,
		partRepo:    partRepo,
		groupRepo:   groupRepo,
		qualityRepo: qualityRepo,
		parser:      parser,
		source:      source,
		seen:        make(map[string]bool),
	}
}

func (s *RssService) Run(ctx context.Context) error {
	indexers := s.source.Active()
	if len(indexers) == 0 {
		slog.Debug("rss: no active indexers")
		return nil
	}

	rels := fetchRss(ctx, indexers, categoriesFor(domain.MediaTypeSeries))
	if len(rels) == 0 {
		return nil
	}

	series, err := s.collectSeries(ctx)
	if err != nil {
		return err
	}
	if len(series) == 0 {
		return nil
	}

	grabbed := s.process(ctx, rels, series)
	if grabbed > 0 {
		slog.Info("rss: submitted grabs", "count", grabbed)
	}
	return nil
}

func (s *RssService) collectSeries(ctx context.Context) ([]rssSeries, error) {
	mt := domain.MediaTypeSeries
	page, limit := 1, 100
	var out []rssSeries
	for {
		items, err := s.mediaRepo.GetPaged(page, limit, &mt)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			break
		}
		for _, m := range items {
			si, ok := s.buildSeriesInfo(m)
			if ok {
				out = append(out, si)
			}
		}
		if len(items) < limit {
			break
		}
		page++
	}
	return out, nil
}

func (s *RssService) buildSeriesInfo(media domain.Media) (rssSeries, bool) {
	parts, err := s.partRepo.GetByMediaId(media.Id)
	if err != nil {
		return rssSeries{}, false
	}

	groups, err := s.groupRepo.GetByMediaID(media.Id)
	if err != nil {
		return rssSeries{}, false
	}
	seasonOf := make(map[domain.ID]int, len(groups))
	for _, g := range groups {
		seasonOf[g.Id] = g.Order
	}

	wanted := make(map[int][]domain.Part)
	for _, p := range parts {
		if !p.Monitored || p.Path != nil {
			continue
		}
		if p.GroupId == nil {
			continue
		}
		season, ok := seasonOf[*p.GroupId]
		if !ok {
			continue
		}
		wanted[season] = append(wanted[season], p)
	}
	if len(wanted) == 0 {
		return rssSeries{}, false
	}

	var allowed map[domain.Quality]bool
	if media.QualityProfileID != nil {
		if profile, err := s.qualityRepo.GetById(*media.QualityProfileID); err == nil {
			allowed = allowedSet(profile)
		}
	}

	return rssSeries{
		media:          media,
		wanted:         wanted,
		seasonOf:       seasonOf,
		qualityAllowed: allowed,
	}, true
}

func (s *RssService) process(ctx context.Context, releases []domain.Release, series []rssSeries) int {
	grabbed := 0
	for _, rel := range releases {
		if rel.InfoHash != "" {
			if s.markSeen(rel.InfoHash) {
				continue
			}
		}

		parsed, err := s.parser.Parse(rel.Title, domain.MediaTypeSeries)
		if err != nil || parsed.Season == nil {
			continue
		}

		idx := s.bestMatch(parsed, rel.Title, series)
		if idx < 0 {
			continue
		}
		si := &series[idx]

		if si.qualityAllowed != nil && !si.qualityAllowed[parsed.Quality] {
			continue
		}

		targets := findWantedParts(parsed, si)
		if len(targets) == 0 {
			continue
		}

		if s.hasActiveGrab(si.media.Id, targets) {
			continue
		}

		partIds := make([]domain.ID, len(targets))
		for i, p := range targets {
			partIds[i] = p.Id
		}
		if _, err := s.grabSvc.Grab(ctx, domain.GrabTarget{
			MediaID: si.media.Id,
			PartIds: partIds,
			Release: &rel,
		}); err != nil {
			slog.Warn("rss: grab failed", "media_id", si.media.Id, "title", rel.Title, "err", err)
			continue
		}
		grabbed++
	}
	return grabbed
}

func (s *RssService) bestMatch(parsed domain.ParsedRelease, title string, series []rssSeries) int {
	best := -1
	bestTier := tierDrop
	bestScore := 0.0
	for i := range series {
		for _, name := range MediaMatchTitles(series[i].media) {
			tier, score := matchScore(name, title)
			if tier == tierDrop {
				continue
			}
			if best < 0 || tier < bestTier || (tier == bestTier && score > bestScore) {
				best = i
				bestTier = tier
				bestScore = score
			}
		}
	}
	return best
}

func (s *RssService) hasActiveGrab(mediaID domain.ID, parts []domain.Part) bool {
	for _, p := range parts {
		if s.grabSvc.HasActiveGrab(mediaID, p.Id) {
			return true
		}
	}
	return false
}

func (s *RssService) markSeen(hash string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.seen) > rssSeenLimit {
		s.seen = make(map[string]bool)
	}
	if s.seen[hash] {
		return true
	}
	s.seen[hash] = true
	return false
}

type rssSeries struct {
	media          domain.Media
	wanted         map[int][]domain.Part
	seasonOf       map[domain.ID]int
	qualityAllowed map[domain.Quality]bool
}

func findWantedParts(parsed domain.ParsedRelease, si *rssSeries) []domain.Part {
	if parsed.Season == nil {
		return nil
	}
	var parts []domain.Part
	for _, ps := range si.wanted {
		parts = append(parts, ps...)
	}
	return coveredWantedParts(parsed, parts, si.seasonOf)
}

func fetchRss(ctx context.Context, indexers []domain.ReleaseIndexer, cats []int) []domain.Release {
	type result struct {
		rels []domain.Release
		err  error
	}
	results := make([]result, len(indexers))
	var wg sync.WaitGroup
	for i, ix := range indexers {
		wg.Add(1)
		go func(idx int, indexer domain.ReleaseIndexer) {
			defer wg.Done()
			rels, err := indexer.RSS(ctx, cats, time.Time{})
			results[idx] = result{rels: rels, err: err}
		}(i, ix)
	}
	wg.Wait()

	var all []domain.Release
	for _, r := range results {
		if r.err != nil {
			slog.Warn("rss: indexer fetch failed", "err", r.err)
			continue
		}
		all = append(all, r.rels...)
	}
	return all
}
