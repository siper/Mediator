package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProxy_Validate(t *testing.T) {
	tests := []struct {
		name    string
		p       Proxy
		wantErr error
	}{
		{
			name: "valid http without scheme",
			p:    Proxy{Name: "office", Type: ProxyHTTP, Endpoint: "proxy.local:8080"},
		},
		{
			name: "valid http with scheme",
			p:    Proxy{Name: "office", Type: ProxyHTTP, Endpoint: "http://proxy.local:8080"},
		},
		{
			name: "valid ipv4 without scheme",
			p:    Proxy{Name: "office", Type: ProxyHTTP, Endpoint: "127.0.0.1:8080"},
		},
		{
			name: "valid http with userinfo",
			p:    Proxy{Name: "office", Type: ProxyHTTP, Endpoint: "http://user:pass@proxy.local:8080"},
		},
		{
			name: "valid socks5 without scheme",
			p:    Proxy{Name: "tor", Type: ProxySOCKS5, Endpoint: "127.0.0.1:9050"},
		},
		{
			name: "valid socks5 with scheme",
			p:    Proxy{Name: "tor", Type: ProxySOCKS5, Endpoint: "socks5://127.0.0.1:9050"},
		},
		{
			name:    "missing name",
			p:       Proxy{Type: ProxyHTTP, Endpoint: "127.0.0.1:8080"},
			wantErr: ErrEmptyName,
		},
		{
			name:    "invalid type",
			p:       Proxy{Name: "x", Type: "ftp", Endpoint: "127.0.0.1:8080"},
			wantErr: ErrInvalidProxyType,
		},
		{
			name:    "empty endpoint",
			p:       Proxy{Name: "x", Type: ProxyHTTP, Endpoint: ""},
			wantErr: ErrEmptyPath,
		},
		{
			name:    "endpoint without port",
			p:       Proxy{Name: "x", Type: ProxyHTTP, Endpoint: "proxy.local"},
			wantErr: ErrInvalidProxyEndpoint,
		},
		{
			name:    "garbage endpoint",
			p:       Proxy{Name: "x", Type: ProxyHTTP, Endpoint: "://bad"},
			wantErr: ErrInvalidProxyEndpoint,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.p.Validate()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}
