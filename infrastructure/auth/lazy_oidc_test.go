package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLazyOIDCProvider_RetriesAfterDiscoveryFailure(t *testing.T) {
	var ready bool
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !ready {
			http.Error(w, "down", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{
			"issuer":%q,
			"authorization_endpoint":%q,
			"token_endpoint":%q,
			"jwks_uri":%q
		}`, srv.URL, srv.URL+"/auth", srv.URL+"/token", srv.URL+"/jwks")
	}))
	defer srv.Close()

	p := NewLazyOIDCProvider(srv.URL, "client", "secret", "roles", "admin")
	assert.False(t, p.Enabled())
	assert.Equal(t, "", p.LoginURL("state", "http://app/callback"))

	ready = true
	require.True(t, p.Enabled())
	url := p.LoginURL("state", "http://app/callback")
	assert.Contains(t, url, "/auth")
	assert.Contains(t, url, "client")

	_, err := p.Callback(context.Background(), "code", "state", "http://app/callback")
	assert.Error(t, err)
}
