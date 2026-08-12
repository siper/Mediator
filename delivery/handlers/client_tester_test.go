package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
	"stersh.ru/mediator/infrastructure/sqlite"
	_ "modernc.org/sqlite"
)

type fakeClientTester struct {
	err    error
	called bool
	got    domain.DownloadClient
}

func (f *fakeClientTester) Test(ctx context.Context, c domain.DownloadClient) error {
	f.called = true
	f.got = c
	return f.err
}

func newClientTestEnv(t *testing.T, tester *fakeClientTester) *gin.Engine {
	t.Helper()
	db := newToggleTestDB(t)
	repo := sqlite.NewSQLiteDownloadClientRepository(db)
	h := NewConfigHandler(nil, noopReloader{}, nil, repo, noopReloader{}, tester, nil, nil, nil, noopReloader{}, nil)
	r := gin.New()
	r.POST("/download-clients/test", h.TestClient)
	return r
}

func TestTestClient_Success(t *testing.T) {
	tester := &fakeClientTester{}
	r := newClientTestEnv(t, tester)

	body := `{"type":"qbittorrent","name":"qb","settings":{"host":"http://localhost:8080","username":"u","password":"p"},"enabled":true}`
	req := httptest.NewRequest(http.MethodPost, "/download-clients/test", io.NopCloser(strings.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, tester.called)
	assert.Equal(t, "qbittorrent", string(tester.got.Type))
	assert.Equal(t, "http://localhost:8080", tester.got.Get("host"))
	assert.Equal(t, "u", tester.got.Get("username"))
	assert.Equal(t, "p", tester.got.Get("password"))
}

func TestTestClient_MissingType(t *testing.T) {
	tester := &fakeClientTester{}
	r := newClientTestEnv(t, tester)

	body := `{"name":"qb"}`
	req := httptest.NewRequest(http.MethodPost, "/download-clients/test", io.NopCloser(strings.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, tester.called)
}

func TestTestClient_ConnectionFailed(t *testing.T) {
	tester := &fakeClientTester{err: errors.New("connection refused")}
	r := newClientTestEnv(t, tester)

	body := `{"type":"qbittorrent","name":"qb","settings":{"host":"http://localhost:8080","username":"u","password":"bad"}}`
	req := httptest.NewRequest(http.MethodPost, "/download-clients/test", io.NopCloser(strings.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "connection refused")
}
