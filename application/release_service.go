package application

import (
	"context"
	"fmt"
	"sort"
	"strings"
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
	type searchJob struct {
		releases []domain.Release
		err      error
	}

	indexers := s.source.Active()
	cats := categoriesFor(mediaType)
	titles := uniqueSearchQueries(query, matchTitles...)
	if len(titles) == 0 {
		return nil, nil
	}

	jobs := make([]searchJob, len(indexers)*len(titles))
	var wg sync.WaitGroup
	for i, ix := range indexers {
		for j, title := range titles {
			wg.Add(1)
			go func(idx int, indexer domain.ReleaseIndexer, q string) {
				defer wg.Done()
				rels, err := indexer.Search(ctx, buildQuery(q, target), cats)
				jobs[idx] = searchJob{releases: rels, err: err}
			}(i*len(titles)+j, ix, title)
		}
	}
	wg.Wait()

	allowed := allowedSet(profile)
	scored := make([]ScoredRelease, 0)
	seen := make(map[string]int)
	failed := 0
	var errs []error
	for _, job := range jobs {
		if job.err != nil {
			errs = append(errs, job.err)
			failed++
			continue
		}
		for _, rel := range job.releases {
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
			item := ScoredRelease{Release: rel, Parsed: parsed}
			id := releaseIdentity(rel)
			if prev, ok := seen[id]; ok {
				if rel.Seeders > scored[prev].Release.Seeders {
					scored[prev] = item
				}
				continue
			}
			seen[id] = len(scored)
			scored = append(scored, item)
		}
	}

	if len(scored) == 0 && failed == len(jobs) && len(jobs) > 0 {
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

func MediaMatchTitles(m domain.Media) []string {
	return uniqueSearchQueries(m.Name, m.OriginalName)
}

func uniqueSearchQueries(query string, titles ...string) []string {
	seen := make(map[string]struct{}, 1+len(titles))
	out := make([]string, 0, 1+len(titles))
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		key := strings.ToLower(s)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, s)
	}
	add(query)
	for _, title := range titles {
		add(title)
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
