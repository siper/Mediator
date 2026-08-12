package application

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

type mockMediaRequestRepo struct {
	mock.Mock
}

func (m *mockMediaRequestRepo) Add(r *domain.MediaRequest) error {
	args := m.Called(r)
	if args.Error(0) == nil {
		if r.Id == 0 {
			r.Id = 1
		}
	}
	return args.Error(0)
}

func (m *mockMediaRequestRepo) GetByID(id domain.ID) (*domain.MediaRequest, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.MediaRequest), args.Error(1)
}

func (m *mockMediaRequestRepo) ListByUser(userID domain.ID, page int, limit int) ([]domain.MediaRequest, error) {
	args := m.Called(userID, page, limit)
	return args.Get(0).([]domain.MediaRequest), args.Error(1)
}

func (m *mockMediaRequestRepo) ListAll(page int, limit int) ([]domain.MediaRequest, error) {
	args := m.Called(page, limit)
	return args.Get(0).([]domain.MediaRequest), args.Error(1)
}

func (m *mockMediaRequestRepo) Update(r *domain.MediaRequest) error {
	return m.Called(r).Error(0)
}

func (m *mockMediaRequestRepo) Remove(id domain.ID) error {
	return m.Called(id).Error(0)
}

func (m *mockMediaRequestRepo) ExistsByProviderExternal(provider string, externalID string) (bool, error) {
	args := m.Called(provider, externalID)
	return args.Bool(0), args.Error(1)
}

type mockSettingRepo struct {
	mock.Mock
}

func (m *mockSettingRepo) Get(key string) (string, error) {
	args := m.Called(key)
	return args.String(0), args.Error(1)
}

func (m *mockSettingRepo) Set(key string, value string) error {
	return m.Called(key, value).Error(0)
}

func (m *mockSettingRepo) List() ([]domain.Setting, error) {
	args := m.Called()
	return args.Get(0).([]domain.Setting), args.Error(1)
}

func (m *mockSettingRepo) AutoApproveOn() {
	m.On("Get", domain.SettingAutoApproveRequests).Return("1", nil)
}

func (m *mockSettingRepo) AutoApproveOff() {
	m.On("Get", domain.SettingAutoApproveRequests).Return("0", nil)
}

func (m *mockSettingRepo) EnabledOn() {
	m.On("Get", domain.SettingRequestsEnabled).Return("true", nil).Maybe()
}

func (m *mockSettingRepo) EnabledOff() {
	m.On("Get", domain.SettingRequestsEnabled).Return("false", nil).Maybe()
}

type mockImporter struct {
	mock.Mock
}

func (m *mockImporter) Import(ctx context.Context, providerName string, externalID string, mediaType domain.MediaType, libraryID domain.ID, profileID *domain.ID, folder string, coverURL string) (*domain.Media, error) {
	args := m.Called(ctx, providerName, externalID, mediaType, libraryID, profileID, folder, coverURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Media), args.Error(1)
}

func (m *mockImporter) LookupMedia(ctx context.Context, provider string, externalID string, mediaType domain.MediaType) (*domain.SearchResult, error) {
	args := m.Called(ctx, provider, externalID, mediaType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.SearchResult), args.Error(1)
}

type fakeCoverStore struct{}

func (fakeCoverStore) Store(ctx context.Context, sourceURL string) (string, error) {
	return sourceURL, nil
}

func newRequestService(t *testing.T) (*MediaRequestService, *mockMediaRequestRepo, *mockSettingRepo, *mockImporter, *mockMediaRepo) {
	t.Helper()
	reqRepo := new(mockMediaRequestRepo)
	settings := new(mockSettingRepo)
	settings.EnabledOn()
	importer := new(mockImporter)
	mediaRepo := new(mockMediaRepo)
	svc := NewMediaRequestService(reqRepo, mediaRepo, importer, settings, importer, fakeCoverStore{})
	return svc, reqRepo, settings, importer, mediaRepo
}

func TestMediaRequestService_Create(t *testing.T) {
	t.Run("rejects when requests disabled", func(t *testing.T) {
		reqRepo := new(mockMediaRequestRepo)
		settings := new(mockSettingRepo)
		settings.EnabledOff()
		importer := new(mockImporter)
		mediaRepo := new(mockMediaRepo)
		svc := NewMediaRequestService(reqRepo, mediaRepo, importer, settings, importer, fakeCoverStore{})

		_, err := svc.Create(context.Background(), 1, "tmdb", "1", "Matrix", "/cover.jpg", domain.MediaTypeMovie, 1, nil, "")

		assert.ErrorIs(t, err, domain.ErrRequestsDisabled)
		reqRepo.AssertNotCalled(t, "Add")
		mediaRepo.AssertNotCalled(t, "GetByProviderExternal")
	})

	t.Run("rejects empty provider", func(t *testing.T) {
		svc, reqRepo, _, _, mediaRepo := newRequestService(t)
		mediaRepo.On("GetByProviderExternal", "", "1").Return(nil, domain.ErrMediaNotFound)

		_, err := svc.Create(context.Background(), 1, "", "1", "Matrix", "/cover.jpg", domain.MediaTypeMovie, 1, nil, "")

		assert.ErrorIs(t, err, domain.ErrEmptyName)
		reqRepo.AssertNotCalled(t, "Add")
	})

	t.Run("rejects empty external id", func(t *testing.T) {
		svc, reqRepo, _, _, mediaRepo := newRequestService(t)
		mediaRepo.On("GetByProviderExternal", "tmdb", "").Return(nil, domain.ErrMediaNotFound)

		_, err := svc.Create(context.Background(), 1, "tmdb", "", "Matrix", "/cover.jpg", domain.MediaTypeMovie, 1, nil, "")

		assert.Error(t, err)
		reqRepo.AssertNotCalled(t, "Add")
	})

	t.Run("rejects zero library id", func(t *testing.T) {
		svc, reqRepo, settings, _, mediaRepo := newRequestService(t)
		mediaRepo.On("GetByProviderExternal", "tmdb", "1").Return(nil, domain.ErrMediaNotFound)
		settings.AutoApproveOff()

		_, err := svc.Create(context.Background(), 1, "tmdb", "1", "Matrix", "/cover.jpg", domain.MediaTypeMovie, 0, nil, "")

		assert.ErrorIs(t, err, domain.ErrLibraryRequired)
		reqRepo.AssertNotCalled(t, "Add")
	})

	t.Run("rejects when media already exists", func(t *testing.T) {
		svc, reqRepo, settings, _, mediaRepo := newRequestService(t)
		mediaRepo.On("GetByProviderExternal", "tmdb", "1").Return(&domain.Media{Id: 5, Name: "X"}, nil)
		settings.AutoApproveOff()

		_, err := svc.Create(context.Background(), 1, "tmdb", "1", "Matrix", "/cover.jpg", domain.MediaTypeMovie, 1, nil, "")

		assert.ErrorIs(t, err, domain.ErrMediaAlreadyExists)
		reqRepo.AssertNotCalled(t, "Add")
	})

	t.Run("rejects duplicate pending/approved request", func(t *testing.T) {
		svc, reqRepo, settings, _, mediaRepo := newRequestService(t)
		mediaRepo.On("GetByProviderExternal", "tmdb", "1").Return(nil, domain.ErrMediaNotFound)
		reqRepo.On("ExistsByProviderExternal", "tmdb", "1").Return(true, nil)
		settings.AutoApproveOff()

		_, err := svc.Create(context.Background(), 1, "tmdb", "1", "Matrix", "/cover.jpg", domain.MediaTypeMovie, 1, nil, "")

		assert.ErrorIs(t, err, domain.ErrDuplicateRequest)
		reqRepo.AssertNotCalled(t, "Add")
	})

	t.Run("creates pending request when auto-approve off", func(t *testing.T) {
		svc, reqRepo, settings, _, mediaRepo := newRequestService(t)
		mediaRepo.On("GetByProviderExternal", "tmdb", "1").Return(nil, domain.ErrMediaNotFound)
		reqRepo.On("ExistsByProviderExternal", "tmdb", "1").Return(false, nil)
		settings.AutoApproveOff()
		reqRepo.On("Add", mock.AnythingOfType("*domain.MediaRequest")).Run(func(args mock.Arguments) {
			r := args.Get(0).(*domain.MediaRequest)
			r.Id = 10
		}).Return(nil)

		req, err := svc.Create(context.Background(), 1, "tmdb", "1", "Matrix", "/cover.jpg", domain.MediaTypeMovie, 1, nil, "")

		require.NoError(t, err)
		assert.Equal(t, domain.MediaRequestPending, req.Status)
		assert.Equal(t, domain.ID(10), req.Id)
	})

	t.Run("auto-approves when setting on", func(t *testing.T) {
		svc, reqRepo, settings, importer, mediaRepo := newRequestService(t)
		mediaRepo.On("GetByProviderExternal", "tmdb", "1").Return(nil, domain.ErrMediaNotFound)
		reqRepo.On("ExistsByProviderExternal", "tmdb", "1").Return(false, nil)
		settings.AutoApproveOn()
		reqRepo.On("Add", mock.AnythingOfType("*domain.MediaRequest")).Run(func(args mock.Arguments) {
			args.Get(0).(*domain.MediaRequest).Id = 10
		}).Return(nil)
		importer.On("Import", context.Background(), "tmdb", "1", domain.MediaTypeMovie, domain.ID(1), (*domain.ID)(nil), "", "/cover.jpg").Return(&domain.Media{Id: 99, Name: "X"}, nil)
		reqRepo.On("Update", mock.AnythingOfType("*domain.MediaRequest")).Return(nil)

		req, err := svc.Create(context.Background(), 1, "tmdb", "1", "Matrix", "/cover.jpg", domain.MediaTypeMovie, 1, nil, "")

		require.NoError(t, err)
		assert.Equal(t, domain.MediaRequestApproved, req.Status)
		assert.NotNil(t, req.ApprovedAt)
		require.NotNil(t, req.MediaID)
		assert.Equal(t, domain.ID(99), *req.MediaID)
	})

	t.Run("auto-approve import failure leaves pending and returns error", func(t *testing.T) {
		svc, reqRepo, settings, importer, mediaRepo := newRequestService(t)
		mediaRepo.On("GetByProviderExternal", "tmdb", "1").Return(nil, domain.ErrMediaNotFound)
		reqRepo.On("ExistsByProviderExternal", "tmdb", "1").Return(false, nil)
		settings.AutoApproveOn()
		reqRepo.On("Add", mock.AnythingOfType("*domain.MediaRequest")).Run(func(args mock.Arguments) {
			args.Get(0).(*domain.MediaRequest).Id = 10
		}).Return(nil)
		importErr := errors.New("provider down")
		importer.On("Import", context.Background(), "tmdb", "1", domain.MediaTypeMovie, domain.ID(1), (*domain.ID)(nil), "", "/cover.jpg").Return(nil, importErr)

		req, err := svc.Create(context.Background(), 1, "tmdb", "1", "Matrix", "/cover.jpg", domain.MediaTypeMovie, 1, nil, "")

		require.Error(t, err)
		require.NotNil(t, req)
		assert.Equal(t, domain.MediaRequestPending, req.Status)
	})

	t.Run("auto-approve update failure still returns approved request", func(t *testing.T) {
		svc, reqRepo, settings, importer, mediaRepo := newRequestService(t)
		mediaRepo.On("GetByProviderExternal", "tmdb", "1").Return(nil, domain.ErrMediaNotFound)
		reqRepo.On("ExistsByProviderExternal", "tmdb", "1").Return(false, nil)
		settings.AutoApproveOn()
		reqRepo.On("Add", mock.AnythingOfType("*domain.MediaRequest")).Run(func(args mock.Arguments) {
			args.Get(0).(*domain.MediaRequest).Id = 10
		}).Return(nil)
		importer.On("Import", context.Background(), "tmdb", "1", domain.MediaTypeMovie, domain.ID(1), (*domain.ID)(nil), "", "/cover.jpg").Return(&domain.Media{Id: 99}, nil)
		reqRepo.On("Update", mock.AnythingOfType("*domain.MediaRequest")).Return(errors.New("db error"))

		req, err := svc.Create(context.Background(), 1, "tmdb", "1", "Matrix", "/cover.jpg", domain.MediaTypeMovie, 1, nil, "")

		require.NoError(t, err)
		assert.Equal(t, domain.MediaRequestApproved, req.Status)
	})

	t.Run("resolves title and cover from provider when empty", func(t *testing.T) {
		svc, reqRepo, settings, importer, mediaRepo := newRequestService(t)
		mediaRepo.On("GetByProviderExternal", "openlibrary", "OL1W").Return(nil, domain.ErrMediaNotFound)
		reqRepo.On("ExistsByProviderExternal", "openlibrary", "OL1W").Return(false, nil)
		settings.AutoApproveOff()
		importer.On("LookupMedia", context.Background(), "openlibrary", "OL1W", domain.MediaTypeBook).
			Return(&domain.SearchResult{Title: "The Wizard of Oz", CoverURL: "http://cover/oz.jpg", MediaType: domain.MediaTypeBook}, nil)
		reqRepo.On("Add", mock.AnythingOfType("*domain.MediaRequest")).Run(func(args mock.Arguments) {
			r := args.Get(0).(*domain.MediaRequest)
			r.Id = 5
		}).Return(nil)

		req, err := svc.Create(context.Background(), 1, "openlibrary", "OL1W", "", "", domain.MediaTypeBook, 1, nil, "")

		require.NoError(t, err)
		assert.Equal(t, "The Wizard of Oz", req.Title)
		assert.Equal(t, "http://cover/oz.jpg", req.Cover)
	})
}

func TestMediaRequestService_Approve(t *testing.T) {
	t.Run("approves pending request and imports", func(t *testing.T) {
		svc, reqRepo, _, importer, _ := newRequestService(t)
		req := &domain.MediaRequest{
			Id:         1,
			UserId:     2,
			Provider:   "tmdb",
			ExternalID: "1",
			Type:       domain.MediaTypeMovie,
			LibraryID:  3,
			Status:     domain.MediaRequestPending,
		}
		reqRepo.On("GetByID", domain.ID(1)).Return(req, nil)
		importer.On("Import", context.Background(), "tmdb", "1", domain.MediaTypeMovie, domain.ID(3), (*domain.ID)(nil), "", "").Return(&domain.Media{Id: 99}, nil)
		reqRepo.On("Update", mock.AnythingOfType("*domain.MediaRequest")).Return(nil)

		result, err := svc.Approve(context.Background(), 1, 5)

		require.NoError(t, err)
		assert.Equal(t, domain.MediaRequestApproved, result.Status)
		assert.NotNil(t, result.ApprovedAt)
		require.NotNil(t, result.ApprovedBy)
		assert.Equal(t, domain.ID(5), *result.ApprovedBy)
		require.NotNil(t, result.MediaID)
		assert.Equal(t, domain.ID(99), *result.MediaID)
	})

	t.Run("rejects non-pending request", func(t *testing.T) {
		svc, reqRepo, _, _, _ := newRequestService(t)
		req := &domain.MediaRequest{Id: 1, Status: domain.MediaRequestApproved}
		reqRepo.On("GetByID", domain.ID(1)).Return(req, nil)

		_, err := svc.Approve(context.Background(), 1, 5)

		assert.ErrorIs(t, err, domain.ErrRequestNotPending)
	})

	t.Run("returns import error and leaves pending", func(t *testing.T) {
		svc, reqRepo, _, importer, _ := newRequestService(t)
		req := &domain.MediaRequest{Id: 1, Status: domain.MediaRequestPending, Provider: "tmdb", ExternalID: "1", Type: domain.MediaTypeMovie, LibraryID: 3}
		reqRepo.On("GetByID", domain.ID(1)).Return(req, nil)
		importer.On("Import", context.Background(), "tmdb", "1", domain.MediaTypeMovie, domain.ID(3), (*domain.ID)(nil), "", "").Return(nil, errors.New("down"))

		_, err := svc.Approve(context.Background(), 1, 5)

		assert.Error(t, err)
		assert.Equal(t, domain.MediaRequestPending, req.Status)
	})
}

func TestMediaRequestService_Reject(t *testing.T) {
	t.Run("rejects with notes", func(t *testing.T) {
		svc, reqRepo, _, _, _ := newRequestService(t)
		req := &domain.MediaRequest{Id: 1, Status: domain.MediaRequestPending, UserId: 2}
		reqRepo.On("GetByID", domain.ID(1)).Return(req, nil)
		reqRepo.On("Update", mock.AnythingOfType("*domain.MediaRequest")).Return(nil)

		notes := "wrong media"
		result, err := svc.Reject(context.Background(), 1, 5, &notes)

		require.NoError(t, err)
		assert.Equal(t, domain.MediaRequestRejected, result.Status)
		assert.NotNil(t, result.RejectedAt)
		assert.Equal(t, domain.ID(5), *result.RejectedBy)
		require.NotNil(t, result.Notes)
		assert.Equal(t, "wrong media", *result.Notes)
	})

	t.Run("rejects non-pending", func(t *testing.T) {
		svc, reqRepo, _, _, _ := newRequestService(t)
		req := &domain.MediaRequest{Id: 1, Status: domain.MediaRequestApproved}
		reqRepo.On("GetByID", domain.ID(1)).Return(req, nil)

		_, err := svc.Reject(context.Background(), 1, 5, nil)

		assert.ErrorIs(t, err, domain.ErrRequestNotPending)
	})
}

func TestMediaRequestService_Cancel(t *testing.T) {
	t.Run("owner cancels own pending request", func(t *testing.T) {
		svc, reqRepo, _, _, _ := newRequestService(t)
		req := &domain.MediaRequest{Id: 1, Status: domain.MediaRequestPending, UserId: 2}
		reqRepo.On("GetByID", domain.ID(1)).Return(req, nil)
		reqRepo.On("Update", mock.AnythingOfType("*domain.MediaRequest")).Return(nil)

		result, err := svc.Cancel(context.Background(), 1, 2)

		require.NoError(t, err)
		assert.Equal(t, domain.MediaRequestCanceled, result.Status)
		assert.NotNil(t, result.CanceledAt)
	})

	t.Run("non-owner cannot cancel", func(t *testing.T) {
		svc, reqRepo, _, _, _ := newRequestService(t)
		req := &domain.MediaRequest{Id: 1, Status: domain.MediaRequestPending, UserId: 2}
		reqRepo.On("GetByID", domain.ID(1)).Return(req, nil)

		_, err := svc.Cancel(context.Background(), 1, 99)

		assert.ErrorIs(t, err, domain.ErrRequestNotAuthorized)
	})

	t.Run("cannot cancel non-pending", func(t *testing.T) {
		svc, reqRepo, _, _, _ := newRequestService(t)
		req := &domain.MediaRequest{Id: 1, Status: domain.MediaRequestApproved, UserId: 2}
		reqRepo.On("GetByID", domain.ID(1)).Return(req, nil)

		_, err := svc.Cancel(context.Background(), 1, 2)

		assert.ErrorIs(t, err, domain.ErrRequestNotPending)
	})
}

func TestMediaRequestService_ListAll(t *testing.T) {
	svc, reqRepo, _, _, _ := newRequestService(t)
	expected := []domain.MediaRequest{{Id: 1, Status: domain.MediaRequestPending}}
	reqRepo.On("ListAll", 1, 20).Return(expected, nil)

	got, err := svc.ListAll(1, 20)

	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestMediaRequestService_ListByUser(t *testing.T) {
	svc, reqRepo, _, _, _ := newRequestService(t)
	expected := []domain.MediaRequest{{Id: 1, Status: domain.MediaRequestPending}}
	reqRepo.On("ListByUser", domain.ID(2), 1, 20).Return(expected, nil)

	got, err := svc.ListByUser(2, 1, 20)

	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestMediaRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     domain.MediaRequest
		wantErr error
	}{
		{"valid", domain.MediaRequest{Provider: "tmdb", ExternalID: "1", LibraryID: 1}, nil},
		{"empty provider", domain.MediaRequest{ExternalID: "1", LibraryID: 1}, domain.ErrEmptyName},
		{"empty external id", domain.MediaRequest{Provider: "tmdb", LibraryID: 1}, errors.New("external id cannot be empty")},
		{"zero library", domain.MediaRequest{Provider: "tmdb", ExternalID: "1"}, domain.ErrLibraryRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}
