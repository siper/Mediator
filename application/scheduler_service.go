package application

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"stersh.ru/mediator/domain"
)

type JobHandler func(ctx context.Context) error

type SchedulerService struct {
	repo      domain.TaskRepository
	handlers  map[string]JobHandler
	rootCtx   context.Context

	mu        sync.Mutex
	running   map[string]struct{}
	tickEvery time.Duration
}

func NewSchedulerService(repo domain.TaskRepository) *SchedulerService {
	return &SchedulerService{
		repo:      repo,
		handlers:  make(map[string]JobHandler),
		running:   make(map[string]struct{}),
		tickEvery: 30 * time.Second,
	}
}

func (s *SchedulerService) WithContext(ctx context.Context) *SchedulerService {
	s.rootCtx = ctx
	return s
}

func (s *SchedulerService) Register(name string, handler JobHandler, defaultInterval string) error {
	return s.RegisterTask(domain.Task{Name: name, Interval: defaultInterval, Enabled: true}, handler)
}

func (s *SchedulerService) RegisterTask(task domain.Task, handler JobHandler) error {
	s.handlers[task.Name] = handler
	if _, err := s.repo.Get(task.Name); err == domain.ErrTaskNotFound {
		if task.Data == "" {
			task.Data = "{}"
		}
		return s.repo.Create(task)
	} else if err != nil {
		return err
	}
	return nil
}

func (s *SchedulerService) Run(ctx context.Context) {
	s.rootCtx = ctx
	ticker := time.NewTicker(s.tickEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *SchedulerService) jobContext() context.Context {
	if s.rootCtx != nil {
		return s.rootCtx
	}
	return context.Background()
}

func (s *SchedulerService) tick(ctx context.Context) {
	tasks, err := s.repo.List()
	if err != nil {
		slog.Warn("scheduler: failed to list tasks", "err", err)
		return
	}
	now := time.Now().UTC()
	for _, t := range tasks {
		if !t.Enabled {
			continue
		}
		handler, ok := s.handlers[t.Name]
		if !ok {
			continue
		}
		if t.NextRun != nil && now.Before(*t.NextRun) {
			continue
		}
		s.runJob(ctx, t.Name, handler, t.Interval)
	}
}

func (s *SchedulerService) Trigger(_ context.Context, name string) error {
	if _, ok := s.handlers[name]; !ok {
		return domain.ErrTaskNotFound
	}
	t, err := s.repo.Get(name)
	if err != nil {
		return err
	}
	s.runJob(s.jobContext(), name, s.handlers[name], t.Interval)
	return nil
}

func (s *SchedulerService) SetConfig(name string, interval string, enabled bool, settings map[string]string) error {
	t, err := s.repo.Get(name)
	if err != nil {
		return err
	}
	if settings == nil {
		settings = t.Settings
	}
	if settings == nil {
		settings = map[string]string{}
	}
	return s.repo.Update(name, interval, enabled, settings)
}

func (s *SchedulerService) runJob(ctx context.Context, name string, handler JobHandler, interval string) {
	if !s.tryAcquire(name) {
		return
	}
	go func() {
		defer s.release(name)
		_ = s.repo.UpdateRun(name, domain.TaskRunning, "", time.Now().UTC(), time.Now().UTC())
		err := handler(ctx)
		lastRun := time.Now().UTC()
		nextRun := lastRun.Add(parseSchedulerInterval(interval))
		if err != nil {
			slog.Warn("scheduler: task failed", "name", name, "err", err)
			_ = s.repo.UpdateRun(name, domain.TaskError, err.Error(), lastRun, nextRun)
			return
		}
		_ = s.repo.UpdateRun(name, domain.TaskOK, "", lastRun, nextRun)
	}()
}

func (s *SchedulerService) tryAcquire(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.running[name]; ok {
		return false
	}
	s.running[name] = struct{}{}
	return true
}

func (s *SchedulerService) release(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.running, name)
}

func parseSchedulerInterval(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return 24 * time.Hour
	}
	return d
}
