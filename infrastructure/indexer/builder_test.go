package indexer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
	"stersh.ru/mediator/infrastructure/indexer/prowlarr"
)

const emptyRSS = `<?xml version="1.0"?><rss><channel></channel></rss>`

func TestHostOnly(t *testing.T) {
	cases := []struct{ in, want string }{
		{"http://jackett:9117", "http://jackett:9117"},
		{"http://jackett:9117/", "http://jackett:9117"},
		{"http://jackett:9117/api/v2.0/indexers/all/results/torznab", "http://jackett:9117"},
		{"https://prowlarr:9696/1/api", "https://prowlarr:9696"},
		{"prowlarr:9696", "http://prowlarr:9696"},
		{"", ""},
		{"   ", ""},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, hostOnly(c.in), "input %q", c.in)
	}
}

func newCaptureServer() (*httptest.Server, *string) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(emptyRSS))
	}))
	return srv, &gotPath
}

func TestBuild_Jackett(t *testing.T) {
	srv, gotPath := newCaptureServer()
	defer srv.Close()

	clients, err := Build(domain.Indexer{
		Name:     "Jackett",
		Type:     domain.IndexerJackett,
		Enabled:  true,
		Settings: map[string]string{"endpoint": srv.URL, "api_key": "k"},
	})
	require.NoError(t, err)
	require.Len(t, clients, 1)
	assert.Equal(t, "Jackett", clients[0].Name())

	_, err = clients[0].Search(context.Background(), "movie", nil)
	require.NoError(t, err)
	assert.Equal(t, jackettTorznabPath, *gotPath)
}

func TestBuild_Jackett_NormalizesFullURL(t *testing.T) {
	srv, gotPath := newCaptureServer()
	defer srv.Close()

	clients, err := Build(domain.Indexer{
		Name:     "Jackett",
		Type:     domain.IndexerJackett,
		Enabled:  true,
		Settings: map[string]string{"endpoint": srv.URL + jackettTorznabPath},
	})
	require.NoError(t, err)
	require.Len(t, clients, 1)

	_, err = clients[0].Search(context.Background(), "movie", nil)
	require.NoError(t, err)
	assert.Equal(t, jackettTorznabPath, *gotPath)
}

func TestBuild_Prowlarr(t *testing.T) {
	srv, gotPath := newCaptureServer()
	defer srv.Close()

	orig := discoverFn
	discoverFn = func(_ context.Context, _, _ string, _ *http.Client) ([]prowlarr.Definition, error) {
		return []prowlarr.Definition{{ID: 1, Name: "1337x"}, {ID: 3, Name: "Nyaa"}}, nil
	}
	defer func() { discoverFn = orig }()

	clients, err := Build(domain.Indexer{
		Name:     "Prowlarr",
		Type:     domain.IndexerProwlarr,
		Enabled:  true,
		Settings: map[string]string{"endpoint": srv.URL, "api_key": "k"},
	})
	require.NoError(t, err)
	require.Len(t, clients, 2)
	assert.Equal(t, "Prowlarr: 1337x", clients[0].Name())
	assert.Equal(t, "Prowlarr: Nyaa", clients[1].Name())

	_, err = clients[0].Search(context.Background(), "movie", nil)
	require.NoError(t, err)
	assert.Equal(t, "/1/api", *gotPath)
}

func TestBuild_Prowlarr_NoIndexers(t *testing.T) {
	orig := discoverFn
	discoverFn = func(_ context.Context, _, _ string, _ *http.Client) ([]prowlarr.Definition, error) {
		return nil, nil
	}
	defer func() { discoverFn = orig }()

	_, err := Build(domain.Indexer{
		Name:     "Prowlarr",
		Type:     domain.IndexerProwlarr,
		Settings: map[string]string{"endpoint": "http://prowlarr:9696"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no enabled indexers")
}

func TestBuild_EmptyEndpoint(t *testing.T) {
	_, err := Build(domain.Indexer{Name: "X", Type: domain.IndexerJackett, Settings: map[string]string{}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestBuild_UnknownType(t *testing.T) {
	_, err := Build(domain.Indexer{
		Name:     "X",
		Type:     "bogus",
		Settings: map[string]string{"endpoint": "http://x:1"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown indexer type")
}
