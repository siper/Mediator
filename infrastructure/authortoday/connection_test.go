package authortoday

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestScraper(t *testing.T, mux *http.ServeMux) *Scraper {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	orig := siteBase
	t.Cleanup(func() { siteBase = orig })
	siteBase = srv.URL
	return &Scraper{http: srv.Client()}
}

func TestScraper_TestConnection_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html></html>"))
	})
	s := newTestScraper(t, mux)

	err := s.TestConnection(context.Background())
	assert.NoError(t, err)
}

func TestScraper_TestConnection_ErrorsOnNonOKStatus(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	s := newTestScraper(t, mux)

	err := s.TestConnection(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "503")
}

func TestProvider_TestConnection_DelegatesToScraper(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s := newTestScraper(t, mux)
	p := &Provider{scraper: s}

	err := p.TestConnection(context.Background())
	assert.NoError(t, err)
}
