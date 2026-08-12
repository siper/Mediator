package application

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"stersh.ru/mediator/domain"
)

type mockTaskRepo struct {
	mock.Mock
}

func (m *mockTaskRepo) List() ([]domain.Task, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Task), args.Error(1)
}

func (m *mockTaskRepo) Get(name string) (*domain.Task, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Task), args.Error(1)
}

func (m *mockTaskRepo) Create(task domain.Task) error {
	return m.Called(task).Error(0)
}

func (m *mockTaskRepo) Update(name string, interval string, enabled bool, settings map[string]string) error {
	return m.Called(name, interval, enabled, settings).Error(0)
}

func (m *mockTaskRepo) UpdateData(name string, data string) error {
	return m.Called(name, data).Error(0)
}

func (m *mockTaskRepo) UpdateRun(name string, status domain.TaskStatus, errMsg string, lastRun time.Time, nextRun time.Time) error {
	return m.Called(name, status, errMsg, lastRun, nextRun).Error(0)
}

func TestSchedulerService_RegisterCreatesIfMissing(t *testing.T) {
	repo := new(mockTaskRepo)
	repo.On("Get", "refresh-metadata").Return(nil, domain.ErrTaskNotFound)
	repo.On("Create", mock.MatchedBy(func(t domain.Task) bool {
		return t.Name == "refresh-metadata" && t.Interval == "24h" && t.Enabled
	})).Return(nil)

	sched := NewSchedulerService(repo)
	err := sched.Register("refresh-metadata", func(ctx context.Context) error { return nil }, "24h")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestSchedulerService_TriggerRunsHandler(t *testing.T) {
	repo := new(mockTaskRepo)
	repo.On("Get", "refresh-metadata").Return(&domain.Task{Name: "refresh-metadata", Interval: "1h"}, nil)
	repo.On("UpdateRun", "refresh-metadata", domain.TaskRunning, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateRun", "refresh-metadata", domain.TaskOK, "", mock.Anything, mock.Anything).Return(nil).Once()

	sched := NewSchedulerService(repo)
	var ran atomic.Int32
	done := make(chan struct{})
	_ = sched.Register("refresh-metadata", func(ctx context.Context) error {
		ran.Add(1)
		close(done)
		return nil
	}, "1h")

	err := sched.Trigger(context.Background(), "refresh-metadata")
	assert.NoError(t, err)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handler did not run")
	}
	assert.Equal(t, int32(1), ran.Load())
}

func TestSchedulerService_AntiOverlap(t *testing.T) {
	repo := new(mockTaskRepo)
	repo.On("Get", "grab-missing").Return(&domain.Task{Name: "grab-missing", Interval: "1h"}, nil)
	repo.On("UpdateRun", "grab-missing", domain.TaskRunning, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateRun", "grab-missing", domain.TaskOK, "", mock.Anything, mock.Anything).Return(nil)

	sched := NewSchedulerService(repo)
	release := make(chan struct{})
	var runs atomic.Int32
	_ = sched.Register("grab-missing", func(ctx context.Context) error {
		runs.Add(1)
		<-release
		return nil
	}, "1h")

	_ = sched.Trigger(context.Background(), "grab-missing")
	_ = sched.Trigger(context.Background(), "grab-missing")
	close(release)
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, int32(1), runs.Load(), "second trigger should be skipped while first is running")
}

func TestSchedulerService_SetConfig(t *testing.T) {
	repo := new(mockTaskRepo)
	repo.On("Get", "refresh-metadata").Return(&domain.Task{Name: "refresh-metadata", Settings: map[string]string{"k": "v"}}, nil)
	repo.On("Update", "refresh-metadata", "30m", false, map[string]string{"k": "v"}).Return(nil)

	sched := NewSchedulerService(repo)
	err := sched.SetConfig("refresh-metadata", "30m", false, nil)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestSchedulerService_TickSkipsDisabled(t *testing.T) {
	repo := new(mockTaskRepo)
	repo.On("Get", "refresh-metadata").Return(&domain.Task{Name: "refresh-metadata", Interval: "1h"}, nil)
	repo.On("List").Return([]domain.Task{{Name: "refresh-metadata", Enabled: false}}, nil)

	sched := NewSchedulerService(repo)
	var ran atomic.Int32
	_ = sched.Register("refresh-metadata", func(ctx context.Context) error {
		ran.Add(1)
		return nil
	}, "1h")

	sched.tick(context.Background())
	assert.Equal(t, int32(0), ran.Load())
}

func TestSchedulerService_TickRunsDueTasks(t *testing.T) {
	past := time.Now().Add(-time.Minute).UTC()
	repo := new(mockTaskRepo)
	repo.On("Get", "refresh-metadata").Return(&domain.Task{Name: "refresh-metadata", Interval: "1h"}, nil)
	repo.On("List").Return([]domain.Task{
		{Name: "refresh-metadata", Enabled: true, Interval: "1h", NextRun: &past},
	}, nil)
	repo.On("UpdateRun", "refresh-metadata", domain.TaskRunning, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateRun", "refresh-metadata", domain.TaskOK, "", mock.Anything, mock.Anything).Return(nil)

	sched := NewSchedulerService(repo)
	var runs atomic.Int32
	_ = sched.Register("refresh-metadata", func(ctx context.Context) error {
		runs.Add(1)
		return nil
	}, "1h")

	sched.tick(context.Background())
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(1), runs.Load())
}
