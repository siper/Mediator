package domain

import "context"

type Repos struct {
	Media   MediaRepository
	Part    PartRepository
	Group   PartGroupRepository
	History HistoryRepository
	Queue   QueueRepository
}

type UnitOfWork interface {
	Run(ctx context.Context, fn func(*Repos) error) error
}
