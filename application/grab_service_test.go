package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

type mockQueueRepo struct {
	mock.Mock
}

func (m *mockQueueRepo) Add(q *domain.QueueItem) error { return m.Called(q).Error(0) }
func (m *mockQueueRepo) GetById(id domain.ID) (*domain.QueueItem, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.QueueItem), args.Error(1)
}
func (m *mockQueueRepo) List(page int, limit int) ([]domain.QueueItem, error) {
	args := m.Called(page, limit)
	return args.Get(0).([]domain.QueueItem), args.Error(1)
}
func (m *mockQueueRepo) ListActive() ([]domain.QueueItem, error) {
	args := m.Called()
	return args.Get(0).([]domain.QueueItem), args.Error(1)
}
func (m *mockQueueRepo) Update(q *domain.QueueItem) error { return m.Called(q).Error(0) }
func (m *mockQueueRepo) Remove(id domain.ID) error        { return m.Called(id).Error(0) }

type mockHistoryRepo struct {
	mock.Mock
}

func (m *mockHistoryRepo) Add(h *domain.History) error { return m.Called(h).Error(0) }
func (m *mockHistoryRepo) GetByMediaId(mediaId domain.ID) ([]domain.History, error) {
	args := m.Called(mediaId)
	return args.Get(0).([]domain.History), args.Error(1)
}
func (m *mockHistoryRepo) List(page int, limit int) ([]domain.History, error) {
	args := m.Called(page, limit)
	return args.Get(0).([]domain.History), args.Error(1)
}

type fakeGrabber struct {
	name        string
	handle      domain.GrabHandle
	status      map[string]domain.GrabStatus
	support     bool
	subErr      error
	cancelErr   error
	cancelCalls int
	lastCancel  domain.GrabHandle
	updates     map[string]int
}

func (f *fakeGrabber) Name() string { return f.name }

func (f *fakeGrabber) Supports(t domain.GrabTarget) bool { return f.support }

func (f *fakeGrabber) SupportsProvider(name string) bool { return name == f.name }

func (f *fakeGrabber) Submit(ctx context.Context, t domain.GrabTarget) (domain.GrabHandle, error) {
	if f.subErr != nil {
		return domain.GrabHandle{}, f.subErr
	}
	return f.handle, nil
}

func (f *fakeGrabber) Status(ctx context.Context, h domain.GrabHandle) (domain.GrabStatus, error) {
	if f.status == nil {
		return domain.GrabStatus{State: domain.GrabRunning}, nil
	}
	return f.status[h.JobID], nil
}

func (f *fakeGrabber) Cancel(ctx context.Context, h domain.GrabHandle) error {
	f.cancelCalls++
	f.lastCancel = h
	return f.cancelErr
}

type grabFakeUoW struct {
	queue   domain.QueueRepository
	history domain.HistoryRepository
}

func (u *grabFakeUoW) Run(ctx context.Context, fn func(*domain.Repos) error) error {
	return fn(&domain.Repos{Queue: u.queue, History: u.history})
}

func newGrabUoW(q domain.QueueRepository, h domain.HistoryRepository) domain.UnitOfWork {
	return &grabFakeUoW{queue: q, history: h}
}

func TestGrabService_Grab(t *testing.T) {
	q := new(mockQueueRepo)
	h := new(mockHistoryRepo)
	g := &fakeGrabber{
		name: "torrent", support: true,
		handle: domain.GrabHandle{GrabberName: "torrent", JobID: "h1"},
	}

	q.On("Add", mock.MatchedBy(func(i *domain.QueueItem) bool {
		return i.JobID == "h1" && i.GrabberName == "torrent" && i.State == domain.GrabQueued && i.ReleaseTitle == "Movie.1080p"
	})).Return(nil)
	h.On("Add", mock.MatchedBy(func(x *domain.History) bool {
		return x.EventType == domain.HistoryGrabbed && x.ReleaseTitle == "Movie.1080p"
	})).Return(nil)

	svc := NewGrabService([]domain.Grabber{g}, q, newGrabUoW(q, h))
	item, err := svc.Grab(context.Background(), domain.GrabTarget{
		MediaID: 1,
		Release: &domain.Release{Title: "Movie.1080p", MagnetURI: "magnet:?x"},
	})

	require.NoError(t, err)
	require.NotNil(t, item)
	assert.Equal(t, "h1", item.JobID)
	assert.Equal(t, 0, g.cancelCalls)
	q.AssertExpectations(t)
	h.AssertExpectations(t)
}

func TestGrabService_NoGrabber(t *testing.T) {
	svc := NewGrabService(nil, new(mockQueueRepo), newGrabUoW(new(mockQueueRepo), new(mockHistoryRepo)))
	_, err := svc.Grab(context.Background(), domain.GrabTarget{URL: "http://x"})
	assert.ErrorIs(t, err, domain.ErrNoGrabber)
}

func TestGrabService_QueueInsertFailureCancels(t *testing.T) {
	q := new(mockQueueRepo)
	h := new(mockHistoryRepo)
	g := &fakeGrabber{
		name: "torrent", support: true,
		handle: domain.GrabHandle{GrabberName: "torrent", JobID: "h1"},
	}

	q.On("Add", mock.Anything).Return(assert.AnError)

	svc := NewGrabService([]domain.Grabber{g}, q, newGrabUoW(q, h))
	item, err := svc.Grab(context.Background(), domain.GrabTarget{
		MediaID: 1,
		Release: &domain.Release{Title: "Movie.1080p", MagnetURI: "magnet:?x"},
	})

	require.Error(t, err)
	assert.Nil(t, item)
	assert.Equal(t, 1, g.cancelCalls)
	h.AssertNotCalled(t, "Add", mock.Anything)
}

func TestGrabService_HistoryInsertFailureCancels(t *testing.T) {
	q := new(mockQueueRepo)
	h := new(mockHistoryRepo)
	g := &fakeGrabber{
		name: "torrent", support: true,
		handle: domain.GrabHandle{GrabberName: "torrent", JobID: "h1"},
	}

	q.On("Add", mock.Anything).Return(nil)
	h.On("Add", mock.Anything).Return(assert.AnError)

	svc := NewGrabService([]domain.Grabber{g}, q, newGrabUoW(q, h))
	item, err := svc.Grab(context.Background(), domain.GrabTarget{
		MediaID: 1,
		Release: &domain.Release{Title: "Movie.1080p", MagnetURI: "magnet:?x"},
	})

	require.Error(t, err)
	assert.Nil(t, item)
	assert.Equal(t, 1, g.cancelCalls)
	q.AssertExpectations(t)
	h.AssertExpectations(t)
}

func TestGrabMonitor_Tick(t *testing.T) {
	q := new(mockQueueRepo)
	h := new(mockHistoryRepo)

	completedItem := domain.QueueItem{Id: 1, MediaId: 1, GrabberName: "g", JobID: "job1", State: domain.GrabQueued}
	runningItem := domain.QueueItem{Id: 2, MediaId: 1, GrabberName: "g", JobID: "job2", State: domain.GrabQueued}
	failedItem := domain.QueueItem{Id: 3, MediaId: 1, GrabberName: "g", JobID: "job3", ReleaseTitle: "job3", State: domain.GrabQueued}

	q.On("ListActive").Return([]domain.QueueItem{completedItem, runningItem, failedItem}, nil)

	q.On("Update", mock.MatchedBy(func(i *domain.QueueItem) bool { return i.Id == 1 && i.State == domain.GrabCompleted })).Return(nil)
	q.On("Update", mock.MatchedBy(func(i *domain.QueueItem) bool { return i.Id == 2 && i.State == domain.GrabRunning })).Return(nil)
	q.On("Update", mock.MatchedBy(func(i *domain.QueueItem) bool { return i.Id == 3 && i.State == domain.GrabFailed })).Return(nil)
	h.On("Add", mock.MatchedBy(func(x *domain.History) bool {
		return x.EventType == domain.HistoryFailed && x.ReleaseTitle == "job3"
	})).Return(nil)

	g := &fakeGrabber{
		name: "g", support: true,
		status: map[string]domain.GrabStatus{
			"job1": {State: domain.GrabCompleted, OutputFiles: []string{"/o/f"}},
			"job2": {State: domain.GrabRunning},
			"job3": {State: domain.GrabFailed, Error: "boom"},
		},
	}

	imported := false
	mon := NewGrabMonitor([]domain.Grabber{g}, q, h)
	mon.OnCompleted = func(ctx context.Context, item domain.QueueItem, status domain.GrabStatus) error {
		imported = true
		return nil
	}

	require.NoError(t, mon.Tick(context.Background()))

	q.AssertExpectations(t)
	h.AssertExpectations(t)
	assert.True(t, imported)
}

func TestGrabMonitor_ProgressPersisted(t *testing.T) {
	q := new(mockQueueRepo)
	h := new(mockHistoryRepo)

	runningItem := domain.QueueItem{
		Id: 1, MediaId: 1, GrabberName: "g", JobID: "job1",
		State: domain.GrabRunning, Progress: 0.3,
	}

	q.On("ListActive").Return([]domain.QueueItem{runningItem}, nil)
	q.On("Update", mock.MatchedBy(func(i *domain.QueueItem) bool {
		return i.Id == 1 && i.State == domain.GrabRunning && i.Progress == 0.7
	})).Return(nil)

	g := &fakeGrabber{
		name: "g", support: true,
		status: map[string]domain.GrabStatus{
			"job1": {State: domain.GrabRunning, Progress: 0.7},
		},
	}

	mon := NewGrabMonitor([]domain.Grabber{g}, q, h)
	require.NoError(t, mon.Tick(context.Background()))

	q.AssertExpectations(t)
}

func TestGrabMonitor_ProgressNoChangeNoUpdate(t *testing.T) {
	q := new(mockQueueRepo)
	h := new(mockHistoryRepo)

	runningItem := domain.QueueItem{
		Id: 1, MediaId: 1, GrabberName: "g", JobID: "job1",
		State: domain.GrabRunning, Progress: 0.5,
	}

	q.On("ListActive").Return([]domain.QueueItem{runningItem}, nil)

	g := &fakeGrabber{
		name: "g", support: true,
		status: map[string]domain.GrabStatus{
			"job1": {State: domain.GrabRunning, Progress: 0.5},
		},
	}

	mon := NewGrabMonitor([]domain.Grabber{g}, q, h)
	require.NoError(t, mon.Tick(context.Background()))

	q.AssertExpectations(t)
}

func TestGrabMonitor_ResolvedIDPersisted(t *testing.T) {
	q := new(mockQueueRepo)
	h := new(mockHistoryRepo)

	queuedItem := domain.QueueItem{
		Id: 1, MediaId: 1, GrabberName: "g", JobID: "http://url/download",
		ReleaseTitle: "Movie Name", State: domain.GrabQueued, Progress: 0,
	}

	q.On("ListActive").Return([]domain.QueueItem{queuedItem}, nil)
	q.On("Update", mock.MatchedBy(func(i *domain.QueueItem) bool {
		return i.Id == 1 && i.DownloadID == "resolved-hash" && i.State == domain.GrabRunning
	})).Return(nil)

	g := &fakeGrabber{
		name: "g", support: true,
		status: map[string]domain.GrabStatus{
			"http://url/download": {State: domain.GrabRunning, Progress: 0.3, ResolvedID: "resolved-hash"},
		},
	}

	mon := NewGrabMonitor([]domain.Grabber{g}, q, h)
	require.NoError(t, mon.Tick(context.Background()))

	q.AssertExpectations(t)
}

func TestGrabMonitor_ImportFailureRetries(t *testing.T) {
	q := new(mockQueueRepo)
	h := new(mockHistoryRepo)

	completedItem := domain.QueueItem{
		Id: 1, MediaId: 1, GrabberName: "g", JobID: "job1",
		ReleaseTitle: "Movie", State: domain.GrabQueued, Progress: 0.9,
	}

	q.On("ListActive").Return([]domain.QueueItem{completedItem}, nil)
	q.On("Update", mock.MatchedBy(func(i *domain.QueueItem) bool {
		return i.Id == 1 && i.State == domain.GrabRunning
	})).Return(nil)

	g := &fakeGrabber{
		name: "g", support: true,
		status: map[string]domain.GrabStatus{
			"job1": {State: domain.GrabCompleted, Progress: 1.0, OutputFiles: []string{"/out/f"}},
		},
	}

	mon := NewGrabMonitor([]domain.Grabber{g}, q, h)
	mon.OnCompleted = func(ctx context.Context, item domain.QueueItem, status domain.GrabStatus) error {
		return assert.AnError
	}

	require.NoError(t, mon.Tick(context.Background()))
	q.AssertExpectations(t)
}

func TestGrabService_Remove_ActiveCancelsAndRemoves(t *testing.T) {
	q := new(mockQueueRepo)
	h := new(mockHistoryRepo)
	g := &fakeGrabber{name: "torrent"}

	item := &domain.QueueItem{
		Id: 7, MediaId: 3, GrabberName: "torrent", JobID: "job-7",
		ReleaseTitle: "Movie.1080p", State: domain.GrabRunning,
	}
	q.On("GetById", domain.ID(7)).Return(item, nil)
	h.On("Add", mock.MatchedBy(func(x *domain.History) bool {
		return x.EventType == domain.HistoryFailed && x.Data == "canceled" && x.ReleaseTitle == "Movie.1080p" && x.MediaId == 3
	})).Return(nil)
	q.On("Remove", domain.ID(7)).Return(nil)

	svc := NewGrabService([]domain.Grabber{g}, q, newGrabUoW(q, h))
	require.NoError(t, svc.Remove(context.Background(), 7))

	assert.Equal(t, 1, g.cancelCalls)
	assert.Equal(t, "job-7", g.lastCancel.JobID)
	assert.Equal(t, "torrent", g.lastCancel.GrabberName)
	assert.Equal(t, "Movie.1080p", g.lastCancel.Name)
	q.AssertExpectations(t)
	h.AssertExpectations(t)
}

func TestGrabService_Remove_ActiveUsesDownloadID(t *testing.T) {
	q := new(mockQueueRepo)
	h := new(mockHistoryRepo)
	g := &fakeGrabber{name: "torrent"}

	item := &domain.QueueItem{
		Id: 8, MediaId: 3, GrabberName: "torrent", JobID: "magnet-url",
		DownloadID: "hash-abc", State: domain.GrabQueued,
	}
	q.On("GetById", domain.ID(8)).Return(item, nil)
	h.On("Add", mock.Anything).Return(nil)
	q.On("Remove", domain.ID(8)).Return(nil)

	svc := NewGrabService([]domain.Grabber{g}, q, newGrabUoW(q, h))
	require.NoError(t, svc.Remove(context.Background(), 8))

	assert.Equal(t, "hash-abc", g.lastCancel.JobID)
}

func TestGrabService_Remove_TerminalSkipsCancel(t *testing.T) {
	q := new(mockQueueRepo)
	h := new(mockHistoryRepo)
	g := &fakeGrabber{name: "torrent"}

	item := &domain.QueueItem{
		Id: 9, MediaId: 3, GrabberName: "torrent", JobID: "job-9",
		State: domain.GrabCompleted,
	}
	q.On("GetById", domain.ID(9)).Return(item, nil)
	q.On("Remove", domain.ID(9)).Return(nil)

	svc := NewGrabService([]domain.Grabber{g}, q, newGrabUoW(q, h))
	require.NoError(t, svc.Remove(context.Background(), 9))

	assert.Equal(t, 0, g.cancelCalls)
	h.AssertNotCalled(t, "Add", mock.Anything)
	q.AssertExpectations(t)
}

func TestGrabService_Remove_NotFound(t *testing.T) {
	q := new(mockQueueRepo)
	h := new(mockHistoryRepo)
	q.On("GetById", domain.ID(99)).Return(nil, domain.ErrQueueNotFound)

	svc := NewGrabService(nil, q, newGrabUoW(q, h))
	err := svc.Remove(context.Background(), 99)
	assert.ErrorIs(t, err, domain.ErrQueueNotFound)
	q.AssertNotCalled(t, "Remove", mock.Anything)
}

func TestGrabService_Remove_CancelErrorStillRemoves(t *testing.T) {
	q := new(mockQueueRepo)
	h := new(mockHistoryRepo)
	g := &fakeGrabber{name: "torrent", cancelErr: assert.AnError}

	item := &domain.QueueItem{
		Id: 10, MediaId: 3, GrabberName: "torrent", JobID: "job-10",
		State: domain.GrabRunning,
	}
	q.On("GetById", domain.ID(10)).Return(item, nil)
	h.On("Add", mock.Anything).Return(nil)
	q.On("Remove", domain.ID(10)).Return(nil)

	svc := NewGrabService([]domain.Grabber{g}, q, newGrabUoW(q, h))
	require.NoError(t, svc.Remove(context.Background(), 10))

	assert.Equal(t, 1, g.cancelCalls)
	q.AssertExpectations(t)
	h.AssertExpectations(t)
}
