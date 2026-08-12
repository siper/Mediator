package httpc

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

func TestNewClient_DirectWhenNoProxy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "direct-ok")
	}))
	defer srv.Close()

	c := NewClient(5*time.Second, nil)
	resp, err := c.Get(srv.URL)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Equal(t, "direct-ok", string(body))
}

func TestNewClient_HTTPProxy(t *testing.T) {
	var proxiedHost string
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxiedHost = r.URL.Host
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "via-proxy")
	}))
	defer proxy.Close()

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("target should not be reached directly")
	}))
	defer target.Close()

	targetHost := targetHost(t, target.URL)
	p := &domain.Proxy{Name: "office", Type: domain.ProxyHTTP, Endpoint: proxy.URL}
	c := NewClient(5*time.Second, p)
	resp, err := c.Get(target.URL)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Equal(t, targetHost, proxiedHost)
	assert.Equal(t, "via-proxy", string(body))
}

func targetHost(t *testing.T, raw string) string {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	return u.Host
}

func TestProxyURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantHost string
		wantNil bool
	}{
		{name: "host and port without scheme", raw: "proxy.local:8080", wantHost: "proxy.local:8080"},
		{name: "ipv4 and port without scheme", raw: "127.0.0.1:8080", wantHost: "127.0.0.1:8080"},
		{name: "host and port with http scheme", raw: "http://proxy.local:8080", wantHost: "proxy.local:8080"},
		{name: "host port with https scheme", raw: "https://proxy.local:8080", wantHost: "proxy.local:8080"},
		{name: "with userinfo", raw: "http://user:pass@proxy.local:8080", wantHost: "proxy.local:8080"},
		{name: "empty returns nil", raw: "", wantNil: true},
		{name: "invalid url returns nil", raw: "://bad", wantNil: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := proxyURL(tt.raw)
			if tt.wantNil {
				assert.Nil(t, u)
				return
			}
			require.NotNil(t, u)
			assert.Equal(t, tt.wantHost, u.Host)
		})
	}
}

func TestNewClient_HTTPProxyWithoutScheme(t *testing.T) {
	var proxiedHost string
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxiedHost = r.URL.Host
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "via-proxy")
	}))
	defer proxy.Close()

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("target should not be reached directly")
	}))
	defer target.Close()

	endpoint := strings.TrimPrefix(proxy.URL, "http://")
	p := &domain.Proxy{Name: "office", Type: domain.ProxyHTTP, Endpoint: endpoint}
	c := NewClient(5*time.Second, p)
	resp, err := c.Get(target.URL)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Equal(t, targetHost(t, target.URL), proxiedHost)
	assert.Equal(t, "via-proxy", string(body))
}
