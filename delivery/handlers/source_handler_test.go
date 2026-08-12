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

type fakeSourceTester struct {
	err     error
	called  bool
	got     domain.Source
}

func (f *fakeSourceTester) Test(ctx context.Context, s domain.Source) error {
	f.called = true
	f.got = s
	return f.err
}

func newSourceTestEnv(t *testing.T, tester *fakeSourceTester) *gin.Engine {
	t.Helper()
	db := newToggleTestDB(t)
	repo := sqlite.NewSQLiteSourceRepository(db)
	h := NewSourceHandler(repo, noopReloader{}, tester)
	r := gin.New()
	r.POST("/sources/test", h.TestSource)
	return r
}

func TestTestSource_Success(t *testing.T) {
	tester := &fakeSourceTester{}
	r := newSourceTestEnv(t, tester)

	body := `{"type":"tmdb","name":"TMDB","settings":{"api_key":"secret","language":"en-US"},"ProxyID":null}`
	req := httptest.NewRequest(http.MethodPost, "/sources/test", io.NopCloser(strings.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, tester.called)
	assert.Equal(t, "tmdb", tester.got.Type)
	assert.Equal(t, "secret", tester.got.Get("api_key"))
	assert.Equal(t, "en-US", tester.got.Get("language"))
}

func TestTestSource_MissingType(t *testing.T) {
	tester := &fakeSourceTester{}
	r := newSourceTestEnv(t, tester)

	body := `{"name":"TMDB"}`
	req := httptest.NewRequest(http.MethodPost, "/sources/test", io.NopCloser(strings.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, tester.called)
}

func TestTestSource_ConnectionFailed(t *testing.T) {
	tester := &fakeSourceTester{err: errors.New("connection refused")}
	r := newSourceTestEnv(t, tester)

	body := `{"type":"tmdb","name":"TMDB","settings":{"api_key":"bad"}}`
	req := httptest.NewRequest(http.MethodPost, "/sources/test", io.NopCloser(strings.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "connection refused")
}
