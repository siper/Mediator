package application

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"stersh.ru/mediator/domain"
)

type ScoredRelease struct {
	Release domain.Release
	Parsed  domain.ParsedRelease
}

type SeriesTarget struct {
	Season  int
	Episode *int
}

type ReleaseService struct {
	source domain.IndexerSource
	parser *ReleaseParser
}

func NewReleaseService(source domain.IndexerSource, parser *ReleaseParser) *ReleaseService {
	return &ReleaseService{source: source, parser: parser}
}

func (s *ReleaseService) Search(ctx context.Context, query string, mediaType domain.MediaType, profile *domain.QualityProfile, target *SeriesTarget, matchTitles ...string) ([]ScoredRelease, error) {
	type indexerResult struct {
		releases []domain.Release
		err      error
	}

	indexers := s.source.Active()
	cats := categoriesFor(mediaType)
	titles := matchTitles
	if len(titles) == 0 {
		titles = []string{query}
	}

	results := make([]indexerResult, len(indexers))
	var wg sync.WaitGroup
	for i, ix := range indexers {
		wg.Add(1)
		go func(idx int, indexer domain.ReleaseIndexer) {
			defer wg.Done()
			rels, err := indexer.Search(ctx, buildQuery(query, target), cats)
			results[idx] = indexerResult{releases: rels, err: err}
		}(i, ix)
	}
	wg.Wait()

	allowed := allowedSet(profile)
	scored := make([]ScoredRelease, 0)
	var errs []error
	for _, res := range results {
		if res.err != nil {
			errs = append(errs, res.err)
			continue
		}
		for _, rel := range res.releases {
			parsed, err := s.parser.Parse(rel.Title, mediaType)
			if err != nil {
				continue
			}
			if !titlesMatchAny(titles, rel.Title) {
				continue
			}
			if allowed != nil && !allowed[parsed.Quality] {
				continue
			}
			if target != nil && !matchesTarget(parsed, *target) {
				continue
			}
			scored = append(scored, ScoredRelease{Release: rel, Parsed: parsed})
		}
	}

	if len(scored) == 0 && len(errs) == len(indexers) && len(indexers) > 0 {
		return nil, fmt.Errorf("all indexers failed: %v", errs)
	}

	sort.SliceStable(scored, func(i, j int) bool {
		ri, rj := scored[i].Parsed.Quality.Rank(), scored[j].Parsed.Quality.Rank()
		if ri != rj {
			return ri > rj
		}
		return scored[i].Release.Seeders > scored[j].Release.Seeders
	})

	return scored, nil
}

func buildQuery(base string, target *SeriesTarget) string {
	if target == nil {
		return base
	}
	if target.Episode == nil {
		return fmt.Sprintf("%s S%02d", base, target.Season)
	}
	return fmt.Sprintf("%s S%02dE%02d", base, target.Season, *target.Episode)
}

func matchesTarget(parsed domain.ParsedRelease, target SeriesTarget) bool {
	if !coversSeason(parsed, target.Season) {
		return false
	}
	if target.Episode == nil {
		return true
	}
	if len(parsed.Episodes) == 0 {
		return true
	}
	for _, ep := range parsed.Episodes {
		if ep == *target.Episode {
			return true
		}
	}
	return false
}

func coversSeason(parsed domain.ParsedRelease, season int) bool {
	if parsed.Complete {
		return true
	}
	if len(parsed.Seasons) > 0 {
		for _, s := range parsed.Seasons {
			if s == season {
				return true
			}
		}
		return false
	}
	if parsed.Season != nil {
		return *parsed.Season == season
	}
	return false
}

func allowedSet(profile *domain.QualityProfile) map[domain.Quality]bool {
	if profile == nil {
		return nil
	}
	out := make(map[domain.Quality]bool, len(profile.Allowed))
	for _, q := range profile.Allowed {
		out[q] = true
	}
	return out
}

func categoriesFor(t domain.MediaType) []int {
	switch t {
	case domain.MediaTypeMovie:
		return []int{2000, 2030, 2040, 2045}
	case domain.MediaTypeSeries:
		return []int{5000, 5030, 5040, 5050}
	case domain.MediaTypeBook:
		return []int{7000, 7020, 7030}
	case domain.MediaTypeMusicAlbum:
		return []int{3000, 3010, 3040}
	default:
		return nil
	}
}

func mediaReleaseQuery(m domain.Media) string {
	if m.OriginalName != "" {
		return m.OriginalName
	}
	return m.Name
}

func mediaMatchTitles(m domain.Media) []string {
	out := []string{m.Name}
	if m.OriginalName != "" && m.OriginalName != m.Name {
		out = append(out, m.OriginalName)
	}
	return out
}

func titlesMatchAny(titles []string, releaseTitle string) bool {
	for _, title := range titles {
		if title == "" {
			continue
		}
		if tier, _ := matchScore(title, releaseTitle); tier != tierDrop {
			return true
		}
	}
	return false
}

func releaseIdentity(rel domain.Release) string {
	if rel.MagnetURI != "" {
		return "magnet:" + rel.MagnetURI
	}
	if rel.DownloadURL != "" {
		return "url:" + rel.DownloadURL
	}
	return "title:" + rel.Title
}
