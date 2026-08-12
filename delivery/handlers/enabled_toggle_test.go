package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
	"stersh.ru/mediator/infrastructure/sqlite"
	_ "modernc.org/sqlite"
)

type noopReloader struct{}

func (noopReloader) Reload() error { return nil }

type noopSourceTester struct{}

func (noopSourceTester) Test(context.Context, domain.Source) error { return nil }

func newToggleTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	require.NoError(t, sqlite.RunMigrations(db))
	t.Cleanup(func() { db.Close() })
	return db
}

type enabledTestEnv struct {
	R            *gin.Engine
	IndexerRepo  *sqlite.SQLiteIndexerRepository
	ClientRepo   *sqlite.SQLiteDownloadClientRepository
	SourceRepo   *sqlite.SQLiteSourceRepository
	ProxyRepo    *sqlite.SQLiteProxyRepository
}

func newEnabledTestEnv(t *testing.T) *enabledTestEnv {
	t.Helper()
	db := newToggleTestDB(t)
	env := &enabledTestEnv{
		IndexerRepo: sqlite.NewSQLiteIndexerRepository(db),
		ClientRepo:  sqlite.NewSQLiteDownloadClientRepository(db),
		SourceRepo:  sqlite.NewSQLiteSourceRepository(db),
		ProxyRepo:   sqlite.NewSQLiteProxyRepository(db),
	}
	configH := NewConfigHandler(env.IndexerRepo, noopReloader{}, nil, env.ClientRepo, noopReloader{}, nil, nil, nil, env.ProxyRepo, noopReloader{}, nil)
	sourceH := NewSourceHandler(env.SourceRepo, noopReloader{}, noopSourceTester{})

	r := gin.New()
	r.PATCH("/indexers/:id/enabled", configH.UpdateIndexerEnabled)
	r.PATCH("/download-clients/:id/enabled", configH.UpdateClientEnabled)
	r.PATCH("/sources/:id/enabled", sourceH.SetEnabled)
	r.PATCH("/proxies/:id/enabled", configH.UpdateProxyEnabled)
	env.R = r
	return env
}

func toggle(t *testing.T, r http.Handler, method, path string, enabled bool) (int, map[string]any) {
	t.Helper()
	b, err := json.Marshal(map[string]bool{"enabled": enabled})
	require.NoError(t, err)
	req := httptest.NewRequest(method, path, io.NopCloser(strings.NewReader(string(b))))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var res map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		res = map[string]any{}
	}
	return w.Code, res
}

func idStr(id domain.ID) string {
	return strconv.FormatUint(uint64(id), 10)
}

func TestPatchIndexerEnabled(t *testing.T) {
	env := newEnabledTestEnv(t)
	ix := &domain.Indexer{
		Name: "jackett", Type: domain.IndexerJackett,
		Settings: map[string]string{"endpoint": "http://x/api", "api_key": "k"}, Enabled: true,
	}
	require.NoError(t, env.IndexerRepo.Add(ix))

	code, body := toggle(t, env.R, http.MethodPatch, "/indexers/"+idStr(ix.Id)+"/enabled", false)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, false, body["Enabled"])

	got, err := env.IndexerRepo.GetById(ix.Id)
	require.NoError(t, err)
	assert.False(t, got.Enabled)
}

func TestPatchIndexerEnabledNotFound(t *testing.T) {
	env := newEnabledTestEnv(t)
	code, _ := toggle(t, env.R, http.MethodPatch, "/indexers/999/enabled", false)
	assert.Equal(t, http.StatusNotFound, code)
}

func TestPatchClientEnabled(t *testing.T) {
	env := newEnabledTestEnv(t)
	cl := &domain.DownloadClient{
		Name: "qb", Type: domain.DownloadClientQBittorrent,
		Settings: map[string]string{"host": "http://x"}, Enabled: false,
	}
	require.NoError(t, env.ClientRepo.Add(cl))

	code, body := toggle(t, env.R, http.MethodPatch, "/download-clients/"+idStr(cl.Id)+"/enabled", true)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, true, body["Enabled"])

	got, err := env.ClientRepo.GetById(cl.Id)
	require.NoError(t, err)
	assert.True(t, got.Enabled)
}

func TestPatchClientEnabledNotFound(t *testing.T) {
	env := newEnabledTestEnv(t)
	code, _ := toggle(t, env.R, http.MethodPatch, "/download-clients/999/enabled", true)
	assert.Equal(t, http.StatusNotFound, code)
}

func TestPatchSourceEnabled(t *testing.T) {
	env := newEnabledTestEnv(t)
	s := &domain.Source{Type: "tmdb", Name: "TMDB", Settings: map[string]string{"api_key": "k"}, Enabled: true}
	require.NoError(t, env.SourceRepo.Add(s))

	code, body := toggle(t, env.R, http.MethodPatch, "/sources/"+idStr(s.Id)+"/enabled", false)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, false, body["Enabled"])

	got, err := env.SourceRepo.GetByID(s.Id)
	require.NoError(t, err)
	assert.False(t, got.Enabled)
}

func TestPatchSourceEnabledNotFound(t *testing.T) {
	env := newEnabledTestEnv(t)
	code, _ := toggle(t, env.R, http.MethodPatch, "/sources/999/enabled", false)
	assert.Equal(t, http.StatusNotFound, code)
}

func TestPatchProxyEnabled(t *testing.T) {
	env := newEnabledTestEnv(t)
	p := &domain.Proxy{
		Name: "proxy", Type: domain.ProxyHTTP, Endpoint: "http://host:8080", Enabled: true,
	}
	require.NoError(t, env.ProxyRepo.Add(p))

	code, body := toggle(t, env.R, http.MethodPatch, "/proxies/"+idStr(p.Id)+"/enabled", false)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, false, body["Enabled"])

	got, err := env.ProxyRepo.GetByID(p.Id)
	require.NoError(t, err)
	assert.False(t, got.Enabled)
}

func TestPatchProxyEnabledNotFound(t *testing.T) {
	env := newEnabledTestEnv(t)
	code, _ := toggle(t, env.R, http.MethodPatch, "/proxies/999/enabled", false)
	assert.Equal(t, http.StatusNotFound, code)
}
