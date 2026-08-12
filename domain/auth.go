package domain

import "context"

type AuthProvider interface {
	LoginURL(state, redirectURI string) string
	Callback(ctx context.Context, code, state, redirectURI string) (OIDCCallbackResult, error)
	Enabled() bool
}

type OIDCCallbackResult struct {
	Email   string
	Name    string
	Subject string
	Roles   []string
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

type AuthService interface {
	Register(ctx context.Context, name, email, password string) (*TokenPair, error)
	LoginLocal(ctx context.Context, email, password string) (*TokenPair, error)
	LoginOIDC(ctx context.Context, state, code, redirectURI string) (*TokenPair, error)
	Refresh(ctx context.Context, refreshToken string) (*TokenPair, error)
	Logout(ctx context.Context, refreshToken string) error
}
