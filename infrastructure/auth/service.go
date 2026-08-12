package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"stersh.ru/mediator/domain"
)

type AuthService struct {
	users               domain.UserRepository
	sessions            domain.SessionRepository
	oidcIdents          domain.OIDCIdentityRepository
	jwt                 *JWTService
	provider            domain.AuthProvider
	refreshTTL          time.Duration
	accessTTL           time.Duration
	loginEnabled        bool
	registrationEnabled bool
	oidcIssuer          string
	adminRole           string
}

func NewAuthService(
	users domain.UserRepository,
	sessions domain.SessionRepository,
	oidcIdents domain.OIDCIdentityRepository,
	jwtSvc *JWTService,
	provider domain.AuthProvider,
	accessTTL time.Duration,
	refreshTTL time.Duration,
	loginEnabled bool,
	registrationEnabled bool,
	oidcIssuer string,
	adminRole string,
) *AuthService {
	return &AuthService{
		users:               users,
		sessions:            sessions,
		oidcIdents:          oidcIdents,
		jwt:                 jwtSvc,
		provider:            provider,
		accessTTL:           accessTTL,
		refreshTTL:          refreshTTL,
		loginEnabled:        loginEnabled,
		registrationEnabled: registrationEnabled,
		oidcIssuer:          oidcIssuer,
		adminRole:           adminRole,
	}
}

func (s *AuthService) Register(ctx context.Context, name, email, password string) (*domain.TokenPair, error) {
	if !s.registrationEnabled {
		return nil, domain.ErrRegistrationDisabled
	}
	if len(password) < 6 {
		return nil, domain.ErrPasswordTooShort
	}

	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	count, err := s.users.Count()
	if err != nil {
		return nil, err
	}

	role := domain.RoleUser
	if count == 0 {
		role = domain.RoleAdmin
	}

	now := time.Now().UTC().Format(time.RFC3339)
	u := &domain.User{
		Name:         name,
		Email:        email,
		PasswordHash: hash,
		Role:         role,
		CreatedAt:    now,
	}
	if err := s.users.Add(u); err != nil {
		return nil, err
	}

	return s.issueTokens(u)
}

func (s *AuthService) LoginLocal(ctx context.Context, email, password string) (*domain.TokenPair, error) {
	if !s.loginEnabled {
		return nil, domain.ErrLoginDisabled
	}

	u, err := s.users.GetByEmail(email)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}
	if u.PasswordHash == "" || !VerifyPassword(password, u.PasswordHash) {
		return nil, domain.ErrInvalidCredentials
	}
	return s.issueTokens(u)
}

func (s *AuthService) LoginOIDC(ctx context.Context, state, code, redirectURI string) (*domain.TokenPair, error) {
	if s.provider == nil || !s.provider.Enabled() {
		return nil, domain.ErrOIDCNotConfigured
	}

	result, err := s.provider.Callback(ctx, code, state, redirectURI)
	if err != nil {
		return nil, err
	}

	if result.Email == "" {
		return nil, fmt.Errorf("oidc callback returned empty email")
	}

	var u *domain.User
	if s.oidcIdents != nil && s.oidcIssuer != "" {
		ident, err := s.oidcIdents.GetByIssuerSubject(s.oidcIssuer, result.Subject)
		if err == nil {
			u, err = s.users.GetByID(ident.UserId)
			if err != nil {
				return nil, err
			}
		}
	}

	if u == nil {
		u, err = s.users.GetByEmail(result.Email)
		if err != nil && err != domain.ErrUserNotFound {
			return nil, err
		}
		if u == nil {
			count, err := s.users.Count()
			if err != nil {
				return nil, err
			}
			role := domain.RoleUser
			if count == 0 {
				role = domain.RoleAdmin
			}
			if hasAdminRole(result.Roles, s.adminRole) {
				role = domain.RoleAdmin
			}
			now := time.Now().UTC().Format(time.RFC3339)
			u = &domain.User{
				Name:      result.Name,
				Email:     result.Email,
				Role:      role,
				CreatedAt: now,
			}
			if err := s.users.Add(u); err != nil {
				return nil, err
			}
			if s.oidcIdents != nil && s.oidcIssuer != "" {
				ident := &domain.OIDCIdentity{
					UserId:   u.Id,
					Issuer:   s.oidcIssuer,
					Subject:  result.Subject,
				}
				if err := s.oidcIdents.Add(ident); err != nil {
					return nil, err
				}
			}
		}
	}

	return s.issueTokens(u)
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*domain.TokenPair, error) {
	if refreshToken == "" {
		return nil, domain.ErrInvalidCredentials
	}

	hash := hashRefreshToken(refreshToken)
	sess, err := s.sessions.GetByRefreshHash(hash)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}
	if sess.Revoked {
		return nil, domain.ErrInvalidCredentials
	}

	expiresAt, err := time.Parse(time.RFC3339, sess.ExpiresAt)
	if err != nil || time.Now().After(expiresAt) {
		return nil, domain.ErrInvalidCredentials
	}

	u, err := s.users.GetByID(sess.UserId)
	if err != nil {
		return nil, err
	}

	if err := s.sessions.RevokeByTokenHash(hash); err == nil {
	}

	return s.issueTokens(u)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	hash := hashRefreshToken(refreshToken)
	if err := s.sessions.RevokeByTokenHash(hash); err != nil {
		return nil
	}
	return nil
}

func (s *AuthService) issueTokens(u *domain.User) (*domain.TokenPair, error) {
	accessToken, err := s.jwt.GenerateToken(u)
	if err != nil {
		return nil, err
	}

	refreshToken, err := generateRefreshToken()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	sess := &domain.Session{
		UserId:      u.Id,
		RefreshHash: hashRefreshToken(refreshToken),
		ExpiresAt:   now.Add(s.refreshTTL).Format(time.RFC3339),
		CreatedAt:   now.Format(time.RFC3339),
	}
	if err := s.sessions.Add(sess); err != nil {
		return nil, err
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.accessTTL.Seconds()),
	}, nil
}

func generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashRefreshToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func hasAdminRole(roles []string, adminRole string) bool {
	if adminRole == "" {
		return false
	}
	for _, r := range roles {
		if r == adminRole {
			return true
		}
	}
	return false
}
