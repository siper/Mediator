package domain

import (
	"fmt"
	"net/url"
	"strings"
)

type ProxyType string

const (
	ProxyHTTP   ProxyType = "http"
	ProxySOCKS5 ProxyType = "socks5"
)

var validProxyTypes = map[ProxyType]bool{
	ProxyHTTP:   true,
	ProxySOCKS5: true,
}

type Proxy struct {
	Id       ID
	Name     string
	Type     ProxyType
	Endpoint string
	Enabled  bool
}

func validateEndpoint(raw string) error {
	if raw == "" {
		return ErrEmptyPath
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || !strings.Contains(u.Host, ":") {
		return fmt.Errorf("%w: %q", ErrInvalidProxyEndpoint, raw)
	}
	return nil
}

func (p *Proxy) Validate() error {
	if p.Name == "" {
		return ErrEmptyName
	}
	if !validProxyTypes[p.Type] {
		return fmt.Errorf("%w: %s", ErrInvalidProxyType, p.Type)
	}
	if err := validateEndpoint(p.Endpoint); err != nil {
		return err
	}
	return nil
}

type ProxyRepository interface {
	Add(p *Proxy) error
	GetByID(id ID) (*Proxy, error)
	List() ([]Proxy, error)
	ListEnabled() ([]Proxy, error)
	Update(p *Proxy) error
	Remove(id ID) error
}
