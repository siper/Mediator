package sqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

func newTestUser(t *testing.T, db DBTX) *domain.User {
	t.Helper()
	repo := NewSQLiteUserRepository(db)
	u := &domain.User{Name: "Test", Email: "test@s.c", PasswordHash: "h", Role: domain.RoleUser, CreatedAt: "2026-01-01T00:00:00Z"}
	require.NoError(t, repo.Add(u))
	return u
}

func TestSQLiteSessionRepository_AddAndGetByHash(t *testing.T) {
	db := newTestDB(t)
	userRepo := NewSQLiteUserRepository(db)
	sessRepo := NewSQLiteSessionRepository(db)

	u := &domain.User{Name: "A", Email: "a@b.c", PasswordHash: "h", Role: domain.RoleUser, CreatedAt: "x"}
	require.NoError(t, userRepo.Add(u))

	s := &domain.Session{UserId: u.Id, RefreshHash: "abc123", ExpiresAt: "2026-12-31T23:59:59Z", CreatedAt: "2026-01-01T00:00:00Z"}
	require.NoError(t, sessRepo.Add(s))
	require.NotZero(t, s.Id)

	got, err := sessRepo.GetByRefreshHash("abc123")
	require.NoError(t, err)
	assert.Equal(t, u.Id, got.UserId)
	assert.False(t, got.Revoked)
	assert.Equal(t, "2026-12-31T23:59:59Z", got.ExpiresAt)
}

func TestSQLiteSessionRepository_GetByRefreshHash_NotFound(t *testing.T) {
	db := newTestDB(t)
	sessRepo := NewSQLiteSessionRepository(db)

	_, err := sessRepo.GetByRefreshHash("nonexistent")
	assert.ErrorIs(t, err, domain.ErrSessionNotFound)
}

func TestSQLiteSessionRepository_RevokeByTokenHash(t *testing.T) {
	db := newTestDB(t)
	u := newTestUser(t, db)
	sessRepo := NewSQLiteSessionRepository(db)

	s := &domain.Session{UserId: u.Id, RefreshHash: "revoke-me", ExpiresAt: "2026-12-31T23:59:59Z", CreatedAt: "x"}
	require.NoError(t, sessRepo.Add(s))

	require.NoError(t, sessRepo.RevokeByTokenHash("revoke-me"))

	got, err := sessRepo.GetByRefreshHash("revoke-me")
	require.NoError(t, err)
	assert.True(t, got.Revoked)
}

func TestSQLiteSessionRepository_RevokeByTokenHash_NotFound(t *testing.T) {
	db := newTestDB(t)
	sessRepo := NewSQLiteSessionRepository(db)

	err := sessRepo.RevokeByTokenHash("nonexistent")
	assert.ErrorIs(t, err, domain.ErrSessionNotFound)
}

func TestSQLiteSessionRepository_RevokeAllForUser(t *testing.T) {
	db := newTestDB(t)
	u := newTestUser(t, db)
	sessRepo := NewSQLiteSessionRepository(db)

	s1 := &domain.Session{UserId: u.Id, RefreshHash: "hash1", ExpiresAt: "2026-12-31T23:59:59Z", CreatedAt: "x"}
	s2 := &domain.Session{UserId: u.Id, RefreshHash: "hash2", ExpiresAt: "2026-12-31T23:59:59Z", CreatedAt: "x"}
	require.NoError(t, sessRepo.Add(s1))
	require.NoError(t, sessRepo.Add(s2))

	require.NoError(t, sessRepo.RevokeAllForUser(u.Id))

	got1, err := sessRepo.GetByRefreshHash("hash1")
	require.NoError(t, err)
	assert.True(t, got1.Revoked)

	got2, err := sessRepo.GetByRefreshHash("hash2")
	require.NoError(t, err)
	assert.True(t, got2.Revoked)
}
