package prowlarr

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscover_ReturnsEnabledIndexers(t *testing.T) {
	var gotPath string
	var gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.URL.Query().Get("apikey")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":1,"name":"1337x","enable":true},
			{"id":2,"name":"RARBG","enable":false},
			{"id":3,"name":"Nyaa","enable":true}
		]`))
	}))
	defer srv.Close()

	defs, err := Discover(context.Background(), srv.URL, "secret", srv.Client())
	require.NoError(t, err)
	assert.Equal(t, "/api/v1/indexer", gotPath)
	assert.Equal(t, "secret", gotKey)

	require.Len(t, defs, 2)
	assert.Equal(t, 1, defs[0].ID)
	assert.Equal(t, "1337x", defs[0].Name)
	assert.Equal(t, 3, defs[1].ID)
	assert.Equal(t, "Nyaa", defs[1].Name)
}

func TestDiscover_IncludesIndexersWhenEnableAbsent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"id":5,"name":"EZTV"}]`))
	}))
	defer srv.Close()

	defs, err := Discover(context.Background(), srv.URL, "", srv.Client())
	require.NoError(t, err)
	require.Len(t, defs, 1)
	assert.Equal(t, 5, defs[0].ID)
	assert.Equal(t, "EZTV", defs[0].Name)
}

func TestDiscover_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	_, err := Discover(context.Background(), srv.URL, "bad", srv.Client())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 401")
}

func TestDiscover_SkipsEntriesWithoutIDOrName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[
			{"id":0,"name":"NoID"},
			{"id":7,"name":""},
			{"id":9,"name":"Valid"}
		]`))
	}))
	defer srv.Close()

	defs, err := Discover(context.Background(), srv.URL, "", srv.Client())
	require.NoError(t, err)
	require.Len(t, defs, 1)
	assert.Equal(t, "Valid", defs[0].Name)
}
