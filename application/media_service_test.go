package application

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"stersh.ru/mediator/domain"
)

type mockMediaRepo struct {
	mock.Mock
}

func (m *mockMediaRepo) GetById(id domain.ID) (*domain.Media, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Media), args.Error(1)
}

func (m *mockMediaRepo) Create(name string, originalName string, folder *string, cover *string, mediaType domain.MediaType, libraryID *domain.ID, providerID string, externalID string, profileID *domain.ID) (*domain.Media, error) {
	args := m.Called(name, originalName, folder, cover, mediaType, libraryID, providerID, externalID, profileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Media), args.Error(1)
}

func (m *mockMediaRepo) Update(id domain.ID, name string, originalName string, cover *string) error {
	args := m.Called(id, name, originalName, cover)
	return args.Error(0)
}

func (m *mockMediaRepo) UpdateProfile(id domain.ID, profileID *domain.ID) error {
	return m.Called(id, profileID).Error(0)
}

func (m *mockMediaRepo) UpdateProviderMeta(id domain.ID, status domain.MediaStatus, lastModified string) error {
	args := m.Called(id, status, lastModified)
	return args.Error(0)
}

func (m *mockMediaRepo) Remove(id domain.ID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *mockMediaRepo) GetPaged(page int, limit int, mediaType *domain.MediaType) ([]domain.Media, error) {
	args := m.Called(page, limit, mediaType)
	return args.Get(0).([]domain.Media), args.Error(1)
}

func (m *mockMediaRepo) GetByProviderExternal(provider string, externalID string) (*domain.Media, error) {
	args := m.Called(provider, externalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Media), args.Error(1)
}

func TestMediaService_Create(t *testing.T) {
	cover := "cover.jpg"

	t.Run("creates valid media", func(t *testing.T) {
		mockRepo := new(mockMediaRepo)
		svc := NewMediaService(mockRepo, nil, nil, nil)
		expected := &domain.Media{Name: "Test", Cover: &cover, Type: domain.MediaTypeMovie, ProviderID: "tmdb", ExternalID: "123"}

		mockRepo.On("Create", "Test", "", (*string)(nil), &cover, domain.MediaTypeMovie, idPtr(1), "tmdb", "123", (*domain.ID)(nil)).Return(expected, nil)

		result, err := svc.Create("Test", "", nil, &cover, domain.MediaTypeMovie, idPtr(1), "tmdb", "123", nil)

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("rejects empty name", func(t *testing.T) {
		mockRepo := new(mockMediaRepo)
		svc := NewMediaService(mockRepo, nil, nil, nil)

		result, err := svc.Create("", "", nil, nil, domain.MediaTypeBook, nil, "", "", nil)

		assert.ErrorIs(t, err, domain.ErrEmptyName)
		assert.Nil(t, result)
		mockRepo.AssertNotCalled(t, "Create")
	})

	t.Run("rejects invalid media type", func(t *testing.T) {
		mockRepo := new(mockMediaRepo)
		svc := NewMediaService(mockRepo, nil, nil, nil)

		result, err := svc.Create("Test", "", nil, nil, domain.MediaType(99), nil, "", "", nil)

		assert.ErrorIs(t, err, domain.ErrInvalidMediaType)
		assert.Nil(t, result)
		mockRepo.AssertNotCalled(t, "Create")
	})

	t.Run("creates media with nil cover", func(t *testing.T) {
		mockRepo := new(mockMediaRepo)
		svc := NewMediaService(mockRepo, nil, nil, nil)
		expected := &domain.Media{Name: "No Cover", Type: domain.MediaTypeMovie}
		var nilCover *string = nil
		var nilLib *domain.ID

		mockRepo.On("Create", "No Cover", "", (*string)(nil), nilCover, domain.MediaTypeMovie, nilLib, "", "", (*domain.ID)(nil)).Return(expected, nil)

		result, err := svc.Create("No Cover", "", nil, nilCover, domain.MediaTypeMovie, nilLib, "", "", nil)

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("creates media with quality profile", func(t *testing.T) {
		mockRepo := new(mockMediaRepo)
		svc := NewMediaService(mockRepo, nil, nil, nil)
		expected := &domain.Media{Name: "Test", Type: domain.MediaTypeMovie, QualityProfileID: idPtr(7)}

		mockRepo.On("Create", "Test", "", (*string)(nil), (*string)(nil), domain.MediaTypeMovie, idPtr(1), "tmdb", "123", idPtr(7)).Return(expected, nil)

		result, err := svc.Create("Test", "", nil, nil, domain.MediaTypeMovie, idPtr(1), "tmdb", "123", idPtr(7))

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
		mockRepo.AssertExpectations(t)
	})
}

func TestMediaService_GetByID(t *testing.T) {
	t.Run("returns media", func(t *testing.T) {
		mockRepo := new(mockMediaRepo)
		svc := NewMediaService(mockRepo, nil, nil, nil)
		expected := &domain.Media{Id: 1, Name: "Test", Type: domain.MediaTypeMovie}

		mockRepo.On("GetById", domain.ID(1)).Return(expected, nil)

		result, err := svc.GetByID(1)

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		mockRepo := new(mockMediaRepo)
		svc := NewMediaService(mockRepo, nil, nil, nil)

		mockRepo.On("GetById", domain.ID(99)).Return(nil, domain.ErrMediaNotFound)

		result, err := svc.GetByID(99)

		assert.ErrorIs(t, err, domain.ErrMediaNotFound)
		assert.Nil(t, result)
	})
}

func TestMediaService_Update(t *testing.T) {
	t.Run("updates media", func(t *testing.T) {
		mockRepo := new(mockMediaRepo)
		svc := NewMediaService(mockRepo, nil, nil, nil)
		cover := "new-cover.jpg"

		mockRepo.On("Update", domain.ID(1), "New Name", "", &cover).Return(nil)

		err := svc.Update(1, "New Name", "", &cover)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		mockRepo := new(mockMediaRepo)
		svc := NewMediaService(mockRepo, nil, nil, nil)
		var nilCover *string

		mockRepo.On("Update", domain.ID(99), "New Name", "", nilCover).Return(domain.ErrMediaNotFound)

		err := svc.Update(99, "New Name", "", nilCover)

		assert.ErrorIs(t, err, domain.ErrMediaNotFound)
	})
}

func TestMediaService_UpdateProfile(t *testing.T) {
	t.Run("sets profile", func(t *testing.T) {
		mockRepo := new(mockMediaRepo)
		svc := NewMediaService(mockRepo, nil, nil, nil)

		mockRepo.On("UpdateProfile", domain.ID(1), idPtr(5)).Return(nil)

		err := svc.UpdateProfile(1, idPtr(5))

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("clears profile with nil", func(t *testing.T) {
		mockRepo := new(mockMediaRepo)
		svc := NewMediaService(mockRepo, nil, nil, nil)

		mockRepo.On("UpdateProfile", domain.ID(1), (*domain.ID)(nil)).Return(nil)

		err := svc.UpdateProfile(1, nil)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestMediaService_Remove(t *testing.T) {
	t.Run("removes media", func(t *testing.T) {
		mockRepo := new(mockMediaRepo)
		svc := NewMediaService(mockRepo, nil, nil, nil)

		mockRepo.On("Remove", domain.ID(1)).Return(nil)

		err := svc.Remove(1, false)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns error", func(t *testing.T) {
		mockRepo := new(mockMediaRepo)
		svc := NewMediaService(mockRepo, nil, nil, nil)

		mockRepo.On("Remove", domain.ID(99)).Return(domain.ErrMediaNotFound)

		err := svc.Remove(99, false)

		assert.ErrorIs(t, err, domain.ErrMediaNotFound)
	})

	t.Run("deletes media folder when requested", func(t *testing.T) {
		mockRepo := new(mockMediaRepo)
		mockPart := new(mockPartRepo)
		mockLib := new(mockLibraryRepo)
		fs := &fakeFS{}
		svc := NewMediaService(mockRepo, mockPart, mockLib, fs)

		libID := domain.ID(2)
		media := &domain.Media{Id: 1, Name: "My Movie", Type: domain.MediaTypeMovie, LibraryID: &libID}
		filePath := filepath.Join("/data", "movies", "My Movie", "My Movie.mkv")
		folderPath := filepath.Join("/data", "movies", "My Movie")
		path := filePath

		mockRepo.On("GetById", domain.ID(1)).Return(media, nil)
		mockPart.On("GetByMediaId", domain.ID(1)).Return([]domain.Part{{Id: 10, MediaId: 1, Path: &path}}, nil)
		mockLib.On("GetById", libID).Return(&domain.Library{Id: 2, Path: "/data/movies", Type: domain.MediaTypeMovie}, nil)
		mockRepo.On("Remove", domain.ID(1)).Return(nil)

		err := svc.Remove(1, true)

		assert.NoError(t, err)
		assert.Contains(t, fs.removed, folderPath)
		mockRepo.AssertExpectations(t)
		mockPart.AssertExpectations(t)
		mockLib.AssertExpectations(t)
	})

	t.Run("deletes series root for multiple seasons", func(t *testing.T) {
		mockRepo := new(mockMediaRepo)
		mockPart := new(mockPartRepo)
		mockLib := new(mockLibraryRepo)
		fs := &fakeFS{}
		svc := NewMediaService(mockRepo, mockPart, mockLib, fs)

		libID := domain.ID(3)
		media := &domain.Media{Id: 1, Name: "Show", Type: domain.MediaTypeSeries, LibraryID: &libID}
		p1 := filepath.Join("/data", "series", "Show", "Season 01", "e1.mkv")
		p2 := filepath.Join("/data", "series", "Show", "Season 02", "e2.mkv")
		root := filepath.Join("/data", "series", "Show")

		mockRepo.On("GetById", domain.ID(1)).Return(media, nil)
		mockPart.On("GetByMediaId", domain.ID(1)).Return([]domain.Part{
			{Id: 10, MediaId: 1, Path: &p1},
			{Id: 11, MediaId: 1, Path: &p2},
		}, nil)
		mockLib.On("GetById", libID).Return(&domain.Library{Id: 3, Path: "/data/series", Type: domain.MediaTypeSeries}, nil)
		mockRepo.On("Remove", domain.ID(1)).Return(nil)

		err := svc.Remove(1, true)

		assert.NoError(t, err)
		assert.Contains(t, fs.removed, root)
	})

	t.Run("skips file deletion when unchecked", func(t *testing.T) {
		mockRepo := new(mockMediaRepo)
		mockPart := new(mockPartRepo)
		fs := &fakeFS{}
		svc := NewMediaService(mockRepo, mockPart, nil, fs)

		mockRepo.On("Remove", domain.ID(1)).Return(nil)

		err := svc.Remove(1, false)

		assert.NoError(t, err)
		assert.Empty(t, fs.removed)
		mockPart.AssertNotCalled(t, "GetByMediaId", mock.Anything)
	})
}

func TestMediaService_GetPaged(t *testing.T) {
	t.Run("returns paged results", func(t *testing.T) {
		mockRepo := new(mockMediaRepo)
		svc := NewMediaService(mockRepo, nil, nil, nil)
		expected := []domain.Media{{Name: "M1"}, {Name: "M2"}}

		mockRepo.On("GetPaged", 1, 20, (*domain.MediaType)(nil)).Return(expected, nil)

		result, err := svc.GetPaged(1, 20, nil)

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("filters by type", func(t *testing.T) {
		mockRepo := new(mockMediaRepo)
		svc := NewMediaService(mockRepo, nil, nil, nil)
		mType := domain.MediaTypeBook
		expected := []domain.Media{{Name: "Book1"}}

		mockRepo.On("GetPaged", 1, 20, &mType).Return(expected, nil)

		result, err := svc.GetPaged(1, 20, &mType)

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("clamps page to minimum 1", func(t *testing.T) {
		mockRepo := new(mockMediaRepo)
		svc := NewMediaService(mockRepo, nil, nil, nil)

		mockRepo.On("GetPaged", 1, 20, (*domain.MediaType)(nil)).Return([]domain.Media{}, nil)

		_, err := svc.GetPaged(0, 20, nil)

		assert.NoError(t, err)
		mockRepo.AssertCalled(t, "GetPaged", 1, 20, (*domain.MediaType)(nil))
	})

	t.Run("clamps limit to valid range", func(t *testing.T) {
		mockRepo := new(mockMediaRepo)
		svc := NewMediaService(mockRepo, nil, nil, nil)

		mockRepo.On("GetPaged", 1, 20, (*domain.MediaType)(nil)).Return([]domain.Media{}, nil)

		_, err := svc.GetPaged(1, 0, nil)

		assert.NoError(t, err)
		mockRepo.AssertCalled(t, "GetPaged", 1, 20, (*domain.MediaType)(nil))
	})

	t.Run("clamps limit to max 100", func(t *testing.T) {
		mockRepo := new(mockMediaRepo)
		svc := NewMediaService(mockRepo, nil, nil, nil)

		mockRepo.On("GetPaged", 1, 100, (*domain.MediaType)(nil)).Return([]domain.Media{}, nil)

		_, err := svc.GetPaged(1, 500, nil)

		assert.NoError(t, err)
		mockRepo.AssertCalled(t, "GetPaged", 1, 100, (*domain.MediaType)(nil))
	})
}
