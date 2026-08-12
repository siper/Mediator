package handlers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/application"
	"stersh.ru/mediator/domain"
	"stersh.ru/mediator/infrastructure/sqlite"
)

type stubMetadataSource struct{}

func (stubMetadataSource) Active() []domain.MediaProvider { return nil }

func newProviderHandlerTestEnv(t *testing.T) (*ProviderHandler, domain.SettingRepository) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	require.NoError(t, sqlite.RunMigrations(db))
	t.Cleanup(func() { db.Close() })
	settingRepo := sqlite.NewSQLiteSettingRepository(db)
	providerSvc := application.NewProviderService(stubMetadataSource{}, nil, nil, nil, nil, nil, nil)
	return NewProviderHandler(providerSvc, settingRepo), settingRepo
}

func providerImportRouter(h *ProviderHandler, user *domain.User) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if user != nil {
			c.Set("user", user)
		}
		c.Next()
	})
	r.POST("/import", h.Import)
	return r
}

func TestProviderHandler_Import_BlockedForNonAdminWhenRequestsEnabled(t *testing.T) {
	h, settingRepo := newProviderHandlerTestEnv(t)
	require.NoError(t, settingRepo.Set(domain.SettingRequestsEnabled, "true"))

	user := &domain.User{Id: 5, Role: domain.RoleUser}
	r := providerImportRouter(h, user)
	body := `{"provider":"tmdb","external_id":"42","type":2,"library_id":1}`
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/import", strings.NewReader(body)))

	assert.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
}

func TestProviderHandler_Import_PassesGateForAdminWhenRequestsEnabled(t *testing.T) {
	h, settingRepo := newProviderHandlerTestEnv(t)
	require.NoError(t, settingRepo.Set(domain.SettingRequestsEnabled, "true"))

	admin := &domain.User{Id: 1, Role: domain.RoleAdmin}
	r := providerImportRouter(h, admin)
	body := `{"provider":"tmdb","external_id":"42","type":2,"library_id":1}`
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/import", strings.NewReader(body)))

	assert.NotEqual(t, http.StatusForbidden, w.Code, "admin must pass the request-system gate; got %d %s", w.Code, w.Body.String())
}

func TestProviderHandler_Import_PassesGateForUserWhenRequestsDisabled(t *testing.T) {
	h, settingRepo := newProviderHandlerTestEnv(t)
	require.NoError(t, settingRepo.Set(domain.SettingRequestsEnabled, "false"))

	user := &domain.User{Id: 5, Role: domain.RoleUser}
	r := providerImportRouter(h, user)
	body := `{"provider":"tmdb","external_id":"42","type":2,"library_id":1}`
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/import", strings.NewReader(body)))

	assert.NotEqual(t, http.StatusForbidden, w.Code, "user must pass the gate when requests disabled; got %d %s", w.Code, w.Body.String())
}

func TestProviderHandler_Import_BlockedWhenUserMissing(t *testing.T) {
	h, settingRepo := newProviderHandlerTestEnv(t)
	require.NoError(t, settingRepo.Set(domain.SettingRequestsEnabled, "true"))

	r := providerImportRouter(h, nil)
	body := `{"provider":"tmdb","external_id":"42","type":2,"library_id":1}`
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/import", strings.NewReader(body)))

	assert.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
}
