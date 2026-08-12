package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	oidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"stersh.ru/mediator/domain"
)

type OIDCProvider struct {
	provider     *oidc.Provider
	verifier     *oidc.IDTokenVerifier
	oauth2Config *oauth2.Config
	roleClaim    string
	adminRole    string
}

func NewOIDCProvider(issuer, clientID, clientSecret, roleClaim, adminRole string) (*OIDCProvider, error) {
	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc provider init: %w", err)
	}

	return &OIDCProvider{
		provider: provider,
		verifier: provider.Verifier(&oidc.Config{ClientID: clientID}),
		oauth2Config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
		roleClaim: roleClaim,
		adminRole: adminRole,
	}, nil
}

func (p *OIDCProvider) withRedirect(redirectURI string) *oauth2.Config {
	cfg := *p.oauth2Config
	cfg.RedirectURL = redirectURI
	return &cfg
}

func (p *OIDCProvider) Enabled() bool {
	return p != nil && p.provider != nil
}

func (p *OIDCProvider) LoginURL(state, redirectURI string) string {
	return p.withRedirect(redirectURI).AuthCodeURL(state, oauth2.AccessTypeOnline)
}

func (p *OIDCProvider) Callback(ctx context.Context, code, state, redirectURI string) (domain.OIDCCallbackResult, error) {
	token, err := p.withRedirect(redirectURI).Exchange(ctx, code)
	if err != nil {
		return domain.OIDCCallbackResult{}, fmt.Errorf("oidc token exchange: %w", err)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return domain.OIDCCallbackResult{}, fmt.Errorf("no id_token in token response")
	}

	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return domain.OIDCCallbackResult{}, fmt.Errorf("oidc verify: %w", err)
	}

	var claims struct {
		Email   string   `json:"email"`
		Name    string   `json:"name"`
		Subject string   `json:"sub"`
		Roles   []string `json:"roles"`
		Groups  []string `json:"groups"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return domain.OIDCCallbackResult{}, fmt.Errorf("oidc parse claims: %w", err)
	}

	name := claims.Name
	if name == "" {
		name = claims.Email
	}

	return domain.OIDCCallbackResult{
		Email:   claims.Email,
		Name:    name,
		Subject: claims.Subject,
		Roles:   extractRoles(claims, p.roleClaim),
	}, nil
}

func (p *OIDCProvider) AdminRole() string {
	return p.adminRole
}

func (p *OIDCProvider) Issuer() string {
	return p.provider.Endpoint().AuthURL
}

func extractRoles(claims struct {
	Email   string   `json:"email"`
	Name    string   `json:"name"`
	Subject string   `json:"sub"`
	Roles   []string `json:"roles"`
	Groups  []string `json:"groups"`
}, roleClaim string) []string {
	switch roleClaim {
	case "groups":
		return claims.Groups
	case "roles":
		return claims.Roles
	default:
		return claims.Roles
	}
}

func GenerateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
