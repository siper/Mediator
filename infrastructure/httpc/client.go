package httpc

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/proxy"

	"stersh.ru/mediator/domain"
)

const defaultTimeout = 30 * time.Second

func proxyURL(raw string) *url.URL {
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return nil
	}
	return u
}

func socks5Dialer(addr string) (proxy.Dialer, bool) {
	if strings.HasPrefix(addr, "socks5://") || strings.HasPrefix(addr, "socks5h://") {
		u, err := url.Parse(addr)
		if err != nil {
			return nil, false
		}
		var auth *proxy.Auth
		if u.User != nil {
			auth = &proxy.Auth{User: u.User.Username()}
			if pw, ok := u.User.Password(); ok {
				auth.Password = pw
			}
		}
		d, err := proxy.SOCKS5("tcp", u.Host, auth, proxy.FromEnvironment())
		if err != nil {
			return nil, false
		}
		return d, true
	}
	d, err := proxy.SOCKS5("tcp", addr, nil, proxy.FromEnvironment())
	if err != nil {
		return nil, false
	}
	return d, true
}

func configureProxy(t *http.Transport, p *domain.Proxy) {
	if p == nil {
		return
	}
	if p.Type == domain.ProxySOCKS5 {
		if d, ok := socks5Dialer(p.Endpoint); ok {
			t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return d.Dial(network, addr)
			}
		}
		return
	}
	if u := proxyURL(p.Endpoint); u != nil {
		t.Proxy = http.ProxyURL(u)
	}
}

func newTransport(p *domain.Proxy) *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	configureProxy(t, p)
	return t
}

func NewClient(timeout time.Duration, p *domain.Proxy) *http.Client {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &http.Client{
		Transport: newTransport(p),
		Timeout:   timeout,
	}
}
