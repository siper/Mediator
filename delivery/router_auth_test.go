package delivery

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/delivery/handlers"
	"stersh.ru/mediator/infrastructure/auth"
	"stersh.ru/mediator/infrastructure/sqlite"
)

func setupAuthRouter(t *testing.T) (*gin.Engine, *http.Cookie) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	require.NoError(t, sqlite.RunMigrations(db))
	t.Cleanup(func() { db.Close() })

	userRepo := sqlite.NewSQLiteUserRepository(db)
	sessRepo := sqlite.NewSQLiteSessionRepository(db)
	oidcRepo := sqlite.NewSQLiteOIDCIdentityRepository(db)
	jwtSvc := auth.NewJWTService("test-secret", 15*time.Minute)
	authSvc := auth.NewAuthService(userRepo, sessRepo, oidcRepo, jwtSvc, nil,
		15*time.Minute, 7*24*time.Hour, true, true, "", "admin")
	authMW := NewAuthMiddleware(jwtSvc, userRepo)
	authH := handlers.NewAuthHandler(authSvc, jwtSvc, userRepo, nil, true, true)

	r := SetupRouter(Deps{
		CoverDir:       t.TempDir(),
		AuthMiddleware: authMW,
		AuthHandler:    authH,
	})

	w := httptest.NewRecorder()
	body := `{"Name":"Test","Email":"t@t.c","Password":"password123"}`
	req := httptest.NewRequest("POST", "/auth/register", strings.NewReader(body))
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, "register should succeed")

	var accessCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "access_token" {
			accessCookie = c
		}
	}
	require.NotNil(t, accessCookie, "access_token cookie must be set after register")
	return r, accessCookie
}

func TestSetupRouter_AuthMe_WithValidCookie(t *testing.T) {
	r, accessCookie := setupAuthRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/auth/me", nil)
	req.AddCookie(accessCookie)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "/auth/me must return 200 when a valid access_token cookie is present")
}

func TestSetupRouter_AuthMe_WithoutCookie(t *testing.T) {
	r, _ := setupAuthRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/auth/me", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "/auth/me must return 401 without a cookie")
}
