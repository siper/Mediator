package sqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

func TestSQLiteMediaRequestRepository_CRUD(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteMediaRequestRepository(db)

	userRepo := NewSQLiteUserRepository(db)
	require.NoError(t, userRepo.Add(&domain.User{Name: "U", Email: "u@t.c", PasswordHash: "h", Role: domain.RoleUser, CreatedAt: "x"}))
	require.NoError(t, userRepo.Add(&domain.User{Name: "A", Email: "a@t.c", PasswordHash: "h", Role: domain.RoleAdmin, CreatedAt: "x"}))

	libRepo := NewSQLiteLibraryRepository(db)
	require.NoError(t, libRepo.Add(&domain.Library{Name: "L", Path: "movies", Type: domain.MediaTypeMovie, Settings: map[string]string{}}))

	qualityRepo := NewSQLiteQualityProfileRepository(db)
	require.NoError(t, qualityRepo.Add(&domain.QualityProfile{Name: "Movies", Type: domain.MediaTypeMovie, Allowed: []domain.Quality{{Kind: "video_movie", Name: "1080p"}}, Cutoff: domain.Quality{Kind: "video_movie", Name: "1080p"}}))

	profileID := domain.ID(1)
	folder := "My Folder"
	req := &domain.MediaRequest{
		UserId:           1,
		Provider:         "tmdb",
		ExternalID:       "42",
		Title:            "Matrix",
		Cover:            "/covers/matrix.jpg",
		Type:             domain.MediaTypeMovie,
		LibraryID:        1,
		QualityProfileID: &profileID,
		Folder:           &folder,
		Status:           domain.MediaRequestPending,
		CreatedAt:        "2024-01-01T00:00:00Z",
	}
	require.NoError(t, repo.Add(req))
	assert.Greater(t, req.Id, domain.ID(0))

	got, err := repo.GetByID(req.Id)
	require.NoError(t, err)
	assert.Equal(t, "tmdb", got.Provider)
	assert.Equal(t, "42", got.ExternalID)
	assert.Equal(t, "Matrix", got.Title)
	assert.Equal(t, "/covers/matrix.jpg", got.Cover)
	assert.Equal(t, domain.MediaTypeMovie, got.Type)
	require.NotNil(t, got.QualityProfileID)
	assert.Equal(t, profileID, *got.QualityProfileID)
	require.NotNil(t, got.Folder)
	assert.Equal(t, "My Folder", *got.Folder)
	assert.Equal(t, domain.MediaRequestPending, got.Status)

	exists, err := repo.ExistsByProviderExternal("tmdb", "42")
	require.NoError(t, err)
	assert.True(t, exists)

	exists2, err := repo.ExistsByProviderExternal("tmdb", "999")
	require.NoError(t, err)
	assert.False(t, exists2)

	now := "2024-02-01T00:00:00Z"
	approvedBy := domain.ID(2)
	notes := "looks good"
	mediaRepo := NewSQLiteMediaRepository(db)
	libID := domain.ID(1)
	media, err := mediaRepo.Create("Movie", "", nil, nil, domain.MediaTypeMovie, &libID, "tmdb", "42", nil)
	require.NoError(t, err)
	req.Status = domain.MediaRequestApproved
	req.ApprovedAt = &now
	req.ApprovedBy = &approvedBy
	req.MediaID = &media.Id
	req.Notes = &notes
	require.NoError(t, repo.Update(req))

	updated, err := repo.GetByID(req.Id)
	require.NoError(t, err)
	assert.Equal(t, domain.MediaRequestApproved, updated.Status)
	require.NotNil(t, updated.ApprovedAt)
	assert.Equal(t, "2024-02-01T00:00:00Z", *updated.ApprovedAt)
	require.NotNil(t, updated.ApprovedBy)
	assert.Equal(t, domain.ID(2), *updated.ApprovedBy)
	require.NotNil(t, updated.MediaID)
	assert.Equal(t, media.Id, *updated.MediaID)
	require.NotNil(t, updated.Notes)
	assert.Equal(t, "looks good", *updated.Notes)

	all, err := repo.ListAll(1, 20)
	require.NoError(t, err)
	assert.Len(t, all, 1)

	byUser, err := repo.ListByUser(1, 1, 20)
	require.NoError(t, err)
	assert.Len(t, byUser, 1)

	require.NoError(t, repo.Remove(req.Id))
	_, err = repo.GetByID(req.Id)
	assert.ErrorIs(t, err, domain.ErrRequestNotFound)
}

func TestSQLiteMediaRequestRepository_GetByID_NotFound(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteMediaRequestRepository(db)
	_, err := repo.GetByID(999)
	assert.ErrorIs(t, err, domain.ErrRequestNotFound)
}

func TestSQLiteSettingRepository(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteSettingRepository(db)

	val, err := repo.Get(domain.SettingAutoApproveRequests)
	require.NoError(t, err)
	assert.Equal(t, "0", val)

	require.NoError(t, repo.Set(domain.SettingAutoApproveRequests, "1"))
	val, err = repo.Get(domain.SettingAutoApproveRequests)
	require.NoError(t, err)
	assert.Equal(t, "1", val)

	all, err := repo.List()
	require.NoError(t, err)
	found := false
	for _, s := range all {
		if s.Key == domain.SettingAutoApproveRequests {
			assert.Equal(t, "1", s.Value)
			found = true
		}
	}
	assert.True(t, found)

	_, err = repo.Get("does_not_exist")
	assert.ErrorIs(t, err, domain.ErrSettingNotFound)
}

func TestSQLiteMediaRepository_GetByProviderExternal(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteMediaRepository(db)

	m, err := repo.Create("Movie", "", nil, nil, domain.MediaTypeMovie, nil, "tmdb", "42", nil)
	require.NoError(t, err)
	require.NotNil(t, m)

	got, err := repo.GetByProviderExternal("tmdb", "42")
	require.NoError(t, err)
	assert.Equal(t, "Movie", got.Name)
	assert.Equal(t, "tmdb", got.ProviderID)
	assert.Equal(t, "42", got.ExternalID)

	_, err = repo.GetByProviderExternal("tmdb", "999")
	assert.ErrorIs(t, err, domain.ErrMediaNotFound)
}
