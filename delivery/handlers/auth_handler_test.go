package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
	"stersh.ru/mediator/infrastructure/auth"
	"stersh.ru/mediator/infrastructure/sqlite"
)

type fakeAuthProvider struct {
	loginURL  string
	result    domain.OIDCCallbackResult
	callbackErr error
	enabledFlag bool
}

func (f *fakeAuthProvider) LoginURL(state, redirectURI string) string {
	if f.loginURL != "" {
		return f.loginURL
	}
	return "https://idp.example/auth?state=" + state + "&redirect_uri=" + redirectURI
}
func (f *fakeAuthProvider) Callback(ctx context.Context, code, state, redirectURI string) (domain.OIDCCallbackResult, error) {
	return f.result, f.callbackErr
}
func (f *fakeAuthProvider) Enabled() bool { return f.enabledFlag }

func newAuthTestEnv(t *testing.T) (*AuthHandler, domain.UserRepository, *auth.JWTService, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	require.NoError(t, sqlite.RunMigrations(db))
	t.Cleanup(func() { db.Close() })

	userRepo := sqlite.NewSQLiteUserRepository(db)
	sessRepo := sqlite.NewSQLiteSessionRepository(db)
	oidcRepo := sqlite.NewSQLiteOIDCIdentityRepository(db)
	jwtSvc := auth.NewJWTService("test-secret", 15*time.Minute)

	svc := auth.NewAuthService(userRepo, sessRepo, oidcRepo, jwtSvc, nil, 15*time.Minute, 7*24*time.Hour, true, true, "", "admin")
	h := NewAuthHandler(svc, jwtSvc, userRepo, nil, true, true)
	return h, userRepo, jwtSvc, db
}

func TestAuthHandler_Register_Success(t *testing.T) {
	h, userRepo, _, _ := newAuthTestEnv(t)
	r := gin.New()
	r.POST("/register", h.Register)

	body := `{"Name":"Admin","Email":"admin@test.com","Password":"password123"}`
	req := httptest.NewRequest("POST", "/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp authResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "admin@test.com", resp.User.Email)
	assert.Equal(t, "admin", resp.User.Role)
	assert.NotEmpty(t, resp.AccessToken)

	users, _ := userRepo.List(1, 10)
	require.Len(t, users, 1)
	assert.True(t, users[0].IsAdmin())
}

func TestAuthHandler_Register_MissingFields(t *testing.T) {
	h, _, _, _ := newAuthTestEnv(t)
	r := gin.New()
	r.POST("/register", h.Register)

	body := `{"Name":"A"}`
	req := httptest.NewRequest("POST", "/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_Register_DuplicateEmail(t *testing.T) {
	h, _, _, _ := newAuthTestEnv(t)
	r := gin.New()
	r.POST("/register", h.Register)

	body1 := `{"Name":"A","Email":"a@t.c","Password":"password123"}`
	body2 := `{"Name":"B","Email":"a@t.c","Password":"password456"}`

	req := httptest.NewRequest("POST", "/register", strings.NewReader(body1))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(httptest.NewRecorder(), req)

	req2 := httptest.NewRequest("POST", "/register", strings.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req2)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_Login_Success(t *testing.T) {
	h, _, _, _ := newAuthTestEnv(t)
	r := gin.New()
	r.POST("/register", h.Register)
	r.POST("/login", h.Login)

	regBody := `{"Name":"A","Email":"a@t.c","Password":"password123"}`
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/register", strings.NewReader(regBody)))

	loginBody := `{"Email":"a@t.c","Password":"password123"}`
	req := httptest.NewRequest("POST", "/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp authResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "a@t.c", resp.User.Email)
}

func TestAuthHandler_Login_WrongPassword(t *testing.T) {
	h, _, _, _ := newAuthTestEnv(t)
	r := gin.New()
	r.POST("/register", h.Register)
	r.POST("/login", h.Login)

	regBody := `{"Name":"A","Email":"a@t.c","Password":"password123"}`
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/register", strings.NewReader(regBody)))

	loginBody := `{"Email":"a@t.c","Password":"wrongpassword"}`
	req := httptest.NewRequest("POST", "/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_Refresh_Success(t *testing.T) {
	h, _, _, _ := newAuthTestEnv(t)
	r := gin.New()
	r.POST("/register", h.Register)
	r.POST("/refresh", h.Refresh)

	regBody := `{"Name":"A","Email":"a@t.c","Password":"password123"}`
	regW := httptest.NewRecorder()
	r.ServeHTTP(regW, httptest.NewRequest("POST", "/register", strings.NewReader(regBody)))

	var regResp authResponse
	require.NoError(t, json.Unmarshal(regW.Body.Bytes(), &regResp))

	refreshToken := ""
	for _, c := range regW.Result().Cookies() {
		if c.Name == "refresh_token" {
			refreshToken = c.Value
		}
	}
	require.NotEmpty(t, refreshToken)

	req := httptest.NewRequest("POST", "/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_Refresh_NoCookie(t *testing.T) {
	h, _, _, _ := newAuthTestEnv(t)
	r := gin.New()
	r.POST("/refresh", h.Refresh)

	req := httptest.NewRequest("POST", "/refresh", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_Logout(t *testing.T) {
	h, _, _, _ := newAuthTestEnv(t)
	r := gin.New()
	r.POST("/logout", h.Logout)

	req := httptest.NewRequest("POST", "/logout", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_Me_Authenticated(t *testing.T) {
	h, userRepo, _, _ := newAuthTestEnv(t)

	u := &domain.User{Name: "Test", Email: "t@t.c", PasswordHash: "h", Role: domain.RoleUser, CreatedAt: "2026-01-01T00:00:00Z"}
	require.NoError(t, userRepo.Add(u))

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user", u)
		c.Next()
	})
	r.GET("/me", h.Me)

	req := httptest.NewRequest("GET", "/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp userDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "t@t.c", resp.Email)
}

func TestAuthHandler_AuthStatus(t *testing.T) {
	h, _, _, _ := newAuthTestEnv(t)
	r := gin.New()
	r.GET("/status", h.AuthStatus)

	req := httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body, _ := io.ReadAll(w.Body)
	assert.Contains(t, string(body), "LoginEnabled")
	assert.Contains(t, string(body), "RegistrationEnabled")
}

func TestAuthHandler_Register_Disabled(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	require.NoError(t, sqlite.RunMigrations(db))
	t.Cleanup(func() { db.Close() })

	userRepo := sqlite.NewSQLiteUserRepository(db)
	sessRepo := sqlite.NewSQLiteSessionRepository(db)
	oidcRepo := sqlite.NewSQLiteOIDCIdentityRepository(db)
	jwtSvc := auth.NewJWTService("test-secret", 15*time.Minute)
	svc := auth.NewAuthService(userRepo, sessRepo, oidcRepo, jwtSvc, nil, 15*time.Minute, 7*24*time.Hour, true, false, "", "admin")
	h := NewAuthHandler(svc, jwtSvc, userRepo, nil, true, false)

	r := gin.New()
	r.POST("/register", h.Register)

	body := `{"Name":"A","Email":"a@t.c","Password":"password123"}`
	req := httptest.NewRequest("POST", "/register", strings.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), domain.ErrRegistrationDisabled.Error())
}

func TestAuthHandler_Login_Disabled(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	require.NoError(t, sqlite.RunMigrations(db))
	t.Cleanup(func() { db.Close() })

	userRepo := sqlite.NewSQLiteUserRepository(db)
	sessRepo := sqlite.NewSQLiteSessionRepository(db)
	oidcRepo := sqlite.NewSQLiteOIDCIdentityRepository(db)
	jwtSvc := auth.NewJWTService("test-secret", 15*time.Minute)
	svc := auth.NewAuthService(userRepo, sessRepo, oidcRepo, jwtSvc, nil, 15*time.Minute, 7*24*time.Hour, false, true, "", "admin")
	h := NewAuthHandler(svc, jwtSvc, userRepo, nil, false, true)

	r := gin.New()
	r.POST("/login", h.Login)

	body := `{"Email":"a@t.c","Password":"password123"}`
	req := httptest.NewRequest("POST", "/login", strings.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), domain.ErrLoginDisabled.Error())
}

func TestOIDCRedirectURI(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("from request host", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/auth/oidc/login", nil)
		c.Request.Host = "mediator.local:42800"
		assert.Equal(t, "http://mediator.local:42800"+AuthOIDCCallbackPath, oidcRedirectURI(c))
		assert.False(t, cookieSecure(c))
	})

	t.Run("from forwarded headers", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/auth/oidc/login", nil)
		c.Request.Host = "localhost:42800"
		c.Request.Header.Set("X-Forwarded-Proto", "https")
		c.Request.Header.Set("X-Forwarded-Host", "media.example.com")
		assert.Equal(t, "https://media.example.com"+AuthOIDCCallbackPath, oidcRedirectURI(c))
		assert.True(t, cookieSecure(c))
	})
}

func TestSetAuthCookies_SecureBehindProxy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Host = "localhost:42800"
	c.Request.Header.Set("X-Forwarded-Proto", "https")
	c.Request.Header.Set("X-Forwarded-Host", "mediator.example.com")

	setAuthCookies(c, "access", "refresh", 900)

	resp := w.Result()
	var accessCookie, refreshCookie *http.Cookie
	for _, ck := range resp.Cookies() {
		switch ck.Name {
		case "access_token":
			accessCookie = ck
		case "refresh_token":
			refreshCookie = ck
		}
	}
	require.NotNil(t, accessCookie)
	require.NotNil(t, refreshCookie)
	assert.True(t, accessCookie.Secure)
	assert.True(t, refreshCookie.Secure)
}

func TestAuthHandler_OIDCCallback_IdPErrorRedirectsToLogin(t *testing.T) {
	h, _, _, _ := newAuthTestEnv(t)
	provider := &fakeAuthProvider{enabledFlag: true}
	h.oidc = provider

	r := gin.New()
	r.GET("/auth/oidc/callback", h.OIDCCallback)

	req := httptest.NewRequest("GET", "/auth/oidc/callback?error=access_denied&error_description=denied&state=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "/login?")
	assert.Contains(t, w.Header().Get("Location"), "error=access_denied")
}

func TestAuthHandler_OIDCLogin_RequiresState(t *testing.T) {
	h, _, _, _ := newAuthTestEnv(t)
	h.oidc = &fakeAuthProvider{enabledFlag: true}

	r := gin.New()
	r.GET("/auth/oidc/login", h.OIDCLogin)

	req := httptest.NewRequest("GET", "/auth/oidc/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "missing oidc state")
}

func TestAuthHandler_OIDCLogin_SetsStateCookie(t *testing.T) {
	h, _, _, _ := newAuthTestEnv(t)
	h.oidc = &fakeAuthProvider{enabledFlag: true}

	r := gin.New()
	r.GET("/auth/oidc/login", h.OIDCLogin)

	req := httptest.NewRequest("GET", "/auth/oidc/login?state=csrf-secret", nil)
	req.Host = "mediator.local:42800"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Contains(t, body["RedirectURL"], "state=csrf-secret")

	var stateCookie *http.Cookie
	for _, ck := range w.Result().Cookies() {
		if ck.Name == "oidc_state" {
			stateCookie = ck
		}
	}
	require.NotNil(t, stateCookie)
	assert.Equal(t, "csrf-secret", stateCookie.Value)
	assert.True(t, stateCookie.HttpOnly)
	assert.Equal(t, 600, stateCookie.MaxAge)
}

func TestAuthHandler_OIDCCallback_RejectsMissingOrMismatchedState(t *testing.T) {
	h, _, _, _ := newAuthTestEnv(t)
	h.oidc = &fakeAuthProvider{enabledFlag: true}

	r := gin.New()
	r.GET("/auth/oidc/callback", h.OIDCCallback)

	t.Run("missing cookie", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/auth/oidc/callback?code=abc&state=csrf-secret", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), domain.ErrOIDCStateMismatch.Error())
	})

	t.Run("mismatched cookie", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/auth/oidc/callback?code=abc&state=csrf-secret", nil)
		req.AddCookie(&http.Cookie{Name: "oidc_state", Value: "other"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), domain.ErrOIDCStateMismatch.Error())
	})

	t.Run("empty query state", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/auth/oidc/callback?code=abc", nil)
		req.AddCookie(&http.Cookie{Name: "oidc_state", Value: "csrf-secret"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), domain.ErrOIDCStateMismatch.Error())
	})
}

