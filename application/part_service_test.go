package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

type mockPartRepo struct {
	mock.Mock
}

func (m *mockPartRepo) Add(part *domain.Part) error {
	args := m.Called(part)
	return args.Error(0)
}

func (m *mockPartRepo) GetById(id domain.ID) (*domain.Part, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Part), args.Error(1)
}

func (m *mockPartRepo) GetByMediaId(mediaId domain.ID) ([]domain.Part, error) {
	args := m.Called(mediaId)
	return args.Get(0).([]domain.Part), args.Error(1)
}

func (m *mockPartRepo) GetWanted(page int, limit int) ([]domain.Part, error) {
	args := m.Called(page, limit)
	return args.Get(0).([]domain.Part), args.Error(1)
}

func (m *mockPartRepo) Remove(id domain.ID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *mockPartRepo) Update(part *domain.Part) error {
	args := m.Called(part)
	return args.Error(0)
}

type mockGroupRepo struct {
	mock.Mock
}

func (m *mockGroupRepo) Add(group *domain.PartGroup) error {
	args := m.Called(group)
	return args.Error(0)
}

func (m *mockGroupRepo) GetByMediaID(mediaID domain.ID) ([]domain.PartGroup, error) {
	args := m.Called(mediaID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.PartGroup), args.Error(1)
}

func (m *mockGroupRepo) Remove(id domain.ID) error {
	args := m.Called(id)
	return args.Error(0)
}

func TestPartService_Add(t *testing.T) {
	name := "Chapter 1"

	t.Run("adds valid part", func(t *testing.T) {
		mockPart := new(mockPartRepo)
		mockMedia := new(mockMediaRepo)
		svc := NewPartService(mockPart, mockMedia, nil)
		part := &domain.Part{Name: &name, MediaId: 1}

		mockMedia.On("GetById", domain.ID(1)).Return(&domain.Media{Id: 1}, nil)
		mockPart.On("Add", part).Return(nil)

		err := svc.Add(part)

		assert.NoError(t, err)
		mockPart.AssertExpectations(t)
		mockMedia.AssertExpectations(t)
	})

	t.Run("rejects part with empty media id", func(t *testing.T) {
		mockPart := new(mockPartRepo)
		mockMedia := new(mockMediaRepo)
		svc := NewPartService(mockPart, mockMedia, nil)

		err := svc.Add(&domain.Part{Name: &name, MediaId: 0})

		assert.ErrorIs(t, err, domain.ErrMediaNotFound)
		mockPart.AssertNotCalled(t, "Add")
		mockMedia.AssertNotCalled(t, "GetById")
	})

	t.Run("rejects when media not found", func(t *testing.T) {
		mockPart := new(mockPartRepo)
		mockMedia := new(mockMediaRepo)
		svc := NewPartService(mockPart, mockMedia, nil)

		mockMedia.On("GetById", domain.ID(99)).Return(nil, domain.ErrMediaNotFound)

		err := svc.Add(&domain.Part{Name: &name, MediaId: 99})

		assert.ErrorIs(t, err, domain.ErrMediaNotFound)
		mockPart.AssertNotCalled(t, "Add")
	})
}

func TestPartService_GetByMediaID(t *testing.T) {
	t.Run("returns parts for media", func(t *testing.T) {
		mockPart := new(mockPartRepo)
		mockMedia := new(mockMediaRepo)
		svc := NewPartService(mockPart, mockMedia, nil)
		expected := []domain.Part{{MediaId: 1}}

		mockPart.On("GetByMediaId", domain.ID(1)).Return(expected, nil)

		result, err := svc.GetByMediaID(1)

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})
}

func TestPartService_Remove(t *testing.T) {
	t.Run("removes part", func(t *testing.T) {
		mockPart := new(mockPartRepo)
		mockMedia := new(mockMediaRepo)
		svc := NewPartService(mockPart, mockMedia, nil)

		mockPart.On("Remove", domain.ID(1)).Return(nil)

		err := svc.Remove(1)

		assert.NoError(t, err)
	})
}

func TestPartService_Update(t *testing.T) {
	name := "Updated"
	monitored := true

	t.Run("applies patch to existing part", func(t *testing.T) {
		mockPart := new(mockPartRepo)
		mockMedia := new(mockMediaRepo)
		svc := NewPartService(mockPart, mockMedia, nil)
		existing := &domain.Part{Id: 1, MediaId: 5, Monitored: false}

		mockPart.On("GetById", domain.ID(1)).Return(existing, nil)
		mockPart.On("Update", &domain.Part{
			Id: 1, Name: &name, MediaId: 5, Monitored: true,
		}).Return(nil)

		result, err := svc.Update(domain.ID(1), PartPatch{
			Name:      &name,
			Monitored: &monitored,
		})

		assert.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "Updated", *result.Name)
		assert.True(t, result.Monitored)
		mockPart.AssertExpectations(t)
	})

	t.Run("rejects empty name in patch", func(t *testing.T) {
		mockPart := new(mockPartRepo)
		mockMedia := new(mockMediaRepo)
		svc := NewPartService(mockPart, mockMedia, nil)
		empty := ""
		existing := &domain.Part{Id: 1, MediaId: 5}

		mockPart.On("GetById", domain.ID(1)).Return(existing, nil)

		_, err := svc.Update(domain.ID(1), PartPatch{Name: &empty})

		assert.ErrorIs(t, err, domain.ErrEmptyName)
		mockPart.AssertNotCalled(t, "Update")
	})

	t.Run("returns error when part not found", func(t *testing.T) {
		mockPart := new(mockPartRepo)
		mockMedia := new(mockMediaRepo)
		svc := NewPartService(mockPart, mockMedia, nil)

		mockPart.On("GetById", domain.ID(99)).Return(nil, domain.ErrPartNotFound)

		_, err := svc.Update(domain.ID(99), PartPatch{Monitored: &monitored})

		assert.ErrorIs(t, err, domain.ErrPartNotFound)
		mockPart.AssertNotCalled(t, "Update")
	})
}

func TestPartService_Wanted(t *testing.T) {
	t.Run("returns monitored parts without path", func(t *testing.T) {
		mockPart := new(mockPartRepo)
		mockMedia := new(mockMediaRepo)
		svc := NewPartService(mockPart, mockMedia, nil)
		expected := []domain.Part{{Id: 1, MediaId: 1, Monitored: true}}

		mockPart.On("GetWanted", 1, 20).Return(expected, nil)

		result, err := svc.Wanted(1, 20)

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("clamps pagination", func(t *testing.T) {
		mockPart := new(mockPartRepo)
		mockMedia := new(mockMediaRepo)
		svc := NewPartService(mockPart, mockMedia, nil)

		mockPart.On("GetWanted", 1, 20).Return([]domain.Part{}, nil)

		_, err := svc.Wanted(0, 0)

		assert.NoError(t, err)
		mockPart.AssertCalled(t, "GetWanted", 1, 20)
	})
}

func TestPartService_WantedItems(t *testing.T) {
	t.Run("hydrates media and season", func(t *testing.T) {
		mockPart := new(mockPartRepo)
		mockMedia := new(mockMediaRepo)
		mockGroup := new(mockGroupRepo)
		svc := NewPartService(mockPart, mockMedia, mockGroup)

		epName := "Pilot"
		groupID := domain.ID(7)
		groupOrder := 1
		parts := []domain.Part{{
			Id: 42, Name: &epName, GroupOrder: &groupOrder, GroupId: &groupID,
			MediaId: 3, Monitored: true,
		}}
		media := &domain.Media{Id: 3, Name: "Show", Type: domain.MediaTypeSeries}
		groups := []domain.PartGroup{{Id: 7, Name: "Season 1", Order: 1, MediaId: 3}}

		mockPart.On("GetWanted", 1, 20).Return(parts, nil)
		mockMedia.On("GetById", domain.ID(3)).Return(media, nil)
		mockGroup.On("GetByMediaID", domain.ID(3)).Return(groups, nil)

		items, err := svc.WantedItems(1, 20)

		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, domain.ID(42), items[0].Part.Id)
		assert.Equal(t, "Show", items[0].Media.Name)
		require.NotNil(t, items[0].Season)
		assert.Equal(t, 1, *items[0].Season)
	})

	t.Run("movie has no season", func(t *testing.T) {
		mockPart := new(mockPartRepo)
		mockMedia := new(mockMediaRepo)
		svc := NewPartService(mockPart, mockMedia, nil)

		parts := []domain.Part{{Id: 1, MediaId: 9, Monitored: true}}
		media := &domain.Media{Id: 9, Name: "Film", Type: domain.MediaTypeMovie}

		mockPart.On("GetWanted", 1, 20).Return(parts, nil)
		mockMedia.On("GetById", domain.ID(9)).Return(media, nil)

		items, err := svc.WantedItems(1, 20)

		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Nil(t, items[0].Season)
		assert.Equal(t, "Film", items[0].Media.Name)
	})
}
