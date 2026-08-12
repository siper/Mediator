package source

import (
	"context"
	"fmt"
	"sync"

	"stersh.ru/mediator/domain"
)

type ProviderBuilder func(domain.Source) (domain.MediaProvider, error)

type Source struct {
	repo    domain.SourceRepository
	builder ProviderBuilder
	mu      sync.RWMutex
	cur     []domain.MediaProvider
}

func New(repo domain.SourceRepository, builder ProviderBuilder) *Source {
	return &Source{repo: repo, builder: builder}
}

func (s *Source) Active() []domain.MediaProvider {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cur
}

func (s *Source) Reload() error {
	configs, err := s.repo.List()
	if err != nil {
		return err
	}
	next := make([]domain.MediaProvider, 0, len(configs))
	for _, cfg := range configs {
		if !cfg.Enabled {
			continue
		}
		p, err := s.builder(cfg)
		if err != nil {
			continue
		}
		next = append(next, p)
	}
	s.mu.Lock()
	s.cur = next
	s.mu.Unlock()
	return nil
}

func (s *Source) Test(ctx context.Context, src domain.Source) error {
	p, err := s.builder(src)
	if err != nil {
		return err
	}
	tp, ok := p.(domain.TestableProvider)
	if !ok {
		return fmt.Errorf("source type %q does not support connection testing", src.Type)
	}
	return tp.TestConnection(ctx)
}

var (
	_ domain.MetadataSource  = (*Source)(nil)
	_ domain.SourceReloader   = (*Source)(nil)
	_ domain.SourceTester     = (*Source)(nil)
)
