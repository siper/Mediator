package application

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

func TestHistoryService_ListItems(t *testing.T) {
	t.Run("enriches with media", func(t *testing.T) {
		history := new(mockHistoryRepo)
		media := new(mockMediaRepo)
		svc := NewHistoryService(history, media, nil, nil)

		cover := "/covers/1.jpg"
		history.On("List", 1, 20).Return([]domain.History{{
			Id: 1, MediaId: 10, EventType: domain.HistoryGrabbed,
			ReleaseTitle: "Movie.1080p", CreatedAt: time.Now(),
		}}, nil)
		media.On("GetById", domain.ID(10)).Return(&domain.Media{
			Id: 10, Name: "Movie", Type: domain.MediaTypeMovie, Cover: &cover,
		}, nil)

		items, err := svc.ListItems(1, 20)
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "Movie", items[0].Media.Name)
		assert.Equal(t, domain.MediaTypeMovie, items[0].Media.Type)
		assert.Equal(t, &cover, items[0].Media.Cover)
		assert.Equal(t, domain.ID(10), items[0].MediaId)
		assert.Nil(t, items[0].PartLabel)
	})

	t.Run("missing media does not fail list", func(t *testing.T) {
		history := new(mockHistoryRepo)
		media := new(mockMediaRepo)
		svc := NewHistoryService(history, media, nil, nil)

		history.On("List", 1, 50).Return([]domain.History{{
			Id: 2, MediaId: 99, EventType: domain.HistoryFailed,
		}}, nil)
		media.On("GetById", domain.ID(99)).Return(nil, domain.ErrMediaNotFound)

		items, err := svc.ListItems(1, 50)
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, domain.ID(99), items[0].Media.Id)
		assert.Empty(t, items[0].Media.Name)
	})

	t.Run("series part label", func(t *testing.T) {
		history := new(mockHistoryRepo)
		media := new(mockMediaRepo)
		parts := new(mockPartRepo)
		groups := new(mockGroupRepo)
		svc := NewHistoryService(history, media, parts, groups)

		epName := "Pilot"
		groupID := domain.ID(7)
		partID := domain.ID(42)
		ep := 1
		history.On("List", 1, 20).Return([]domain.History{{
			Id: 3, MediaId: 5, PartId: &partID, EventType: domain.HistoryImported,
		}}, nil)
		media.On("GetById", domain.ID(5)).Return(&domain.Media{
			Id: 5, Name: "Show", Type: domain.MediaTypeSeries,
		}, nil)
		parts.On("GetById", domain.ID(42)).Return(&domain.Part{
			Id: 42, Name: &epName, GroupOrder: &ep, GroupId: &groupID, MediaId: 5,
		}, nil)
		groups.On("GetByMediaID", domain.ID(5)).Return([]domain.PartGroup{{
			Id: 7, Name: "Season 1", Order: 1, MediaId: 5,
		}}, nil)

		items, err := svc.ListItems(1, 20)
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.NotNil(t, items[0].PartLabel)
		assert.Equal(t, "S01E01 · Pilot", *items[0].PartLabel)
	})

	t.Run("caches media across items", func(t *testing.T) {
		history := new(mockHistoryRepo)
		media := new(mockMediaRepo)
		svc := NewHistoryService(history, media, nil, nil)

		history.On("List", 1, 20).Return([]domain.History{
			{Id: 1, MediaId: 10, EventType: domain.HistoryGrabbed},
			{Id: 2, MediaId: 10, EventType: domain.HistoryImported},
		}, nil)
		media.On("GetById", domain.ID(10)).Return(&domain.Media{
			Id: 10, Name: "Same", Type: domain.MediaTypeBook,
		}, nil).Once()

		items, err := svc.ListItems(1, 20)
		require.NoError(t, err)
		require.Len(t, items, 2)
		assert.Equal(t, "Same", items[0].Media.Name)
		assert.Equal(t, "Same", items[1].Media.Name)
		media.AssertNumberOfCalls(t, "GetById", 1)
	})
}
