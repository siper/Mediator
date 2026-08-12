package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"stersh.ru/mediator/domain"
)

type Importer interface {
	Import(ctx context.Context, providerName string, externalID string, mediaType domain.MediaType, libraryID domain.ID, profileID *domain.ID, folder string, coverURL string) (*domain.Media, error)
}

type MediaLookuper interface {
	LookupMedia(ctx context.Context, provider string, externalID string, mediaType domain.MediaType) (*domain.SearchResult, error)
}

type MediaRequestService struct {
	repo       domain.MediaRequestRepository
	media      domain.MediaRepository
	importer   Importer
	setting    domain.SettingRepository
	lookuper   MediaLookuper
	coverStore domain.CoverStore
}

func NewMediaRequestService(
	repo domain.MediaRequestRepository,
	mediaRepo domain.MediaRepository,
	importer Importer,
	setting domain.SettingRepository,
	lookuper MediaLookuper,
	coverStore domain.CoverStore,
) *MediaRequestService {
	return &MediaRequestService{
		repo:       repo,
		media:      mediaRepo,
		importer:   importer,
		setting:    setting,
		lookuper:   lookuper,
		coverStore: coverStore,
	}
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func (s *MediaRequestService) boolSetting(key string, def bool) bool {
	v, err := s.setting.Get(key)
	if err != nil {
		if errors.Is(err, domain.ErrSettingNotFound) {
			return def
		}
		return def
	}
	return v == "1" || v == "true"
}

func (s *MediaRequestService) Create(ctx context.Context, userID domain.ID, provider string, externalID string, title string, cover string, mediaType domain.MediaType, libraryID domain.ID, profileID *domain.ID, folder string) (*domain.MediaRequest, error) {
	if !s.boolSetting(domain.SettingRequestsEnabled, true) {
		return nil, domain.ErrRequestsDisabled
	}
	if provider == "" {
		return nil, domain.ErrEmptyName
	}
	if externalID == "" {
		return nil, fmt.Errorf("external id cannot be empty")
	}
	if libraryID == 0 {
		return nil, domain.ErrLibraryRequired
	}

	media, _ := s.media.GetByProviderExternal(provider, externalID)
	if media != nil {
		return nil, domain.ErrMediaAlreadyExists
	}

	exists, err := s.repo.ExistsByProviderExternal(provider, externalID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domain.ErrDuplicateRequest
	}

	title, cover = s.resolveMetadata(ctx, provider, externalID, mediaType, title, cover)
	storedCover := s.storeCover(ctx, cover)

	req := &domain.MediaRequest{
		UserId:           userID,
		Provider:         provider,
		ExternalID:       externalID,
		Title:            title,
		Cover:         storedCover,
		Type:             mediaType,
		LibraryID:        libraryID,
		QualityProfileID: profileID,
		Status:           domain.MediaRequestPending,
		CreatedAt:        nowRFC3339(),
	}
	if folder != "" {
		req.Folder = &folder
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Add(req); err != nil {
		return nil, err
	}

	if s.boolSetting(domain.SettingAutoApproveRequests, false) {
		media, err := s.importer.Import(ctx, provider, externalID, mediaType, libraryID, profileID, folder, storedCover)
		if err != nil {
			return req, fmt.Errorf("failed to import media: %w", err)
		}
		now := nowRFC3339()
		req.Status = domain.MediaRequestApproved
		req.ApprovedAt = &now
		req.MediaID = &media.Id
		if err := s.repo.Update(req); err != nil {
			return req, nil
		}
	}

	return req, nil
}

func (s *MediaRequestService) Approve(ctx context.Context, requestID domain.ID, adminUserID domain.ID) (*domain.MediaRequest, error) {
	req, err := s.repo.GetByID(requestID)
	if err != nil {
		return nil, err
	}
	if !req.CanEdit() {
		return nil, domain.ErrRequestNotPending
	}

	coverURL := req.Cover
	media, err := s.importer.Import(ctx, req.Provider, req.ExternalID, req.Type, req.LibraryID, req.QualityProfileID, folderStr(req.Folder), coverURL)
	if err != nil {
		return req, fmt.Errorf("failed to import media: %w", err)
	}

	now := nowRFC3339()
	req.Status = domain.MediaRequestApproved
	req.ApprovedAt = &now
	req.ApprovedBy = &adminUserID
	req.MediaID = &media.Id
	if err := s.repo.Update(req); err != nil {
		return req, nil
	}
	return req, nil
}

func (s *MediaRequestService) Reject(ctx context.Context, requestID domain.ID, adminUserID domain.ID, notes *string) (*domain.MediaRequest, error) {
	req, err := s.repo.GetByID(requestID)
	if err != nil {
		return nil, err
	}
	if !req.CanEdit() {
		return nil, domain.ErrRequestNotPending
	}
	now := nowRFC3339()
	req.Status = domain.MediaRequestRejected
	req.RejectedAt = &now
	req.RejectedBy = &adminUserID
	req.Notes = notes
	if err := s.repo.Update(req); err != nil {
		return req, nil
	}
	return req, nil
}

func (s *MediaRequestService) Cancel(ctx context.Context, requestID domain.ID, userID domain.ID) (*domain.MediaRequest, error) {
	req, err := s.repo.GetByID(requestID)
	if err != nil {
		return nil, err
	}
	if !req.CanEdit() {
		return nil, domain.ErrRequestNotPending
	}
	if req.UserId != userID {
		return nil, domain.ErrRequestNotAuthorized
	}
	now := nowRFC3339()
	req.Status = domain.MediaRequestCanceled
	req.CanceledAt = &now
	if err := s.repo.Update(req); err != nil {
		return req, nil
	}
	return req, nil
}

func (s *MediaRequestService) ListAll(page int, limit int) ([]domain.MediaRequest, error) {
	return s.repo.ListAll(page, limit)
}

func (s *MediaRequestService) ListByUser(userID domain.ID, page int, limit int) ([]domain.MediaRequest, error) {
	return s.repo.ListByUser(userID, page, limit)
}

func (s *MediaRequestService) GetByID(id domain.ID) (*domain.MediaRequest, error) {
	return s.repo.GetByID(id)
}

func folderStr(f *string) string {
	if f == nil {
		return ""
	}
	return *f
}

func (s *MediaRequestService) resolveMetadata(ctx context.Context, provider string, externalID string, mediaType domain.MediaType, title string, cover string) (string, string) {
	if title != "" && cover != "" {
		return title, cover
	}
	sr, err := s.lookuper.LookupMedia(ctx, provider, externalID, mediaType)
	if err != nil {
		return title, cover
	}
	if title == "" {
		title = sr.Title
	}
	if cover == "" {
		cover = sr.CoverURL
	}
	return title, cover
}

func (s *MediaRequestService) storeCover(ctx context.Context, cover string) string {
	if cover == "" {
		return ""
	}
	served, err := s.coverStore.Store(ctx, cover)
	if err != nil || served == "" {
		return cover
	}
	return served
}
