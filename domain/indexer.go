package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	IndexerProwlarr = "prowlarr"
	IndexerJackett  = "jackett"
)

var validIndexerTypes = map[string]bool{
	IndexerProwlarr: true,
	IndexerJackett:  true,
}

type Indexer struct {
	Id       ID
	Name     string
	Type     string
	Settings map[string]string
	Enabled  bool
}

func (i *Indexer) Get(key string) string {
	if i == nil || i.Settings == nil {
		return ""
	}
	return i.Settings[key]
}

func (i *Indexer) ApplyDefaultName() {
	if i == nil {
		return
	}
	i.Name = strings.TrimSpace(i.Name)
	if i.Name != "" {
		return
	}
	switch i.Type {
	case IndexerProwlarr:
		i.Name = "Prowlarr"
	case IndexerJackett:
		i.Name = "Jackett"
	default:
		i.Name = i.Type
	}
}

func (i *Indexer) Validate() error {
	i.ApplyDefaultName()
	if i.Name == "" {
		return ErrEmptyName
	}
	if !validIndexerTypes[i.Type] {
		return fmt.Errorf("%w: %s", ErrInvalidIndexerType, i.Type)
	}
	if i.Get("endpoint") == "" {
		return errors.New("indexer endpoint is required")
	}
	return nil
}

type IndexerRepository interface {
	Add(i *Indexer) error
	GetById(id ID) (*Indexer, error)
	List() ([]Indexer, error)
	Update(i *Indexer) error
	Remove(id ID) error
}

type ReleaseIndexer interface {
	Name() string
	Search(ctx context.Context, query string, cats []int) ([]Release, error)
	RSS(ctx context.Context, cats []int, since time.Time) ([]Release, error)
}

type IndexerSource interface {
	Active() []ReleaseIndexer
}

type IndexerReloader interface {
	Reload() error
}

type TestableIndexer interface {
	TestConnection(ctx context.Context) error
}

type IndexerTester interface {
	Test(ctx context.Context, i Indexer) error
}
