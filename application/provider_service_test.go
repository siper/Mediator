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

type mockPartGroupRepo struct {
	mock.Mock
}

func (m *mockPartGroupRepo) Add(group *domain.PartGroup) error {
	args := m.Called(group)
	return args.Error(0)
}

func (m *mockPartGroupRepo) GetByMediaID(mediaID domain.ID) ([]domain.PartGroup, error) {
	args := m.Called(mediaID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.PartGroup), args.Error(1)
}

func (m *mockPartGroupRepo) Remove(id domain.ID) error {
	args := m.Called(id)
	return args.Error(0)
}

type mockCoverStore struct {
	mock.Mock
}

func (m *mockCoverStore) Store(ctx context.Context, sourceURL string) (string, error) {
	args := m.Called(ctx, sourceURL)
	return args.String(0), args.Error(1)
}

type mockQualityRepo struct {
	mock.Mock
}

func (m *mockQualityRepo) Add(p *domain.QualityProfile) error { return m.Called(p).Error(0) }
func (m *mockQualityRepo) GetById(id domain.ID) (*domain.QualityProfile, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.QualityProfile), args.Error(1)
}
func (m *mockQualityRepo) List() ([]domain.QualityProfile, error) {
	args := m.Called()
	return args.Get(0).([]domain.QualityProfile), args.Error(1)
}
func (m *mockQualityRepo) ListByType(t domain.MediaType) ([]domain.QualityProfile, error) {
	args := m.Called(t)
	return args.Get(0).([]domain.QualityProfile), args.Error(1)
}
func (m *mockQualityRepo) Update(p *domain.QualityProfile) error { return m.Called(p).Error(0) }
func (m *mockQualityRepo) Remove(id domain.ID) error             { return m.Called(id).Error(0) }

type fakeProvider struct {
	name string
	pm   *domain.ProviderMedia
	err  error
}

func (p *fakeProvider) Name() string { return p.name }

func (p *fakeProvider) Search(query string, mediaType *domain.MediaType, page, limit int) ([]domain.SearchResult, bool, error) {
	return nil, false, nil
}

func (p *fakeProvider) GetMedia(externalID string, mediaType domain.MediaType) (*domain.ProviderMedia, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.pm, nil
}

type fakeMetadataSource struct {
	providers []domain.MediaProvider
}

func (f *fakeMetadataSource) Active() []domain.MediaProvider { return f.providers }

func TestProviderService_Import(t *testing.T) {
	monitoredPart := func(mediaId domain.ID) any {
		return mock.MatchedBy(func(p *domain.Part) bool {
			return p != nil && p.MediaId == mediaId && p.Monitored
		})
	}

	setupMediaAndPart := func(mockMedia *mockMediaRepo, mockPart *mockPartRepo, mediaId domain.ID, mediaType domain.MediaType) {
		mockMedia.On("GetById", mediaId).Return(&domain.Media{Id: mediaId, Type: mediaType}, nil)
		mockPart.On("Add", monitoredPart(mediaId)).Return(nil)
	}

	t.Run("movie stores cover and creates one monitored placeholder part", func(t *testing.T) {
		mockMedia := new(mockMediaRepo)
		mockPart := new(mockPartRepo)
		mockGroup := new(mockPartGroupRepo)
		mockCover := new(mockCoverStore)

		provider := &fakeProvider{
			name: "fake",
			pm: &domain.ProviderMedia{
				Title:    "Title",
				CoverURL: "http://img.example/x.jpg",
			},
		}

		mockCover.On("Store", mock.Anything, "http://img.example/x.jpg").
			Return("/covers/abc.jpg", nil)
		folder := "My Folder"
		mockMedia.On("Create", "Title", "", &folder, strPtrOrNil("/covers/abc.jpg"),
			domain.MediaTypeMovie, idPtr(1), "fake", "123", (*domain.ID)(nil)).
			Return(&domain.Media{Id: 1, Name: "Title", Type: domain.MediaTypeMovie}, nil)
		setupMediaAndPart(mockMedia, mockPart, 1, domain.MediaTypeMovie)

		svc := NewProviderService(&fakeMetadataSource{providers: []domain.MediaProvider{provider}},
			NewMediaService(mockMedia, nil, nil, nil), NewPartService(mockPart, mockMedia, nil),
			mockGroup, mockPart, mockCover, new(mockQualityRepo))

		media, err := svc.Import(context.Background(), "fake", "123", domain.MediaTypeMovie, domain.ID(1), nil, "My Folder", "http://img.example/x.jpg")

		assert.NoError(t, err)
		require.NotNil(t, media)
		assert.Equal(t, domain.ID(1), media.Id)
		mockPart.AssertNumberOfCalls(t, "Add", 1)
		mockMedia.AssertCalled(t, "Create", "Title", "", &folder, strPtrOrNil("/covers/abc.jpg"),
			domain.MediaTypeMovie, idPtr(1), "fake", "123", (*domain.ID)(nil))
	})

	t.Run("cover store error is tolerated, placeholder part still created", func(t *testing.T) {
		mockMedia := new(mockMediaRepo)
		mockPart := new(mockPartRepo)
		mockGroup := new(mockPartGroupRepo)
		mockCover := new(mockCoverStore)

		provider := &fakeProvider{
			name: "fake",
			pm: &domain.ProviderMedia{
				Title:    "Title",
				CoverURL: "http://img.example/x.jpg",
			},
		}

		mockCover.On("Store", mock.Anything, "http://img.example/x.jpg").
			Return("", errors.New("boom"))
		mockMedia.On("Create", "Title", "", (*string)(nil), strPtrOrNil("http://img.example/x.jpg"),
			domain.MediaTypeMovie, idPtr(1), "fake", "123", (*domain.ID)(nil)).
			Return(&domain.Media{Id: 1}, nil)
		setupMediaAndPart(mockMedia, mockPart, 1, domain.MediaTypeMovie)

		svc := NewProviderService(&fakeMetadataSource{providers: []domain.MediaProvider{provider}},
			NewMediaService(mockMedia, nil, nil, nil), NewPartService(mockPart, mockMedia, nil),
			mockGroup, mockPart, mockCover, new(mockQualityRepo))

		_, err := svc.Import(context.Background(), "fake", "123", domain.MediaTypeMovie, domain.ID(1), nil, "", "http://img.example/x.jpg")

		assert.NoError(t, err)
		mockPart.AssertNumberOfCalls(t, "Add", 1)
	})

	t.Run("book creates single monitored placeholder part when provider has none", func(t *testing.T) {
		mockMedia := new(mockMediaRepo)
		mockPart := new(mockPartRepo)
		mockGroup := new(mockPartGroupRepo)
		mockCover := new(mockCoverStore)

		provider := &fakeProvider{name: "fake", pm: &domain.ProviderMedia{Title: "Book"}}

		mockCover.On("Store", mock.Anything, "").Return("", nil)
		var nilCover *string
		mockMedia.On("Create", "Book", "", (*string)(nil), nilCover, domain.MediaTypeBook, idPtr(5), "fake", "9", (*domain.ID)(nil)).
			Return(&domain.Media{Id: 5, Type: domain.MediaTypeBook}, nil)
		setupMediaAndPart(mockMedia, mockPart, 5, domain.MediaTypeBook)

		svc := NewProviderService(&fakeMetadataSource{providers: []domain.MediaProvider{provider}},
			NewMediaService(mockMedia, nil, nil, nil), NewPartService(mockPart, mockMedia, nil),
			mockGroup, mockPart, mockCover, new(mockQualityRepo))

		_, err := svc.Import(context.Background(), "fake", "9", domain.MediaTypeBook, domain.ID(5), nil, "", "")

		assert.NoError(t, err)
		mockPart.AssertNumberOfCalls(t, "Add", 1)
	})

	t.Run("series creates monitored parts from provider episodes", func(t *testing.T) {
		mockMedia := new(mockMediaRepo)
		mockPart := new(mockPartRepo)
		mockGroup := new(mockPartGroupRepo)
		mockCover := new(mockCoverStore)

		ep1 := "E1"
		ep2 := "E2"
		season := "Season 1"
		provider := &fakeProvider{name: "fake", pm: &domain.ProviderMedia{
			Title:  "Series",
			Groups: []domain.ProviderGroup{{Name: season, Order: 1}},
			Parts: []domain.ProviderPart{
				{Name: &ep1, GroupName: &season, GroupOrder: intPtr(1)},
				{Name: &ep2, GroupName: &season, GroupOrder: intPtr(2)},
			},
		}}

		mockCover.On("Store", mock.Anything, "").Return("", nil)
		var nilCover *string
		mockMedia.On("Create", "Series", "", (*string)(nil), nilCover, domain.MediaTypeSeries, idPtr(3), "fake", "7", (*domain.ID)(nil)).
			Return(&domain.Media{Id: 3, Type: domain.MediaTypeSeries}, nil)
		mockGroup.On("Add", mock.MatchedBy(func(g *domain.PartGroup) bool {
			return g != nil && g.Name == season && g.Order == 1 && g.MediaId == 3
		})).Return(nil)
		mockMedia.On("GetById", domain.ID(3)).Return(&domain.Media{Id: 3, Type: domain.MediaTypeSeries}, nil)
		mockPart.On("Add", monitoredPart(3)).Return(nil).Twice()

		svc := NewProviderService(&fakeMetadataSource{providers: []domain.MediaProvider{provider}},
			NewMediaService(mockMedia, nil, nil, nil), NewPartService(mockPart, mockMedia, nil),
			mockGroup, mockPart, mockCover, new(mockQualityRepo))

		_, err := svc.Import(context.Background(), "fake", "7", domain.MediaTypeSeries, domain.ID(3), nil, "", "")

		assert.NoError(t, err)
		mockPart.AssertNumberOfCalls(t, "Add", 2)
		mockGroup.AssertNumberOfCalls(t, "Add", 1)
	})

	t.Run("unknown provider returns error", func(t *testing.T) {
		svc := NewProviderService(&fakeMetadataSource{providers: nil},
			NewMediaService(new(mockMediaRepo), nil, nil, nil), NewPartService(new(mockPartRepo), new(mockMediaRepo), nil),
			new(mockPartGroupRepo), new(mockPartRepo), new(mockCoverStore), new(mockQualityRepo))

		_, err := svc.Import(context.Background(), "nope", "1", domain.MediaTypeMovie, domain.ID(1), nil, "", "")
		assert.ErrorIs(t, err, domain.ErrProviderNotSupported)
	})
}

func intPtr(v int) *int { return &v }

func TestProviderService_RefreshMedia(t *testing.T) {
	t.Run("updates media with refreshed cover", func(t *testing.T) {
		mockMedia := new(mockMediaRepo)
		mockPart := new(mockPartRepo)
		mockGroup := new(mockPartGroupRepo)
		mockCover := new(mockCoverStore)

		existing := &domain.Media{Id: 7, Name: "Old", Type: domain.MediaTypeMovie,
			ProviderID: "fake", ExternalID: "123"}

		provider := &fakeProvider{
			name: "fake",
			pm: &domain.ProviderMedia{
				Title:    "New",
				CoverURL: "http://img.example/y.jpg",
			},
		}

		mockMedia.On("GetById", domain.ID(7)).Return(existing, nil)
		mockCover.On("Store", mock.Anything, "http://img.example/y.jpg").
			Return("/covers/yyy.jpg", nil)
		mockMedia.On("Update", domain.ID(7), "New", "", strPtrOrNil("/covers/yyy.jpg")).Return(nil)
		mockGroup.On("GetByMediaID", domain.ID(7)).Return([]domain.PartGroup{}, nil)
		mockPart.On("GetByMediaId", domain.ID(7)).Return([]domain.Part{}, nil)

		svc := NewProviderService(&fakeMetadataSource{providers: []domain.MediaProvider{provider}},
			NewMediaService(mockMedia, nil, nil, nil), NewPartService(mockPart, mockMedia, nil),
			mockGroup, mockPart, mockCover, new(mockQualityRepo))

		err := svc.RefreshMedia(context.Background(), domain.ID(7))

		assert.NoError(t, err)
		mockMedia.AssertCalled(t, "Update", domain.ID(7), "New", "", strPtrOrNil("/covers/yyy.jpg"))
	})

	t.Run("preserves path and monitored for matched part", func(t *testing.T) {
		mockMedia := new(mockMediaRepo)
		mockPart := new(mockPartRepo)
		mockGroup := new(mockPartGroupRepo)
		mockCover := new(mockCoverStore)

		ep1 := "E1"
		path := "/lib/series/s1e1.mkv"
		existingMedia := &domain.Media{Id: 3, Name: "Old", Type: domain.MediaTypeSeries,
			ProviderID: "fake", ExternalID: "7"}
		order := 1
		existingPart := domain.Part{
			Id: 100, Name: &ep1, GroupOrder: &order, MediaId: 3,
			Path: &path, Monitored: false,
		}

		provider := &fakeProvider{name: "fake", pm: &domain.ProviderMedia{
			Title: "New",
			Parts: []domain.ProviderPart{{Name: &ep1, GroupOrder: &order}},
		}}

		mockMedia.On("GetById", domain.ID(3)).Return(existingMedia, nil)
		mockMedia.On("Update", domain.ID(3), "New", "", mock.Anything).Return(nil)
		mockGroup.On("GetByMediaID", domain.ID(3)).Return([]domain.PartGroup{}, nil)
		mockPart.On("GetByMediaId", domain.ID(3)).Return([]domain.Part{existingPart}, nil)
		mockPart.On("Update", mock.MatchedBy(func(p *domain.Part) bool {
			return p.Id == 100 && p.Path != nil && *p.Path == path && p.Monitored == false
		})).Return(nil)

		svc := NewProviderService(&fakeMetadataSource{providers: []domain.MediaProvider{provider}},
			NewMediaService(mockMedia, nil, nil, nil), NewPartService(mockPart, mockMedia, nil),
			mockGroup, mockPart, mockCover, new(mockQualityRepo))

		err := svc.RefreshMedia(context.Background(), domain.ID(3))

		assert.NoError(t, err)
		mockPart.AssertCalled(t, "Update", mock.MatchedBy(func(p *domain.Part) bool {
			return p.Path != nil && *p.Path == path && !p.Monitored
		}))
		mockPart.AssertNotCalled(t, "Add")
	})

	t.Run("preserves existing cover when provider returns none", func(t *testing.T) {
		mockMedia := new(mockMediaRepo)
		mockPart := new(mockPartRepo)
		mockGroup := new(mockPartGroupRepo)
		mockCover := new(mockCoverStore)

		existingCover := "/covers/old.jpg"
		existing := &domain.Media{Id: 9, Name: "Old", Type: domain.MediaTypeBook,
			ProviderID: "fake", ExternalID: "5", Cover: &existingCover}

		provider := &fakeProvider{name: "fake", pm: &domain.ProviderMedia{Title: "New"}}

		mockMedia.On("GetById", domain.ID(9)).Return(existing, nil)
		mockMedia.On("Update", domain.ID(9), "New", "", &existingCover).Return(nil)
		mockGroup.On("GetByMediaID", domain.ID(9)).Return([]domain.PartGroup{}, nil)
		mockPart.On("GetByMediaId", domain.ID(9)).Return([]domain.Part{}, nil)

		svc := NewProviderService(&fakeMetadataSource{providers: []domain.MediaProvider{provider}},
			NewMediaService(mockMedia, nil, nil, nil), NewPartService(mockPart, mockMedia, nil),
			mockGroup, mockPart, mockCover, new(mockQualityRepo))

		err := svc.RefreshMedia(context.Background(), domain.ID(9))

		assert.NoError(t, err)
		mockMedia.AssertCalled(t, "Update", domain.ID(9), "New", "", &existingCover)
	})

	t.Run("adds missing provider part as monitored", func(t *testing.T) {
		mockMedia := new(mockMediaRepo)
		mockPart := new(mockPartRepo)
		mockGroup := new(mockPartGroupRepo)
		mockCover := new(mockCoverStore)

		ep2 := "E2"
		existingMedia := &domain.Media{Id: 3, Name: "Old", Type: domain.MediaTypeSeries,
			ProviderID: "fake", ExternalID: "7"}

		provider := &fakeProvider{name: "fake", pm: &domain.ProviderMedia{
			Title: "New",
			Parts: []domain.ProviderPart{{Name: &ep2}},
		}}

		mockMedia.On("GetById", domain.ID(3)).Return(existingMedia, nil)
		mockMedia.On("Update", domain.ID(3), "New", "", mock.Anything).Return(nil)
		mockGroup.On("GetByMediaID", domain.ID(3)).Return([]domain.PartGroup{}, nil)
		mockPart.On("GetByMediaId", domain.ID(3)).Return([]domain.Part{}, nil)
		mockPart.On("Add", mock.MatchedBy(func(p *domain.Part) bool {
			return p.MediaId == 3 && p.Monitored && p.Name != nil && *p.Name == "E2"
		})).Return(nil)

		svc := NewProviderService(&fakeMetadataSource{providers: []domain.MediaProvider{provider}},
			NewMediaService(mockMedia, nil, nil, nil), NewPartService(mockPart, mockMedia, nil),
			mockGroup, mockPart, mockCover, new(mockQualityRepo))

		err := svc.RefreshMedia(context.Background(), domain.ID(3))

		assert.NoError(t, err)
		mockPart.AssertNumberOfCalls(t, "Add", 1)
	})
}
