package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		require.NoError(t, err)
	}
	require.NoError(t, RunMigrations(db))
	t.Cleanup(func() { db.Close() })
	return db
}

func TestSQLitePartRepository_MonitoredAndWanted(t *testing.T) {
	db := newTestDB(t)
	mediaRepo := NewSQLiteMediaRepository(db)
	partRepo := NewSQLitePartRepository(db)

	media, err := mediaRepo.Create("Movie", "", nil, nil, domain.MediaTypeMovie, nil, "tmdb", "1", nil)
	require.NoError(t, err)

	path := "/lib/movie.mkv"
	p1 := &domain.Part{MediaId: media.Id, Monitored: true}
	p2 := &domain.Part{MediaId: media.Id, Monitored: true, Path: &path}
	p3 := &domain.Part{MediaId: media.Id, Monitored: false}
	require.NoError(t, partRepo.Add(p1))
	require.NoError(t, partRepo.Add(p2))
	require.NoError(t, partRepo.Add(p3))

	got, err := partRepo.GetById(p1.Id)
	require.NoError(t, err)
	assert.True(t, got.Monitored)

	all, err := partRepo.GetByMediaId(media.Id)
	require.NoError(t, err)
	assert.Len(t, all, 3)

	wanted, err := partRepo.GetWanted(1, 20)
	require.NoError(t, err)
	require.Len(t, wanted, 1)
	assert.Equal(t, p1.Id, wanted[0].Id)

	p1.Monitored = true
	p1.Path = &path
	require.NoError(t, partRepo.Update(p1))

	none, err := partRepo.GetWanted(1, 20)
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestSQLitePartRepository_GetById_NotFound(t *testing.T) {
	db := newTestDB(t)
	partRepo := NewSQLitePartRepository(db)

	_, err := partRepo.GetById(999)
	assert.ErrorIs(t, err, domain.ErrPartNotFound)
}

func TestTxManager_Commit(t *testing.T) {
	db := newTestDB(t)
	tm := NewTxManager(db)

	err := tm.Run(context.Background(), func(repos *domain.Repos) error {
		_, err := repos.Media.Create("In Tx", "", nil, nil, domain.MediaTypeMovie, nil, "tmdb", "42", nil)
		if err != nil {
			return err
		}
		return repos.Part.Add(&domain.Part{MediaId: 1, Monitored: true})
	})
	require.NoError(t, err)

	media, err := NewSQLiteMediaRepository(db).GetById(1)
	require.NoError(t, err)
	assert.Equal(t, "In Tx", media.Name)

	parts, err := NewSQLitePartRepository(db).GetByMediaId(1)
	require.NoError(t, err)
	assert.Len(t, parts, 1)
	assert.True(t, parts[0].Monitored)
}

func TestTxManager_Rollback(t *testing.T) {
	db := newTestDB(t)
	tm := NewTxManager(db)

	boom := errors.New("boom")
	err := tm.Run(context.Background(), func(repos *domain.Repos) error {
		if _, err := repos.Media.Create("Will Rollback", "", nil, nil, domain.MediaTypeMovie, nil, "tmdb", "1", nil); err != nil {
			return err
		}
		return boom
	})
	assert.ErrorIs(t, err, boom)

	_, err = NewSQLiteMediaRepository(db).GetById(1)
	assert.ErrorIs(t, err, domain.ErrMediaNotFound)
}

func TestTxManager_RecoverPanic(t *testing.T) {
	db := newTestDB(t)
	tm := NewTxManager(db)

	assert.Panics(t, func() {
		_ = tm.Run(context.Background(), func(repos *domain.Repos) error {
			_, _ = repos.Media.Create("Panic", "", nil, nil, domain.MediaTypeMovie, nil, "tmdb", "1", nil)
			panic("oops")
		})
	})

	_, err := NewSQLiteMediaRepository(db).GetById(1)
	assert.ErrorIs(t, err, domain.ErrMediaNotFound)
}

func TestSQLiteMediaRepository_Folder(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteMediaRepository(db)

	folder := "Custom Folder"
	media, err := repo.Create("Movie", "", &folder, nil, domain.MediaTypeMovie, nil, "tmdb", "1", nil)
	require.NoError(t, err)
	require.NotNil(t, media.Folder)
	assert.Equal(t, "Custom Folder", *media.Folder)

	got, err := repo.GetById(media.Id)
	require.NoError(t, err)
	require.NotNil(t, got.Folder)
	assert.Equal(t, "Custom Folder", *got.Folder)

	gotNil, err := repo.Create("No Folder", "", nil, nil, domain.MediaTypeMovie, nil, "tmdb", "2", nil)
	require.NoError(t, err)
	assert.Nil(t, gotNil.Folder)

	byID, err := repo.GetById(gotNil.Id)
	require.NoError(t, err)
	assert.Nil(t, byID.Folder)

	mt := domain.MediaTypeMovie
	all, err := repo.GetPaged(1, 20, &mt)
	require.NoError(t, err)
	require.Len(t, all, 2)
}
