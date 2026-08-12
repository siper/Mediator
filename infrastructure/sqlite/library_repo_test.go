package sqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

func TestLibrary_RepoListByTypeAndUpdate(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteLibraryRepository(db)

	movies1 := &domain.Library{Name: "Movies", Path: "/lib/movies", Type: domain.MediaTypeMovie}
	movies2 := &domain.Library{Name: "Anime", Path: "/lib/anime", Type: domain.MediaTypeMovie}
	books := &domain.Library{Name: "Books", Path: "/lib/books", Type: domain.MediaTypeBook}

	require.NoError(t, repo.Add(movies1))
	require.NoError(t, repo.Add(movies2))
	require.NoError(t, repo.Add(books))

	got, err := repo.GetById(movies1.Id)
	require.NoError(t, err)
	assert.Equal(t, "Movies", got.Name)
	assert.Equal(t, "/lib/movies", got.Path)

	byType, err := repo.ListByType(domain.MediaTypeMovie)
	require.NoError(t, err)
	assert.Len(t, byType, 2)

	none, err := repo.ListByType(domain.MediaTypeMusicAlbum)
	require.NoError(t, err)
	assert.Empty(t, none)

	all, err := repo.List()
	require.NoError(t, err)
	assert.Len(t, all, 3)

	movies1.Name = "Films"
	movies1.Path = "/lib/films"
	require.NoError(t, repo.Update(movies1))
	updated, err := repo.GetById(movies1.Id)
	require.NoError(t, err)
	assert.Equal(t, "Films", updated.Name)
	assert.Equal(t, "/lib/films", updated.Path)

	require.NoError(t, repo.Remove(movies2.Id))
	remaining, err := repo.ListByType(domain.MediaTypeMovie)
	require.NoError(t, err)
	assert.Len(t, remaining, 1)
}

func TestLibrary_NotFound(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteLibraryRepository(db)

	_, err := repo.GetById(999)
	assert.ErrorIs(t, err, domain.ErrLibraryNotFound)
	assert.ErrorIs(t, repo.Remove(999), domain.ErrLibraryNotFound)
	assert.ErrorIs(t, repo.Update(&domain.Library{Id: 999, Name: "x", Path: "/x", Type: domain.MediaTypeMovie}), domain.ErrLibraryNotFound)
}
