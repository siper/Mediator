package indexer

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

type fakeRepo struct{ items []domain.Indexer }

func (f *fakeRepo) Add(*domain.Indexer) error                  { return nil }
func (f *fakeRepo) GetById(domain.ID) (*domain.Indexer, error) { return nil, domain.ErrIndexerNotFound }
func (f *fakeRepo) List() ([]domain.Indexer, error)            { return f.items, nil }
func (f *fakeRepo) Update(*domain.Indexer) error               { return nil }
func (f *fakeRepo) Remove(domain.ID) error                     { return nil }

type nameOnly struct{ name string }

func (n *nameOnly) Name() string { return n.name }
func (n *nameOnly) Search(context.Context, string, []int) ([]domain.Release, error) {
	return nil, nil
}
func (n *nameOnly) RSS(context.Context, []int, time.Time) ([]domain.Release, error) {
	return nil, nil
}

func TestSource_ReloadSkipsDisabled(t *testing.T) {
	built := 0
	src := NewSource(&fakeRepo{items: []domain.Indexer{
		{Name: "on", Type: domain.IndexerProwlarr, Enabled: true},
		{Name: "off", Type: domain.IndexerJackett, Enabled: false},
	}}, func(ix domain.Indexer) ([]domain.ReleaseIndexer, error) {
		built++
		return []domain.ReleaseIndexer{&nameOnly{name: ix.Name}}, nil
	})

	require.NoError(t, src.Reload())
	active := src.Active()
	require.Len(t, active, 1)
	assert.Equal(t, "on", active[0].Name())
	assert.Equal(t, 1, built)
}

func TestSource_ReloadReflectsRepoChanges(t *testing.T) {
	repo := &fakeRepo{items: []domain.Indexer{
		{Name: "a", Type: domain.IndexerProwlarr, Enabled: true},
	}}
	src := NewSource(repo, func(ix domain.Indexer) ([]domain.ReleaseIndexer, error) {
		return []domain.ReleaseIndexer{&nameOnly{name: ix.Name}}, nil
	})

	require.NoError(t, src.Reload())
	assert.Len(t, src.Active(), 1)

	repo.items = append(repo.items, domain.Indexer{Name: "b", Type: domain.IndexerJackett, Enabled: true})
	require.NoError(t, src.Reload())
	require.Len(t, src.Active(), 2)
	assert.Equal(t, "b", src.Active()[1].Name())
}

func TestSource_ReloadFlattensMultiClientBuilder(t *testing.T) {
	src := NewSource(&fakeRepo{items: []domain.Indexer{
		{Name: "prowlarr", Type: domain.IndexerProwlarr, Enabled: true},
	}}, func(ix domain.Indexer) ([]domain.ReleaseIndexer, error) {
		return []domain.ReleaseIndexer{
			&nameOnly{name: ix.Name + ": a"},
			&nameOnly{name: ix.Name + ": b"},
			&nameOnly{name: ix.Name + ": c"},
		}, nil
	})

	require.NoError(t, src.Reload())
	active := src.Active()
	require.Len(t, active, 3)
	assert.Equal(t, "prowlarr: c", active[2].Name())
}

func TestSource_ReloadSkipsBuilderError(t *testing.T) {
	src := NewSource(&fakeRepo{items: []domain.Indexer{
		{Name: "broken", Type: domain.IndexerProwlarr, Enabled: true},
	}}, func(domain.Indexer) ([]domain.ReleaseIndexer, error) {
		return nil, context.Canceled
	})

	require.NoError(t, src.Reload())
	assert.Empty(t, src.Active())
}

type fakeTestable struct {
	name string
	err  error
}

func (f *fakeTestable) Name() string { return f.name }
func (f *fakeTestable) Search(context.Context, string, []int) ([]domain.Release, error) {
	return nil, nil
}
func (f *fakeTestable) RSS(context.Context, []int, time.Time) ([]domain.Release, error) {
	return nil, nil
}
func (f *fakeTestable) TestConnection(context.Context) error { return f.err }

func TestSource_Test_OK(t *testing.T) {
	src := NewSource(&fakeRepo{}, func(ix domain.Indexer) ([]domain.ReleaseIndexer, error) {
		return []domain.ReleaseIndexer{&fakeTestable{name: ix.Name}}, nil
	})
	err := src.Test(context.Background(), domain.Indexer{Name: "jackett", Type: domain.IndexerJackett})
	require.NoError(t, err)
}

func TestSource_Test_PropagatesTestConnectionError(t *testing.T) {
	src := NewSource(&fakeRepo{}, func(ix domain.Indexer) ([]domain.ReleaseIndexer, error) {
		return []domain.ReleaseIndexer{&fakeTestable{name: ix.Name, err: context.DeadlineExceeded}}, nil
	})
	err := src.Test(context.Background(), domain.Indexer{Name: "jackett", Type: domain.IndexerJackett})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestSource_Test_PropagatesBuilderError(t *testing.T) {
	src := NewSource(&fakeRepo{}, func(domain.Indexer) ([]domain.ReleaseIndexer, error) {
		return nil, context.Canceled
	})
	err := src.Test(context.Background(), domain.Indexer{Name: "prowlarr", Type: domain.IndexerProwlarr})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestSource_Test_EmptyClients(t *testing.T) {
	src := NewSource(&fakeRepo{}, func(domain.Indexer) ([]domain.ReleaseIndexer, error) {
		return nil, nil
	})
	err := src.Test(context.Background(), domain.Indexer{Name: "prowlarr", Type: domain.IndexerProwlarr})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "produced no clients")
}

func TestSource_Test_NonTestableClient(t *testing.T) {
	src := NewSource(&fakeRepo{}, func(ix domain.Indexer) ([]domain.ReleaseIndexer, error) {
		return []domain.ReleaseIndexer{&nameOnly{name: ix.Name}}, nil
	})
	err := src.Test(context.Background(), domain.Indexer{Name: "prowlarr", Type: domain.IndexerProwlarr})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support connection testing")
}
