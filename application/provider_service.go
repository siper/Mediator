package application

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"stersh.ru/mediator/domain"
)

type ProviderService struct {
	source       domain.MetadataSource
	mediaService *MediaService
	partService  *PartService
	groupRepo    domain.PartGroupRepository
	partRepo     domain.PartRepository
	coverStore   domain.CoverStore
	qualityRepo  domain.QualityProfileRepository
}

func NewProviderService(
	source domain.MetadataSource,
	mediaSvc *MediaService,
	partSvc *PartService,
	groupRepo domain.PartGroupRepository,
	partRepo domain.PartRepository,
	coverStore domain.CoverStore,
	qualityRepo domain.QualityProfileRepository,
) *ProviderService {
	return &ProviderService{
		source:       source,
		mediaService: mediaSvc,
		partService:  partSvc,
		groupRepo:    groupRepo,
		partRepo:     partRepo,
		coverStore:   coverStore,
		qualityRepo:  qualityRepo,
	}
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (s *ProviderService) providers() map[string]domain.MediaProvider {
	m := make(map[string]domain.MediaProvider)
	for _, p := range s.source.Active() {
		m[p.Name()] = p
	}
	return m
}

func (s *ProviderService) Search(query string, mediaType *domain.MediaType, providerName string, page, limit int) (*domain.SearchPage, error) {
	page, limit = clampPageLimit(page, limit)
	providers := s.providers()

	if providerName != "" {
		p, ok := providers[providerName]
		if !ok {
			return nil, domain.ErrProviderNotSupported
		}
		res, more, err := p.Search(query, mediaType, page, limit)
		if err != nil {
			return nil, err
		}
		return &domain.SearchPage{
			Results: dedupResults(res),
			Page:    page,
			Limit:   limit,
			HasMore: more,
		}, nil
	}

	if len(providers) == 0 {
		return nil, domain.ErrNoProviders
	}

	var (
		mu      sync.Mutex
		results []domain.SearchResult
		wg      sync.WaitGroup
		errs    []error
		hasMore bool
	)

	for _, p := range providers {
		wg.Add(1)
		go func(provider domain.MediaProvider) {
			defer wg.Done()
			r, more, err := provider.Search(query, mediaType, page, limit)
			mu.Lock()
			if err != nil {
				slog.Warn("provider search failed", "provider", provider.Name(), "err", err)
				errs = append(errs, err)
			} else {
				if more {
					hasMore = true
				}
				results = append(results, r...)
			}
			mu.Unlock()
		}(p)
	}
	wg.Wait()

	if len(results) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("all providers failed: %v", errs)
	}
	return &domain.SearchPage{
		Results: dedupResults(results),
		Page:    page,
		Limit:   limit,
		HasMore: hasMore,
	}, nil
}

func (s *ProviderService) LookupMedia(ctx context.Context, providerName string, externalID string, mediaType domain.MediaType) (*domain.SearchResult, error) {
	p, ok := s.providers()[providerName]
	if !ok {
		return nil, domain.ErrProviderNotSupported
	}
	pm, err := p.GetMedia(externalID, mediaType)
	if err != nil {
		return nil, err
	}
	return &domain.SearchResult{
		ProviderName: providerName,
		ExternalID:   externalID,
		Title:        pm.Title,
		CoverURL:     pm.CoverURL,
		Overview:     pm.Overview,
		MediaType:    mediaType,
	}, nil
}

func (s *ProviderService) storeCover(ctx context.Context, sourceURL string) *string {
	if sourceURL == "" {
		return nil
	}
	detached := context.WithoutCancel(ctx)
	served, err := s.coverStore.Store(detached, sourceURL)
	if err != nil || served == "" {
		slog.Warn("cover store failed, using source url", "url", sourceURL, "err", err)
		return strPtrOrNil(sourceURL)
	}
	return strPtrOrNil(served)
}

func (s *ProviderService) Import(ctx context.Context, providerName string, externalID string, mediaType domain.MediaType, libraryID domain.ID, profileID *domain.ID, folder string, coverURL string) (*domain.Media, error) {
	if libraryID == 0 {
		return nil, domain.ErrLibraryRequired
	}
	var resolvedProfile *domain.ID
	if profileID != nil {
		qp, err := s.qualityRepo.GetById(*profileID)
		if err != nil {
			return nil, domain.ErrProfileNotFound
		}
		if qp.Type != mediaType {
			return nil, domain.ErrProfileTypeMismatch
		}
		resolvedProfile = profileID
	}
	p, ok := s.providers()[providerName]
	if !ok {
		return nil, domain.ErrProviderNotSupported
	}

	var coverPtr *string
	if coverURL != "" {
		coverPtr = s.storeCover(ctx, coverURL)
	}

	pm, err := p.GetMedia(externalID, mediaType)
	if err != nil {
		return nil, fmt.Errorf("failed to get media from provider: %w", err)
	}

	if pm.CoverURL != "" {
		if coverPtr == nil || (coverURL != "" && coverURL != pm.CoverURL) {
			coverPtr = s.storeCover(ctx, pm.CoverURL)
		}
	}
	var folderPtr *string
	if folder != "" {
		folderPtr = &folder
	}
	media, err := s.mediaService.Create(pm.Title, pm.OriginalName, folderPtr, coverPtr, mediaType, &libraryID, providerName, externalID, resolvedProfile)
	if err != nil {
		return nil, fmt.Errorf("failed to create media: %w", err)
	}
	if pm.Status != "" || pm.LastModified != "" {
		_ = s.mediaService.UpdateProviderMeta(media.Id, pm.Status, pm.LastModified)
	}

	groupByName := make(map[string]domain.ID)
	for _, g := range pm.Groups {
		group := &domain.PartGroup{
			Name:    g.Name,
			Order:   g.Order,
			MediaId: media.Id,
		}
		if err := s.groupRepo.Add(group); err != nil {
			return nil, fmt.Errorf("failed to create part group: %w", err)
		}
		groupByName[g.Name] = group.Id
	}

	for _, part := range s.partsToCreate(mediaType, pm.Parts) {
		p := &domain.Part{
			Name:       part.Name,
			GroupOrder: part.GroupOrder,
			MediaId:    media.Id,
			Monitored:  true,
		}
		if part.GroupName != nil {
			if gid, ok := groupByName[*part.GroupName]; ok {
				p.GroupId = &gid
			}
		}
		if err := s.partService.Add(p); err != nil {
			return nil, fmt.Errorf("failed to create part: %w", err)
		}
	}

	return media, nil
}

func (s *ProviderService) partsToCreate(mediaType domain.MediaType, parts []domain.ProviderPart) []domain.ProviderPart {
	if (mediaType == domain.MediaTypeMovie || mediaType == domain.MediaTypeBook || mediaType == domain.MediaTypeMusicAlbum) && len(parts) == 0 {
		return []domain.ProviderPart{{}}
	}
	return parts
}

func (s *ProviderService) RefreshMedia(ctx context.Context, id domain.ID) error {
	media, err := s.mediaService.GetByID(id)
	if err != nil {
		return err
	}

	if media.ProviderID == "" {
		return fmt.Errorf("media has no provider")
	}

	p, ok := s.providers()[media.ProviderID]
	if !ok {
		return domain.ErrProviderNotSupported
	}

	pm, err := p.GetMedia(media.ExternalID, media.Type)
	if err != nil {
		return fmt.Errorf("failed to get media from provider: %w", err)
	}

	newCover := s.storeCover(ctx, pm.CoverURL)
	if newCover == nil {
		newCover = media.Cover
	}
	if err := s.mediaService.Update(media.Id, pm.Title, pm.OriginalName, newCover); err != nil {
		return fmt.Errorf("failed to update media: %w", err)
	}
	if pm.Status != "" {
		_ = s.mediaService.UpdateProviderMeta(media.Id, pm.Status, media.LastModified)
	}

	existingGroups, err := s.groupRepo.GetByMediaID(media.Id)
	if err != nil {
		return err
	}
	groupByName := make(map[string]domain.ID, len(existingGroups))
	for _, g := range existingGroups {
		groupByName[g.Name] = g.Id
	}
	for _, g := range pm.Groups {
		if _, ok := groupByName[g.Name]; ok {
			continue
		}
		ng := &domain.PartGroup{Name: g.Name, Order: g.Order, MediaId: media.Id}
		if err := s.groupRepo.Add(ng); err != nil {
			return err
		}
		groupByName[g.Name] = ng.Id
	}

	existingParts, err := s.partRepo.GetByMediaId(media.Id)
	if err != nil {
		return err
	}
	existingByKey := make(map[partKey]domain.Part, len(existingParts))
	for _, pt := range existingParts {
		existingByKey[partKeyOf(pt)] = pt
	}

	for _, pp := range pm.Parts {
		pp := pp
		key := providerPartKeyOf(pp)
		if ex, ok := existingByKey[key]; ok {
			ex.Name = pp.Name
			ex.GroupOrder = pp.GroupOrder
			if pp.GroupName != nil {
				if gid, ok := groupByName[*pp.GroupName]; ok {
					ex.GroupId = &gid
				}
			}
			if err := s.partRepo.Update(&ex); err != nil {
				return err
			}
			continue
		}
		np := &domain.Part{
			Name:       pp.Name,
			GroupOrder: pp.GroupOrder,
			MediaId:    media.Id,
			Monitored:  true,
		}
		if pp.GroupName != nil {
			if gid, ok := groupByName[*pp.GroupName]; ok {
				np.GroupId = &gid
			}
		}
		if err := s.partRepo.Add(np); err != nil {
			return err
		}
	}

	return nil
}

type partKey struct {
	name  string
	order int
}

func partKeyOf(p domain.Part) partKey {
	k := partKey{}
	if p.Name != nil {
		k.name = *p.Name
	}
	if p.GroupOrder != nil {
		k.order = *p.GroupOrder
	}
	return k
}

func providerPartKeyOf(p domain.ProviderPart) partKey {
	k := partKey{}
	if p.Name != nil {
		k.name = *p.Name
	}
	if p.GroupOrder != nil {
		k.order = *p.GroupOrder
	}
	return k
}

func (s *ProviderService) CheckUpdates(ctx context.Context) (int, error) {
	providers := s.providers()
	probes := make(map[string]domain.VersionedReader)
	for name, p := range providers {
		if vr, ok := p.(domain.VersionedReader); ok {
			probes[name] = vr
		}
	}
	if len(probes) == 0 {
		return 0, nil
	}

	page, limit := 1, 100
	var marked int
	for {
		items, err := s.mediaService.GetPaged(page, limit, nil)
		if err != nil {
			return marked, err
		}
		if len(items) == 0 {
			return marked, nil
		}
		for _, media := range items {
			if media.Status != domain.MediaStatusContinuing || media.ProviderID == "" {
				continue
			}
			probe, ok := probes[media.ProviderID]
			if !ok {
				continue
			}
			remote, err := probe.FetchLastModified(ctx, media.ExternalID, media.Type)
			if err != nil {
				slog.Warn("check-content: fetch version failed", "id", media.Id, "err", err)
				continue
			}
			if remote == "" || remote == media.LastModified {
				continue
			}
			if err := s.markPartsWanted(media.Id, remote); err != nil {
				slog.Warn("check-content: mark wanted failed", "id", media.Id, "err", err)
				continue
			}
			marked++
		}
		if len(items) < limit {
			return marked, nil
		}
		page++
	}
}

func (s *ProviderService) markPartsWanted(mediaID domain.ID, remoteVersion string) error {
	parts, err := s.partRepo.GetByMediaId(mediaID)
	if err != nil {
		return err
	}
	for i := range parts {
		p := parts[i]
		if p.Path == nil {
			continue
		}
		p.Path = nil
		p.Monitored = true
		if err := s.partRepo.Update(&p); err != nil {
			return err
		}
	}
	return s.mediaService.UpdateProviderMeta(mediaID, domain.MediaStatusContinuing, remoteVersion)
}
