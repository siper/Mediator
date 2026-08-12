package handlers

import (
	"database/sql"
	"encoding/json"
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

func newUserTestEnv(t *testing.T) (*UserHandler, domain.UserRepository) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	require.NoError(t, sqlite.RunMigrations(db))
	t.Cleanup(func() { db.Close() })

	userRepo := sqlite.NewSQLiteUserRepository(db)
	jwtSvc := auth.NewJWTService("test-secret", 15*time.Minute)
	h := NewUserHandler(userRepo, jwtSvc)
	return h, userRepo
}

func TestUserHandler_List(t *testing.T) {
	h, userRepo := newUserTestEnv(t)
	require.NoError(t, userRepo.Add(&domain.User{Name: "A", Email: "a@t.c", PasswordHash: "h", Role: domain.RoleUser, CreatedAt: "x"}))
	require.NoError(t, userRepo.Add(&domain.User{Name: "B", Email: "b@t.c", PasswordHash: "h", Role: domain.RoleAdmin, CreatedAt: "x"}))

	r := gin.New()
	r.GET("/users", h.List)

	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp []userDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp, 2)
}

func TestUserHandler_GetByID(t *testing.T) {
	h, userRepo := newUserTestEnv(t)
	u := &domain.User{Name: "A", Email: "a@t.c", PasswordHash: "h", Role: domain.RoleUser, CreatedAt: "x"}
	require.NoError(t, userRepo.Add(u))

	r := gin.New()
	r.GET("/users/:id", h.GetByID)

	req := httptest.NewRequest("GET", "/users/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp userDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "a@t.c", resp.Email)
}

func TestUserHandler_GetByID_NotFound(t *testing.T) {
	h, _ := newUserTestEnv(t)
	r := gin.New()
	r.GET("/users/:id", h.GetByID)

	req := httptest.NewRequest("GET", "/users/999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUserHandler_Create(t *testing.T) {
	h, _ := newUserTestEnv(t)
	r := gin.New()
	r.POST("/users", h.Create)

	body := `{"Name":"New","Email":"new@t.c","Password":"password123","Role":"user"}`
	req := httptest.NewRequest("POST", "/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp userDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "new@t.c", resp.Email)
	assert.Equal(t, "user", resp.Role)
}

func TestUserHandler_Create_ShortPassword(t *testing.T) {
	h, _ := newUserTestEnv(t)
	r := gin.New()
	r.POST("/users", h.Create)

	body := `{"Name":"New","Email":"new@t.c","Password":"short","Role":"user"}`
	req := httptest.NewRequest("POST", "/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_Update(t *testing.T) {
	h, userRepo := newUserTestEnv(t)
	u := &domain.User{Name: "A", Email: "a@t.c", PasswordHash: "h", Role: domain.RoleUser, CreatedAt: "x"}
	require.NoError(t, userRepo.Add(u))

	r := gin.New()
	r.PUT("/users/:id", h.Update)

	body := `{"Name":"Updated","Role":"admin"}`
	req := httptest.NewRequest("PUT", "/users/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp userDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Updated", resp.Name)
	assert.Equal(t, "admin", resp.Role)
}

func TestUserHandler_Delete(t *testing.T) {
	h, userRepo := newUserTestEnv(t)
	u := &domain.User{Name: "A", Email: "a@t.c", PasswordHash: "h", Role: domain.RoleUser, CreatedAt: "x"}
	require.NoError(t, userRepo.Add(u))

	r := gin.New()
	r.DELETE("/users/:id", h.Delete)

	req := httptest.NewRequest("DELETE", "/users/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	users, _ := userRepo.List(1, 10)
	assert.Empty(t, users)
}
