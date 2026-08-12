package sqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

func TestSQLiteUserRepository_AddAndGetByEmail(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteUserRepository(db)

	u := &domain.User{
		Name:         "Admin",
		Email:        "admin@test.com",
		PasswordHash: "hashed",
		Role:         domain.RoleAdmin,
		CreatedAt:    "2026-01-01T00:00:00Z",
	}
	require.NoError(t, repo.Add(u))
	require.NotZero(t, u.Id)

	got, err := repo.GetByEmail("admin@test.com")
	require.NoError(t, err)
	assert.Equal(t, "Admin", got.Name)
	assert.True(t, got.IsAdmin())
	assert.Equal(t, "hashed", got.PasswordHash)
	assert.Equal(t, "2026-01-01T00:00:00Z", got.CreatedAt)
}

func TestSQLiteUserRepository_GetByID(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteUserRepository(db)

	u := &domain.User{Name: "U", Email: "u@t.c", PasswordHash: "h", Role: domain.RoleUser, CreatedAt: "x"}
	require.NoError(t, repo.Add(u))

	got, err := repo.GetByID(u.Id)
	require.NoError(t, err)
	assert.Equal(t, u.Email, got.Email)
}

func TestSQLiteUserRepository_GetByEmail_NotFound(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteUserRepository(db)

	_, err := repo.GetByEmail("nobody@test.com")
	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestSQLiteUserRepository_GetByID_NotFound(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteUserRepository(db)

	_, err := repo.GetByID(999)
	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestSQLiteUserRepository_DuplicateEmail(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteUserRepository(db)

	u := &domain.User{Name: "A", Email: "a@b.c", PasswordHash: "h", Role: domain.RoleUser, CreatedAt: "x"}
	require.NoError(t, repo.Add(u))
	err := repo.Add(&domain.User{Name: "B", Email: "a@b.c", PasswordHash: "h", Role: domain.RoleUser, CreatedAt: "x"})
	assert.ErrorIs(t, err, domain.ErrDuplicateEmail)
}

func TestSQLiteUserRepository_Update(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteUserRepository(db)

	u := &domain.User{Name: "A", Email: "a@b.c", PasswordHash: "h", Role: domain.RoleUser, CreatedAt: "x"}
	require.NoError(t, repo.Add(u))

	u.Name = "Updated"
	u.Role = domain.RoleAdmin
	require.NoError(t, repo.Update(u))

	got, err := repo.GetByID(u.Id)
	require.NoError(t, err)
	assert.Equal(t, "Updated", got.Name)
	assert.True(t, got.IsAdmin())
}

func TestSQLiteUserRepository_Update_NotFound(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteUserRepository(db)

	err := repo.Update(&domain.User{Id: 999, Name: "X", Email: "x@y.z", PasswordHash: "h", Role: domain.RoleUser})
	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestSQLiteUserRepository_Remove(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteUserRepository(db)

	u := &domain.User{Name: "A", Email: "a@b.c", PasswordHash: "h", Role: domain.RoleUser, CreatedAt: "x"}
	require.NoError(t, repo.Add(u))

	require.NoError(t, repo.Remove(u.Id))

	_, err := repo.GetByID(u.Id)
	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestSQLiteUserRepository_Remove_NotFound(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteUserRepository(db)

	err := repo.Remove(999)
	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestSQLiteUserRepository_List(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteUserRepository(db)

	for i := 0; i < 5; i++ {
		require.NoError(t, repo.Add(&domain.User{
			Name: "U", Email: string(rune('a'+i)) + "@t.c", PasswordHash: "h", Role: domain.RoleUser, CreatedAt: "x",
		}))
	}

	all, err := repo.List(1, 10)
	require.NoError(t, err)
	assert.Len(t, all, 5)

	page1, err := repo.List(1, 3)
	require.NoError(t, err)
	assert.Len(t, page1, 3)

	page2, err := repo.List(2, 3)
	require.NoError(t, err)
	assert.Len(t, page2, 2)
}

func TestSQLiteUserRepository_Count(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteUserRepository(db)

	n, err := repo.Count()
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	require.NoError(t, repo.Add(&domain.User{Name: "A", Email: "a@b.c", PasswordHash: "h", Role: domain.RoleUser, CreatedAt: "x"}))
	require.NoError(t, repo.Add(&domain.User{Name: "B", Email: "b@b.c", PasswordHash: "h", Role: domain.RoleUser, CreatedAt: "x"}))

	n, err = repo.Count()
	require.NoError(t, err)
	assert.Equal(t, 2, n)
}
