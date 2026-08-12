package sqlite

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

func TestDownloadClient_RepoRoundtrip(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteDownloadClientRepository(db)

	c := &domain.DownloadClient{
		Name:     "qb",
		Type:     domain.DownloadClientQBittorrent,
		Settings: map[string]string{"host": "http://localhost:8080", "username": "admin", "password": "secret"},
		Enabled:  true,
	}
	require.NoError(t, repo.Add(c))
	assert.NotZero(t, c.Id)

	got, err := repo.GetById(c.Id)
	require.NoError(t, err)
	assert.Equal(t, "qb", got.Name)
	assert.Equal(t, domain.DownloadClientQBittorrent, got.Type)
	assert.Equal(t, "http://localhost:8080", got.Get("host"))
	assert.Equal(t, "admin", got.Get("username"))
	assert.Equal(t, "secret", got.Get("password"))
	assert.True(t, got.Enabled)

	got.Settings["password"] = "new"
	require.NoError(t, repo.Update(got))
	all, err := repo.List()
	require.NoError(t, err)
	require.Len(t, all, 1)
	assert.Equal(t, "new", all[0].Get("password"))

	require.NoError(t, repo.Remove(c.Id))
	_, err = repo.GetById(c.Id)
	assert.ErrorIs(t, err, domain.ErrDownloadClientNotFound)
}

func TestQueue_ListActiveAndStateUpdate(t *testing.T) {
	db := newTestDB(t)
	mediaRepo := NewSQLiteMediaRepository(db)
	queueRepo := NewSQLiteQueueRepository(db)

	media, err := mediaRepo.Create("Movie", "", nil, nil, domain.MediaTypeMovie, nil, "tmdb", "1", nil)
	require.NoError(t, err)

	now := time.Now().UTC()

	require.NoError(t, queueRepo.Add(&domain.QueueItem{
		MediaId: media.Id, GrabberName: "torrent", JobID: "h1",
		ReleaseTitle: "R1", State: domain.GrabQueued, Progress: 0, AddedAt: now,
	}))
	require.NoError(t, queueRepo.Add(&domain.QueueItem{
		MediaId: media.Id, GrabberName: "torrent", JobID: "h2", DownloadID: "hash2",
		ReleaseTitle: "R2", State: domain.GrabRunning, Progress: 0.5, AddedAt: now,
	}))
	require.NoError(t, queueRepo.Add(&domain.QueueItem{
		MediaId: media.Id, GrabberName: "torrent", JobID: "h3", DownloadID: "hash3",
		ReleaseTitle: "R3", State: domain.GrabCompleted, Progress: 1.0, AddedAt: now,
	}))

	active, err := queueRepo.ListActive()
	require.NoError(t, err)
	require.Len(t, active, 2)

	item, err := queueRepo.GetById(1)
	require.NoError(t, err)
	assert.Equal(t, domain.GrabQueued, item.State)
	assert.Equal(t, 0.0, item.Progress)
	assert.WithinDuration(t, now, item.AddedAt, time.Second)

	item.State = domain.GrabFailed
	require.NoError(t, queueRepo.Update(item))

	active, err = queueRepo.ListActive()
	require.NoError(t, err)
	assert.Len(t, active, 1)

	list, err := queueRepo.List(1, 50)
	require.NoError(t, err)
	assert.Len(t, list, 3)

	runningItem, err := queueRepo.GetById(2)
	require.NoError(t, err)
	assert.Equal(t, domain.GrabRunning, runningItem.State)
	assert.Equal(t, 0.5, runningItem.Progress)
	assert.Equal(t, "hash2", runningItem.DownloadID)

	completedItem, err := queueRepo.GetById(3)
	require.NoError(t, err)
	assert.Equal(t, domain.GrabCompleted, completedItem.State)
	assert.Equal(t, 1.0, completedItem.Progress)
	assert.Equal(t, "hash3", completedItem.DownloadID)
}

func TestQueue_NotFound(t *testing.T) {
	db := newTestDB(t)
	queueRepo := NewSQLiteQueueRepository(db)

	_, err := queueRepo.GetById(999)
	assert.ErrorIs(t, err, domain.ErrQueueNotFound)

	err = queueRepo.Update(&domain.QueueItem{Id: 999, State: domain.GrabFailed})
	assert.ErrorIs(t, err, domain.ErrQueueNotFound)

	err = queueRepo.Remove(999)
	assert.ErrorIs(t, err, domain.ErrQueueNotFound)
}

func TestHistory_AddAndQuery(t *testing.T) {
	db := newTestDB(t)
	mediaRepo := NewSQLiteMediaRepository(db)
	histRepo := NewSQLiteHistoryRepository(db)

	media, err := mediaRepo.Create("Movie", "", nil, nil, domain.MediaTypeMovie, nil, "tmdb", "1", nil)
	require.NoError(t, err)

	now := time.Now().UTC()
	require.NoError(t, histRepo.Add(&domain.History{
		MediaId: media.Id, EventType: domain.HistoryGrabbed,
		ReleaseTitle: "Movie.1080p.WEB-DL", CreatedAt: now,
	}))
	require.NoError(t, histRepo.Add(&domain.History{
		MediaId: media.Id, EventType: domain.HistoryFailed,
		ReleaseTitle: "Movie.720p", CreatedAt: now,
	}))

	byMedia, err := histRepo.GetByMediaId(media.Id)
	require.NoError(t, err)
	assert.Len(t, byMedia, 2)

	all, err := histRepo.List(1, 50)
	require.NoError(t, err)
	assert.Len(t, all, 2)
	assert.WithinDuration(t, now, all[0].CreatedAt, time.Second)
}
