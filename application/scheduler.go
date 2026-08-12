package application

import (
	"context"
	"time"
)

type Scheduler struct {
	tick     func(ctx context.Context) error
	interval time.Duration
}

func NewScheduler(interval time.Duration, tick func(ctx context.Context) error) *Scheduler {
	return &Scheduler{tick: tick, interval: interval}
}

func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.tick(ctx)
		}
	}
}
