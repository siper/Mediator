package application

import (
	"context"
	"log/slog"
	"sort"

	"stersh.ru/mediator/domain"
)

func NewRefreshMetadataJob(providerSvc *ProviderService, mediaSvc *MediaService) JobHandler {
	return func(ctx context.Context) error {
		page, limit := 1, 100
		var refreshed, failed int
		for {
			items, err := mediaSvc.GetPaged(page, limit, nil)
			if err != nil {
				return err
			}
			if len(items) == 0 {
				break
			}
			for _, media := range items {
				if media.ProviderID == "" {
					continue
				}
				if err := providerSvc.RefreshMedia(ctx, media.Id); err != nil {
					failed++
					slog.Warn("refresh-metadata: failed", "id", media.Id, "err", err)
					continue
				}
				refreshed++
			}
			if len(items) < limit {
				break
			}
			page++
		}
		slog.Info("refresh-metadata: done", "refreshed", refreshed, "failed", failed)
		return nil
	}
}

func NewGrabMissingJob(
	grabSvc *GrabService,
	partSvc *PartService,
	mediaSvc *MediaService,
	releaseSvc *ReleaseService,
	qualityRepo domain.QualityProfileRepository,
	groupRepo domain.PartGroupRepository,
) JobHandler {
	return NewGrabMissingService(grabSvc, partSvc, mediaSvc, releaseSvc, qualityRepo, groupRepo).Run
}

func NewRssJob(
	grabSvc *GrabService,
	mediaRepo domain.MediaRepository,
	partRepo domain.PartRepository,
	groupRepo domain.PartGroupRepository,
	qualityRepo domain.QualityProfileRepository,
	source domain.IndexerSource,
) JobHandler {
	return NewRssService(grabSvc, mediaRepo, partRepo, groupRepo, qualityRepo, NewReleaseParser(), source).Run
}

func NewCheckContentJob(providerSvc *ProviderService) JobHandler {
	return func(ctx context.Context) error {
		marked, err := providerSvc.CheckUpdates(ctx)
		if err != nil {
			return err
		}
		if marked > 0 {
			slog.Info("check-content: content changed, marked for re-grab", "count", marked)
		}
		return nil
	}
}

func rankByCoverage(scored []ScoredRelease, wantedEps []int) []ScoredRelease {
	if len(scored) == 0 {
		return scored
	}
	wantedSet := make(map[int]bool, len(wantedEps))
	for _, e := range wantedEps {
		wantedSet[e] = true
	}
	totalWanted := len(wantedEps)

	type entry struct {
		idx      int
		coverage int
	}
	entries := make([]entry, len(scored))
	for i, s := range scored {
		cov := 0
		if len(s.Parsed.Episodes) == 0 {
			cov = totalWanted
		} else {
			for _, ep := range s.Parsed.Episodes {
				if wantedSet[ep] {
					cov++
				}
			}
		}
		entries[i] = entry{idx: i, coverage: cov}
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
	out := make([]ScoredRelease, 0, len(entries))
	for _, e := range entries {
		if e.coverage == 0 {
			continue
		}
		out = append(out, scored[e.idx])
	}
	return out
}
