package sqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

func TestIndexer_RepoRoundtrip(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteIndexerRepository(db)

	ix := &domain.Indexer{
		Name:     "jackett",
		Type:     domain.IndexerJackett,
		Settings: map[string]string{"endpoint": "http://localhost:9117/api/v2.0/indexers/all/results/torznab", "api_key": "secret"},
		Enabled:  true,
	}
	require.NoError(t, repo.Add(ix))
	assert.NotZero(t, ix.Id)

	got, err := repo.GetById(ix.Id)
	require.NoError(t, err)
	assert.Equal(t, "jackett", got.Name)
	assert.Equal(t, domain.IndexerJackett, got.Type)
	assert.Equal(t, "http://localhost:9117/api/v2.0/indexers/all/results/torznab", got.Get("endpoint"))
	assert.Equal(t, "secret", got.Get("api_key"))
	assert.True(t, got.Enabled)

	all, err := repo.List()
	require.NoError(t, err)
	require.Len(t, all, 1)

	got.Settings["api_key"] = "updated"
	got.Enabled = false
	require.NoError(t, repo.Update(got))

	after, err := repo.GetById(ix.Id)
	require.NoError(t, err)
	assert.Equal(t, "updated", after.Get("api_key"))
	assert.False(t, after.Enabled)

	require.NoError(t, repo.Remove(ix.Id))
	_, err = repo.GetById(ix.Id)
	assert.ErrorIs(t, err, domain.ErrIndexerNotFound)
}
