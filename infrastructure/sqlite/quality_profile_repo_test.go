package sqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

func TestQualityProfile_RepoRoundtrip(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteQualityProfileRepository(db)

	p := &domain.QualityProfile{
		Name:    "HD",
		Type:    domain.MediaTypeMovie,
		Allowed: []domain.Quality{{Kind: domain.QualityKindVideoMovie, Name: "1080p WEB-DL"}, {Kind: domain.QualityKindVideoMovie, Name: "2160p BluRay"}},
		Cutoff:  domain.Quality{Kind: domain.QualityKindVideoMovie, Name: "1080p WEB-DL"},
	}
	require.NoError(t, repo.Add(p))
	assert.NotZero(t, p.Id)

	got, err := repo.GetById(p.Id)
	require.NoError(t, err)
	assert.Equal(t, "HD", got.Name)
	assert.Equal(t, domain.MediaTypeMovie, got.Type)
	assert.Equal(t, p.Allowed, got.Allowed)
	assert.Equal(t, p.Cutoff, got.Cutoff)
}

func TestQualityProfile_ListByType(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteQualityProfileRepository(db)

	require.NoError(t, repo.Add(&domain.QualityProfile{
		Name: "MovieHD", Type: domain.MediaTypeMovie,
		Allowed: []domain.Quality{{Kind: domain.QualityKindVideoMovie, Name: "1080p WEB-DL"}},
	}))
	require.NoError(t, repo.Add(&domain.QualityProfile{
		Name: "SeriesHD", Type: domain.MediaTypeSeries,
		Allowed: []domain.Quality{{Kind: domain.QualityKindVideoSeries, Name: "1080p WEB-DL"}},
	}))

	movies, err := repo.ListByType(domain.MediaTypeMovie)
	require.NoError(t, err)
	require.Len(t, movies, 1)
	assert.Equal(t, "MovieHD", movies[0].Name)

	all, err := repo.List()
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestQualityProfile_UpdateAndRemove(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteQualityProfileRepository(db)

	p := &domain.QualityProfile{
		Name: "Old", Type: domain.MediaTypeMovie,
		Allowed: []domain.Quality{{Kind: domain.QualityKindVideoMovie, Name: "720p WEB-DL"}},
	}
	require.NoError(t, repo.Add(p))

	p.Name = "New"
	p.Cutoff = domain.Quality{Kind: domain.QualityKindVideoMovie, Name: "1080p WEB-DL"}
	require.NoError(t, repo.Update(p))

	got, err := repo.GetById(p.Id)
	require.NoError(t, err)
	assert.Equal(t, "New", got.Name)
	assert.Equal(t, "1080p WEB-DL", got.Cutoff.Name)

	require.NoError(t, repo.Remove(p.Id))
	_, err = repo.GetById(p.Id)
	assert.Error(t, err)
}

func TestQualityProfile_Validate(t *testing.T) {
	t.Run("rejects cross-kind quality in allowed", func(t *testing.T) {
		p := &domain.QualityProfile{
			Name:    "Bad",
			Type:    domain.MediaTypeMovie,
			Allowed: []domain.Quality{{Kind: domain.QualityKindBook, Name: "epub"}},
		}
		assert.ErrorIs(t, p.Validate(), domain.ErrInvalidQuality)
	})

	t.Run("rejects cross-video-kind quality in allowed", func(t *testing.T) {
		p := &domain.QualityProfile{
			Name:    "Bad",
			Type:    domain.MediaTypeMovie,
			Allowed: []domain.Quality{{Kind: domain.QualityKindVideoSeries, Name: "1080p WEB-DL"}},
		}
		assert.ErrorIs(t, p.Validate(), domain.ErrInvalidQuality)
	})

	t.Run("rejects unknown quality name", func(t *testing.T) {
		p := &domain.QualityProfile{
			Name:    "Bad",
			Type:    domain.MediaTypeMovie,
			Allowed: []domain.Quality{{Kind: domain.QualityKindVideoMovie, Name: "8k"}},
		}
		assert.ErrorIs(t, p.Validate(), domain.ErrInvalidQuality)
	})

	t.Run("rejects book profile type", func(t *testing.T) {
		p := &domain.QualityProfile{
			Name:    "Books",
			Type:    domain.MediaTypeBook,
			Allowed: []domain.Quality{{Kind: domain.QualityKindBook, Name: "epub"}},
		}
		assert.ErrorIs(t, p.Validate(), domain.ErrProfileTypeNotAllowed)
	})

	t.Run("accepts music profile type", func(t *testing.T) {
		p := &domain.QualityProfile{
			Name:    "Music",
			Type:    domain.MediaTypeMusicAlbum,
			Allowed: []domain.Quality{{Kind: domain.QualityKindAudio, Name: "flac"}},
		}
		assert.NoError(t, p.Validate())
	})

	t.Run("accepts valid movie profile without cutoff", func(t *testing.T) {
		p := &domain.QualityProfile{
			Name:    "OK",
			Type:    domain.MediaTypeMovie,
			Allowed: []domain.Quality{{Kind: domain.QualityKindVideoMovie, Name: "1080p WEB-DL"}},
		}
		assert.NoError(t, p.Validate())
	})
}
