package auth

import (
	"context"
	"log/slog"
	"sync"

	"stersh.ru/mediator/domain"
)

type LazyOIDCProvider struct {
	mu           sync.Mutex
	inner        *OIDCProvider
	issuer       string
	clientID     string
	clientSecret string
	roleClaim    string
	adminRole    string
}

func NewLazyOIDCProvider(issuer, clientID, clientSecret, roleClaim, adminRole string) *LazyOIDCProvider {
	return &LazyOIDCProvider{
		issuer:       issuer,
		clientID:     clientID,
		clientSecret: clientSecret,
		roleClaim:    roleClaim,
		adminRole:    adminRole,
	}
}

func (p *LazyOIDCProvider) ensure() *OIDCProvider {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.inner != nil {
		return p.inner
	}
	op, err := NewOIDCProvider(p.issuer, p.clientID, p.clientSecret, p.roleClaim, p.adminRole)
	if err != nil {
		slog.Warn("oidc provider not ready", "issuer", p.issuer, "err", err)
		return nil
	}
	p.inner = op
	slog.Info("OIDC provider configured", "issuer", p.issuer)
	return p.inner
}

func (p *LazyOIDCProvider) Enabled() bool {
	return p != nil && p.ensure() != nil
}

func (p *LazyOIDCProvider) LoginURL(state, redirectURI string) string {
	op := p.ensure()
	if op == nil {
		return ""
	}
	return op.LoginURL(state, redirectURI)
}

func (p *LazyOIDCProvider) Callback(ctx context.Context, code, state, redirectURI string) (domain.OIDCCallbackResult, error) {
	op := p.ensure()
	if op == nil {
		return domain.OIDCCallbackResult{}, domain.ErrOIDCNotConfigured
	}
	return op.Callback(ctx, code, state, redirectURI)
}

func (p *LazyOIDCProvider) AdminRole() string {
	return p.adminRole
}

func (p *LazyOIDCProvider) Issuer() string {
	return p.issuer
}
