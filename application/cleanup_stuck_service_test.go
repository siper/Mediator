package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

type cleanupQueueRepo struct {
	mock.Mock
}

func (m *cleanupQueueRepo) Add(q *domain.QueueItem) error {
	return m.Called(q).Error(0)
}
func (m *cleanupQueueRepo) GetById(id domain.ID) (*domain.QueueItem, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.QueueItem), args.Error(1)
}
func (m *cleanupQueueRepo) List(page int, limit int) ([]domain.QueueItem, error) {
	args := m.Called(page, limit)
	return args.Get(0).([]domain.QueueItem), args.Error(1)
}
func (m *cleanupQueueRepo) ListActive() ([]domain.QueueItem, error) {
	args := m.Called()
	return args.Get(0).([]domain.QueueItem), args.Error(1)
}
func (m *cleanupQueueRepo) Update(q *domain.QueueItem) error {
	return m.Called(q).Error(0)
}
func (m *cleanupQueueRepo) Remove(id domain.ID) error {
	return m.Called(id).Error(0)
}

type cleanupHistoryRepo struct {
	mock.Mock
}

func (m *cleanupHistoryRepo) Add(h *domain.History) error {
	return m.Called(h).Error(0)
}
func (m *cleanupHistoryRepo) GetByMediaId(mediaId domain.ID) ([]domain.History, error) {
	args := m.Called(mediaId)
	return args.Get(0).([]domain.History), args.Error(1)
}
func (m *cleanupHistoryRepo) List(page int, limit int) ([]domain.History, error) {
	args := m.Called(page, limit)
	return args.Get(0).([]domain.History), args.Error(1)
}

type cleanupGrabber struct {
	mock.Mock
	name string
}

func (g *cleanupGrabber) Name() string { return g.name }
func (g *cleanupGrabber) Supports(t domain.GrabTarget) bool {
	return g.Called(t).Bool(0)
}
func (g *cleanupGrabber) SupportsProvider(provider string) bool {
	return g.Called(provider).Bool(0)
}
func (g *cleanupGrabber) Submit(ctx context.Context, t domain.GrabTarget) (domain.GrabHandle, error) {
	args := g.Called(ctx, t)
	return args.Get(0).(domain.GrabHandle), args.Error(1)
}
func (g *cleanupGrabber) Status(ctx context.Context, h domain.GrabHandle) (domain.GrabStatus, error) {
	args := g.Called(ctx, h)
	return args.Get(0).(domain.GrabStatus), args.Error(1)
}
func (g *cleanupGrabber) Cancel(ctx context.Context, h domain.GrabHandle) error {
	return g.Called(ctx, h).Error(0)
}

func TestCleanupStuckService_FirstSnapshotNoIncrement(t *testing.T) {
	tasks := new(mockTaskRepo)
	queue := new(cleanupQueueRepo)
	history := new(cleanupHistoryRepo)

	tasks.On("Get", domain.TaskCleanupStuck).Return(&domain.Task{
		Name:     domain.TaskCleanupStuck,
		Settings: map[string]string{domain.TaskSettingPenalty: "3"},
		Data:     "{}",
	}, nil)
	queue.On("ListActive").Return([]domain.QueueItem{
		{Id: 12, Progress: 0.1, State: domain.GrabRunning, GrabberName: "torrent", JobID: "j1", MediaId: 1},
	}, nil)
	tasks.On("UpdateData", domain.TaskCleanupStuck, mock.MatchedBy(func(data string) bool {
		stalls := decodeStallMap(data)
		e, ok := stalls["12"]
		return ok && e.Count == 0 && e.Progress == 0.1 && e.State == string(domain.GrabRunning)
	})).Return(nil)

	svc := NewCleanupStuckService(nil, tasks, queue, history)
	require.NoError(t, svc.Run(context.Background()))
	tasks.AssertExpectations(t)
	history.AssertNotCalled(t, "Add", mock.Anything)
}

func TestCleanupStuckService_IncrementAndReset(t *testing.T) {
	tasks := new(mockTaskRepo)
	queue := new(cleanupQueueRepo)
	history := new(cleanupHistoryRepo)

	tasks.On("Get", domain.TaskCleanupStuck).Return(&domain.Task{
		Name:     domain.TaskCleanupStuck,
		Settings: map[string]string{domain.TaskSettingPenalty: "5"},
		Data:     `{"12":{"progress":0.1,"state":"running","count":1}}`,
	}, nil).Once()
	queue.On("ListActive").Return([]domain.QueueItem{
		{Id: 12, Progress: 0.1, State: domain.GrabRunning, GrabberName: "torrent", JobID: "j1", MediaId: 1},
	}, nil).Once()
	tasks.On("UpdateData", domain.TaskCleanupStuck, mock.MatchedBy(func(data string) bool {
		return decodeStallMap(data)["12"].Count == 2
	})).Return(nil).Once()

	svc := NewCleanupStuckService(nil, tasks, queue, history)
	require.NoError(t, svc.Run(context.Background()))

	tasks.On("Get", domain.TaskCleanupStuck).Return(&domain.Task{
		Name:     domain.TaskCleanupStuck,
		Settings: map[string]string{domain.TaskSettingPenalty: "5"},
		Data:     `{"12":{"progress":0.1,"state":"running","count":2}}`,
	}, nil).Once()
	queue.On("ListActive").Return([]domain.QueueItem{
		{Id: 12, Progress: 0.5, State: domain.GrabRunning, GrabberName: "torrent", JobID: "j1", MediaId: 1},
	}, nil).Once()
	tasks.On("UpdateData", domain.TaskCleanupStuck, mock.MatchedBy(func(data string) bool {
		e := decodeStallMap(data)["12"]
		return e.Count == 0 && e.Progress == 0.5
	})).Return(nil).Once()

	require.NoError(t, svc.Run(context.Background()))
	tasks.AssertExpectations(t)
}

func TestCleanupStuckService_GCStaleEntries(t *testing.T) {
	tasks := new(mockTaskRepo)
	queue := new(cleanupQueueRepo)
	history := new(cleanupHistoryRepo)

	tasks.On("Get", domain.TaskCleanupStuck).Return(&domain.Task{
		Name:     domain.TaskCleanupStuck,
		Settings: map[string]string{domain.TaskSettingPenalty: "3"},
		Data:     `{"12":{"progress":0.1,"state":"running","count":2},"99":{"progress":0,"state":"queued","count":1}}`,
	}, nil)
	queue.On("ListActive").Return([]domain.QueueItem{
		{Id: 12, Progress: 0.2, State: domain.GrabRunning, GrabberName: "torrent", JobID: "j1", MediaId: 1},
	}, nil)
	tasks.On("UpdateData", domain.TaskCleanupStuck, mock.MatchedBy(func(data string) bool {
		stalls := decodeStallMap(data)
		_, stale := stalls["99"]
		e := stalls["12"]
		return !stale && e.Count == 0 && e.Progress == 0.2
	})).Return(nil)

	svc := NewCleanupStuckService(nil, tasks, queue, history)
	require.NoError(t, svc.Run(context.Background()))
	tasks.AssertExpectations(t)
}

func TestCleanupStuckService_RemovesAtPenalty(t *testing.T) {
	tasks := new(mockTaskRepo)
	queue := new(cleanupQueueRepo)
	history := new(cleanupHistoryRepo)
	grabber := &cleanupGrabber{name: "torrent"}

	tasks.On("Get", domain.TaskCleanupStuck).Return(&domain.Task{
		Name:     domain.TaskCleanupStuck,
		Settings: map[string]string{domain.TaskSettingPenalty: "3"},
		Data:     `{"12":{"progress":0.1,"state":"running","count":2}}`,
	}, nil)
	queue.On("ListActive").Return([]domain.QueueItem{
		{Id: 12, MediaId: 7, Progress: 0.1, State: domain.GrabRunning, GrabberName: "torrent", JobID: "j1", DownloadID: "d1", ReleaseTitle: "Film"},
	}, nil)
	grabber.On("Cancel", mock.Anything, domain.GrabHandle{GrabberName: "torrent", JobID: "d1", Name: "Film"}).Return(nil)
	history.On("Add", mock.MatchedBy(func(h *domain.History) bool {
		return h.MediaId == 7 && h.EventType == domain.HistoryFailed && h.Data == "stuck"
	})).Return(nil)
	queue.On("Remove", domain.ID(12)).Return(nil)
	tasks.On("UpdateData", domain.TaskCleanupStuck, "{}").Return(nil)

	svc := NewCleanupStuckService([]domain.Grabber{grabber}, tasks, queue, history)
	require.NoError(t, svc.Run(context.Background()))
	grabber.AssertExpectations(t)
	history.AssertExpectations(t)
	queue.AssertExpectations(t)
}

func TestParseStuckPenalty(t *testing.T) {
	assert.Equal(t, 3, parseStuckPenalty(nil))
	assert.Equal(t, 3, parseStuckPenalty(map[string]string{}))
	assert.Equal(t, 3, parseStuckPenalty(map[string]string{domain.TaskSettingPenalty: "0"}))
	assert.Equal(t, 7, parseStuckPenalty(map[string]string{domain.TaskSettingPenalty: "7"}))
}
