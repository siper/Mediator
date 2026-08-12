package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
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

func newRequestTestEnv(t *testing.T) (*RequestHandler, domain.UserRepository, domain.LibraryRepository) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	require.NoError(t, sqlite.RunMigrations(db))
	t.Cleanup(func() { db.Close() })

	userRepo := sqlite.NewSQLiteUserRepository(db)
	settingRepo := sqlite.NewSQLiteSettingRepository(db)
	mediaRepo := sqlite.NewSQLiteMediaRepository(db)
	requestRepo := sqlite.NewSQLiteMediaRequestRepository(db)
	imp := requestTestImporter{}
	svc := application.NewMediaRequestService(requestRepo, mediaRepo, imp, settingRepo, imp, requestTestCoverStore{})
	h := NewRequestHandler(svc)
	return h, userRepo, sqlite.NewSQLiteLibraryRepository(db)
}

type requestTestImporter struct{}

func (requestTestImporter) Import(ctx context.Context, providerName string, externalID string, mediaType domain.MediaType, libraryID domain.ID, profileID *domain.ID, folder string, coverURL string) (*domain.Media, error) {
	return &domain.Media{Id: 99}, nil
}

func (requestTestImporter) LookupMedia(ctx context.Context, provider string, externalID string, mediaType domain.MediaType) (*domain.SearchResult, error) {
	return &domain.SearchResult{ProviderName: provider, ExternalID: externalID, Title: "Resolved Title", MediaType: mediaType}, nil
}

type requestTestCoverStore struct{}

func (requestTestCoverStore) Store(ctx context.Context, sourceURL string) (string, error) {
	return sourceURL, nil
}

func requestTestRouter(h *RequestHandler, user *domain.User) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user", user)
		c.Next()
	})
	r.POST("/requests", h.Create)
	r.GET("/requests", h.List)
	r.GET("/requests/:id", h.GetByID)
	r.POST("/requests/:id/cancel", h.Cancel)
	r.POST("/requests/:id/approve", h.Approve)
	r.POST("/requests/:id/reject", h.Reject)
	return r
}

func TestRequestHandler_Create(t *testing.T) {
	h, userRepo, libRepo := newRequestTestEnv(t)
	require.NoError(t, userRepo.Add(&domain.User{Name: "U", Email: "u@t.c", PasswordHash: "h", Role: domain.RoleUser, CreatedAt: "x"}))
	require.NoError(t, libRepo.Add(&domain.Library{Name: "L", Path: "movies", Type: domain.MediaTypeMovie, Settings: map[string]string{}}))

	admin := &domain.User{Id: 1, Role: domain.RoleAdmin}
	r := requestTestRouter(h, admin)

	body := `{"provider":"tmdb","external_id":"42","type":2,"library_id":1}`
	req := httptest.NewRequest("POST", "/requests", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var resp domain.MediaRequest
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, domain.MediaRequestPending, resp.Status)
	assert.Equal(t, domain.ID(1), resp.UserId)
}

func TestRequestHandler_List(t *testing.T) {
	h, userRepo, libRepo := newRequestTestEnv(t)
	require.NoError(t, userRepo.Add(&domain.User{Name: "U", Email: "u@t.c", PasswordHash: "h", Role: domain.RoleUser, CreatedAt: "x"}))
	require.NoError(t, libRepo.Add(&domain.Library{Name: "L", Path: "movies", Type: domain.MediaTypeMovie, Settings: map[string]string{}}))
	require.NoError(t, userRepo.Add(&domain.User{Id: 2, Name: "A", Email: "a@t.c", PasswordHash: "h", Role: domain.RoleAdmin, CreatedAt: "x"}))

	svc := h.svc
	_, err := svc.Create(context.Background(), 2, "tmdb", "1", "Matrix", "/cover.jpg", domain.MediaTypeMovie, 1, nil, "")
	require.NoError(t, err)

	adminUser := &domain.User{Id: 2, Role: domain.RoleAdmin}
	r := requestTestRouter(h, adminUser)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/requests", nil))
	require.Equal(t, http.StatusOK, w.Code)
	var list []domain.MediaRequest
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
	assert.Len(t, list, 1)

	userUser := &domain.User{Id: 1, Role: domain.RoleUser}
	rUser := requestTestRouter(h, userUser)
	w2 := httptest.NewRecorder()
	rUser.ServeHTTP(w2, httptest.NewRequest("GET", "/requests", nil))
	require.Equal(t, http.StatusOK, w2.Code)
	var userList []domain.MediaRequest
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &userList))
	assert.Empty(t, userList, "regular user should not see other users' requests")
}

func TestRequestHandler_GetByID_Visibility(t *testing.T) {
	h, userRepo, libRepo := newRequestTestEnv(t)
	require.NoError(t, userRepo.Add(&domain.User{Id: 1, Name: "U", Email: "u@t.c", PasswordHash: "h", Role: domain.RoleUser, CreatedAt: "x"}))
	require.NoError(t, libRepo.Add(&domain.Library{Name: "L", Path: "movies", Type: domain.MediaTypeMovie, Settings: map[string]string{}}))

	svc := h.svc
	_, err := svc.Create(context.Background(), 1, "tmdb", "1", "Matrix", "/cover.jpg", domain.MediaTypeMovie, 1, nil, "")
	require.NoError(t, err)

	admin := &domain.User{Id: 2, Role: domain.RoleAdmin}
	r := requestTestRouter(h, admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/requests/1", nil))
	require.Equal(t, http.StatusOK, w.Code)

	other := &domain.User{Id: 9, Role: domain.RoleUser}
	r2 := requestTestRouter(h, other)
	w2 := httptest.NewRecorder()
	r2.ServeHTTP(w2, httptest.NewRequest("GET", "/requests/1", nil))
	require.Equal(t, http.StatusNotFound, w2.Code, "non-owner non-admin should not see others' requests")

	owner := &domain.User{Id: 1, Role: domain.RoleUser}
	r3 := requestTestRouter(h, owner)
	w3 := httptest.NewRecorder()
	r3.ServeHTTP(w3, httptest.NewRequest("GET", "/requests/1", nil))
	require.Equal(t, http.StatusOK, w3.Code)
}

func TestRequestHandler_Cancel(t *testing.T) {
	h, userRepo, libRepo := newRequestTestEnv(t)
	require.NoError(t, userRepo.Add(&domain.User{Id: 1, Name: "U", Email: "u@t.c", PasswordHash: "h", Role: domain.RoleUser, CreatedAt: "x"}))
	require.NoError(t, libRepo.Add(&domain.Library{Name: "L", Path: "movies", Type: domain.MediaTypeMovie, Settings: map[string]string{}}))

	svc := h.svc
	_, err := svc.Create(context.Background(), 1, "tmdb", "1", "Matrix", "/cover.jpg", domain.MediaTypeMovie, 1, nil, "")
	require.NoError(t, err)

	other := &domain.User{Id: 9, Role: domain.RoleUser}
	r2 := requestTestRouter(h, other)
	w2 := httptest.NewRecorder()
	r2.ServeHTTP(w2, httptest.NewRequest("POST", "/requests/1/cancel", nil))
	require.Equal(t, http.StatusForbidden, w2.Code)

	owner := &domain.User{Id: 1, Role: domain.RoleUser}
	r := requestTestRouter(h, owner)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/requests/1/cancel", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp domain.MediaRequest
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, domain.MediaRequestCanceled, resp.Status)
}

func TestRequestHandler_Reject(t *testing.T) {
	h, userRepo, libRepo := newRequestTestEnv(t)
	require.NoError(t, userRepo.Add(&domain.User{Id: 1, Name: "U", Email: "u@t.c", PasswordHash: "h", Role: domain.RoleUser, CreatedAt: "x"}))
	require.NoError(t, libRepo.Add(&domain.Library{Name: "L", Path: "movies", Type: domain.MediaTypeMovie, Settings: map[string]string{}}))

	svc := h.svc
	_, err := svc.Create(context.Background(), 1, "tmdb", "1", "Matrix", "/cover.jpg", domain.MediaTypeMovie, 1, nil, "")
	require.NoError(t, err)

	admin := &domain.User{Id: 2, Role: domain.RoleAdmin}
	r := requestTestRouter(h, admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/requests/1/reject", strings.NewReader(`{"notes":"bad"}`)))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp domain.MediaRequest
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, domain.MediaRequestRejected, resp.Status)
	require.NotNil(t, resp.Notes)
	assert.Equal(t, "bad", *resp.Notes)
	require.NotNil(t, resp.RejectedBy)
	assert.Equal(t, domain.ID(2), *resp.RejectedBy)
}
