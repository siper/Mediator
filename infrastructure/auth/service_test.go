package auth

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
	"stersh.ru/mediator/infrastructure/sqlite"
)

func newAuthTestDB(t *testing.T) (*sqlite.SQLiteUserRepository, *sqlite.SQLiteSessionRepository, *sqlite.SQLiteOIDCIdentityRepository) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	require.NoError(t, sqlite.RunMigrations(db))
	t.Cleanup(func() { db.Close() })
	return sqlite.NewSQLiteUserRepository(db), sqlite.NewSQLiteSessionRepository(db), sqlite.NewSQLiteOIDCIdentityRepository(db)
}

func newTestService(t *testing.T, enabled bool) (*AuthService, *sqlite.SQLiteUserRepository) {
	t.Helper()
	return newTestServiceOpts(t, enabled, enabled)
}

func newTestServiceOpts(t *testing.T, loginEnabled, registrationEnabled bool) (*AuthService, *sqlite.SQLiteUserRepository) {
	t.Helper()
	userRepo, sessRepo, oidcRepo := newAuthTestDB(t)
	jwtSvc := NewJWTService("test-secret", 15*time.Minute)
	return NewAuthService(userRepo, sessRepo, oidcRepo, jwtSvc, nil, 15*time.Minute, 7*24*time.Hour, loginEnabled, registrationEnabled, "", "admin"), userRepo
}

func TestAuthService_Register_FirstUserIsAdmin(t *testing.T) {
	svc, _ := newTestService(t, true)

	pair, err := svc.Register(context.Background(), "Admin", "admin@test.com", "password123")
	require.NoError(t, err)
	require.NotEmpty(t, pair.AccessToken)
	require.NotEmpty(t, pair.RefreshToken)
	assert.Equal(t, 900, pair.ExpiresIn)

	claims, err := svc.jwt.ValidateToken(pair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "admin", claims.Role)
}

func TestAuthService_Register_SecondUserIsRegular(t *testing.T) {
	svc, _ := newTestService(t, true)

	_, err := svc.Register(context.Background(), "Admin", "a@t.c", "password123")
	require.NoError(t, err)

	pair, err := svc.Register(context.Background(), "User", "b@t.c", "password123")
	require.NoError(t, err)

	claims, err := svc.jwt.ValidateToken(pair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "user", claims.Role)
}

func TestAuthService_Register_ShortPassword(t *testing.T) {
	svc, _ := newTestService(t, true)

	_, err := svc.Register(context.Background(), "A", "a@t.c", "short")
	assert.ErrorIs(t, err, domain.ErrPasswordTooShort)
}

func TestAuthService_Register_RegistrationDisabled(t *testing.T) {
	svc, _ := newTestServiceOpts(t, true, false)

	_, err := svc.Register(context.Background(), "A", "a@t.c", "password123")
	assert.ErrorIs(t, err, domain.ErrRegistrationDisabled)
}

func TestAuthService_Register_WhenLoginDisabled(t *testing.T) {
	svc, _ := newTestServiceOpts(t, false, true)

	pair, err := svc.Register(context.Background(), "A", "a@t.c", "password123")
	require.NoError(t, err)
	require.NotEmpty(t, pair.AccessToken)
}

func TestAuthService_LoginLocal_WhenRegistrationDisabled(t *testing.T) {
	svc, _ := newTestServiceOpts(t, true, true)
	_, err := svc.Register(context.Background(), "A", "a@t.c", "password123")
	require.NoError(t, err)

	svc.registrationEnabled = false

	pair, err := svc.LoginLocal(context.Background(), "a@t.c", "password123")
	require.NoError(t, err)
	require.NotEmpty(t, pair.AccessToken)
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	svc, _ := newTestService(t, true)

	_, err := svc.Register(context.Background(), "A", "a@t.c", "password123")
	require.NoError(t, err)

	_, err = svc.Register(context.Background(), "B", "a@t.c", "password456")
	assert.ErrorIs(t, err, domain.ErrDuplicateEmail)
}

func TestAuthService_LoginLocal_Success(t *testing.T) {
	svc, _ := newTestService(t, true)

	_, err := svc.Register(context.Background(), "Admin", "a@t.c", "password123")
	require.NoError(t, err)

	pair, err := svc.LoginLocal(context.Background(), "a@t.c", "password123")
	require.NoError(t, err)
	require.NotEmpty(t, pair.AccessToken)
}

func TestAuthService_LoginLocal_WrongPassword(t *testing.T) {
	svc, _ := newTestService(t, true)

	_, err := svc.Register(context.Background(), "A", "a@t.c", "password123")
	require.NoError(t, err)

	_, err = svc.LoginLocal(context.Background(), "a@t.c", "wrong")
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthService_LoginLocal_UnknownEmail(t *testing.T) {
	svc, _ := newTestService(t, true)

	_, err := svc.LoginLocal(context.Background(), "nobody@t.c", "password123")
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthService_LoginLocal_Disabled(t *testing.T) {
	svc, _ := newTestServiceOpts(t, false, true)

	_, err := svc.LoginLocal(context.Background(), "a@t.c", "password123")
	assert.ErrorIs(t, err, domain.ErrLoginDisabled)
}

func TestAuthService_Refresh_Success(t *testing.T) {
	svc, _ := newTestService(t, true)

	pair, err := svc.Register(context.Background(), "A", "a@t.c", "password123")
	require.NoError(t, err)

	pair2, err := svc.Refresh(context.Background(), pair.RefreshToken)
	require.NoError(t, err)
	require.NotEmpty(t, pair2.AccessToken)
	require.NotEmpty(t, pair2.RefreshToken)

	claims, err := svc.jwt.ValidateToken(pair2.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "admin", claims.Role)
}

func TestAuthService_Refresh_RevokedTokenFails(t *testing.T) {
	svc, _ := newTestService(t, true)

	pair, err := svc.Register(context.Background(), "A", "a@t.c", "password123")
	require.NoError(t, err)

	require.NoError(t, svc.Logout(context.Background(), pair.RefreshToken))

	_, err = svc.Refresh(context.Background(), pair.RefreshToken)
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthService_Refresh_InvalidToken(t *testing.T) {
	svc, _ := newTestService(t, true)

	_, err := svc.Refresh(context.Background(), "garbage-token")
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthService_Logout(t *testing.T) {
	svc, _ := newTestService(t, true)

	pair, err := svc.Register(context.Background(), "A", "a@t.c", "password123")
	require.NoError(t, err)

	require.NoError(t, svc.Logout(context.Background(), pair.RefreshToken))
}

func TestAuthService_LoginOIDC_NotConfigured(t *testing.T) {
	svc, _ := newTestService(t, true)

	_, err := svc.LoginOIDC(context.Background(), "state", "code", "http://localhost/auth/oidc/callback")
	assert.ErrorIs(t, err, domain.ErrOIDCNotConfigured)
}
