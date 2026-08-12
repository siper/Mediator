package authortoday

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

func TestAPIClient_TestConnection_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/work/1/details", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer valid-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":1,"title":"Test"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	orig := apiBase
	t.Cleanup(func() { apiBase = orig })
	apiBase = srv.URL + "/"

	c := NewAPIClient(func() string { return "valid-token" })
	err := c.TestConnection(context.Background())
	assert.NoError(t, err)
}

func TestAPIClient_TestConnection_InvalidToken(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/work/1/details", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	orig := apiBase
	t.Cleanup(func() { apiBase = orig })
	apiBase = srv.URL + "/"

	c := NewAPIClient(func() string { return "bad-token" })
	err := c.TestConnection(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")
}

func TestAPIClient_TestConnection_NoToken(t *testing.T) {
	c := NewAPIClient(func() string { return "" })
	err := c.TestConnection(context.Background())
	assert.ErrorIs(t, err, domain.ErrNoToken)
}

func TestAPIClient_TestConnection_HTTPError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/work/1/details", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	orig := apiBase
	t.Cleanup(func() { apiBase = orig })
	apiBase = srv.URL + "/"

	c := NewAPIClient(func() string { return "valid-token" })
	err := c.TestConnection(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "503")
}
