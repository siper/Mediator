package indexer

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"stersh.ru/mediator/domain"
)

type Builder func(domain.Indexer) ([]domain.ReleaseIndexer, error)

type Source struct {
	repo    domain.IndexerRepository
	builder Builder
	mu      sync.RWMutex
	cur     []domain.ReleaseIndexer
}

func NewSource(repo domain.IndexerRepository, builder Builder) *Source {
	return &Source{repo: repo, builder: builder}
}

func (s *Source) Active() []domain.ReleaseIndexer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cur
}

func (s *Source) Reload() error {
	cfgs, err := s.repo.List()
	if err != nil {
		return err
	}
	next := make([]domain.ReleaseIndexer, 0, len(cfgs))
	for _, ix := range cfgs {
		if !ix.Enabled {
			continue
		}
		clients, err := s.builder(ix)
		if err != nil {
			slog.Warn("failed to build indexer", "name", ix.Name, "type", ix.Type, "err", err)
			continue
		}
		next = append(next, clients...)
	}
	s.mu.Lock()
	s.cur = next
	s.mu.Unlock()
	return nil
}

func (s *Source) Test(ctx context.Context, ix domain.Indexer) error {
	clients, err := s.builder(ix)
	if err != nil {
		return err
	}
	if len(clients) == 0 {
		return fmt.Errorf("indexer %q produced no clients", ix.Name)
	}
	ti, ok := clients[0].(domain.TestableIndexer)
	if !ok {
		return fmt.Errorf("indexer type %q does not support connection testing", ix.Type)
	}
	return ti.TestConnection(ctx)
}

var (
	_ domain.IndexerSource   = (*Source)(nil)
	_ domain.IndexerReloader = (*Source)(nil)
	_ domain.IndexerTester   = (*Source)(nil)
)
