package handlers

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"stersh.ru/mediator/domain"
	"stersh.ru/mediator/infrastructure/auth"
)

const AuthOIDCCallbackPath = "/auth/oidc/callback"

type AuthHandler struct {
	svc                 domain.AuthService
	jwt                 *auth.JWTService
	users               domain.UserRepository
	oidc                domain.AuthProvider
	loginEnabled        bool
	registrationEnabled bool
}

func NewAuthHandler(svc domain.AuthService, j *auth.JWTService, users domain.UserRepository, oidc domain.AuthProvider, loginEnabled, registrationEnabled bool) *AuthHandler {
	return &AuthHandler{svc: svc, jwt: j, users: users, oidc: oidc, loginEnabled: loginEnabled, registrationEnabled: registrationEnabled}
}

type loginRequest struct {
	Email    string `json:"Email"`
	Password string `json:"Password"`
}

type registerRequest struct {
	Name     string `json:"Name"`
	Email    string `json:"Email"`
	Password string `json:"Password"`
}

type authResponse struct {
	User        userDTO `json:"User"`
	AccessToken string  `json:"AccessToken"`
}

type userDTO struct {
	Id        domain.ID `json:"Id"`
	Name      string    `json:"Name"`
	Email     string    `json:"Email"`
	Role      string    `json:"Role"`
	CreatedAt string    `json:"CreatedAt"`
}

func toUserDTO(u *domain.User) userDTO {
	return userDTO{
		Id:        u.Id,
		Name:      u.Name,
		Email:     u.Email,
		Role:      string(u.Role),
		CreatedAt: u.CreatedAt,
	}
}

func setAuthCookies(c *gin.Context, accessToken, refreshToken string, accessTTL int) {
	secure := cookieSecure(c)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", accessToken, accessTTL, "/", "", secure, true)
	c.SetCookie("refresh_token", refreshToken, 7*24*3600, "/", "", secure, true)
}

func clearAuthCookies(c *gin.Context) {
	secure := cookieSecure(c)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", "", -1, "/", "", secure, true)
	c.SetCookie("refresh_token", "", -1, "/", "", secure, true)
}

func (h *AuthHandler) Register(c *gin.Context) {
	if !h.registrationEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrRegistrationDisabled.Error()})
		return
	}
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Name == "" || req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name, email and password are required"})
		return
	}

	pair, err := h.svc.Register(c.Request.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	u, err := h.getUserFromToken(pair.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	setAuthCookies(c, pair.AccessToken, pair.RefreshToken, pair.ExpiresIn)
	c.JSON(http.StatusCreated, authResponse{User: toUserDTO(u), AccessToken: pair.AccessToken})
}

func (h *AuthHandler) Login(c *gin.Context) {
	if !h.loginEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrLoginDisabled.Error()})
		return
	}
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pair, err := h.svc.LoginLocal(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	u, err := h.getUserFromToken(pair.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	setAuthCookies(c, pair.AccessToken, pair.RefreshToken, pair.ExpiresIn)
	c.JSON(http.StatusOK, authResponse{User: toUserDTO(u), AccessToken: pair.AccessToken})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, _ := c.Cookie("refresh_token")
	if refreshToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no refresh token"})
		return
	}

	pair, err := h.svc.Refresh(c.Request.Context(), refreshToken)
	if err != nil {
		clearAuthCookies(c)
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	u, err := h.getUserFromToken(pair.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	setAuthCookies(c, pair.AccessToken, pair.RefreshToken, pair.ExpiresIn)
	c.JSON(http.StatusOK, authResponse{User: toUserDTO(u), AccessToken: pair.AccessToken})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	refreshToken, _ := c.Cookie("refresh_token")
	_ = h.svc.Logout(c.Request.Context(), refreshToken)
	clearAuthCookies(c)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	u := currentUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	c.JSON(http.StatusOK, toUserDTO(u))
}

func (h *AuthHandler) OIDCLogin(c *gin.Context) {
	if h.oidc == nil || !h.oidc.Enabled() {
		c.JSON(http.StatusNotFound, gin.H{"error": "oidc not configured"})
		return
	}
	state := c.Query("state")
	if state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing oidc state"})
		return
	}
	c.SetCookie("oidc_state", state, 600, "/", "", cookieSecure(c), true)
	redirectURL := h.oidc.LoginURL(state, oidcRedirectURI(c))
	c.JSON(http.StatusOK, gin.H{"RedirectURL": redirectURL})
}

func (h *AuthHandler) OIDCCallback(c *gin.Context) {
	if h.oidc == nil || !h.oidc.Enabled() {
		c.JSON(http.StatusNotFound, gin.H{"error": "oidc not configured"})
		return
	}

	if errParam := c.Query("error"); errParam != "" {
		q := url.Values{"error": {errParam}}
		if desc := c.Query("error_description"); desc != "" {
			q.Set("error_description", desc)
		}
		c.Redirect(http.StatusFound, "/login?"+q.Encode())
		return
	}

	code := c.Query("code")
	state := c.Query("state")
	savedState, _ := c.Cookie("oidc_state")
	clearOIDCStateCookie(c)

	if code == "" {
		c.Redirect(http.StatusFound, "/login?error=missing_code")
		return
	}
	if savedState == "" || state == "" || state != savedState {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrOIDCStateMismatch.Error()})
		return
	}

	pair, err := h.svc.LoginOIDC(c.Request.Context(), state, code, oidcRedirectURI(c))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	u, err := h.getUserFromToken(pair.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	setAuthCookies(c, pair.AccessToken, pair.RefreshToken, pair.ExpiresIn)
	c.Redirect(http.StatusFound, "/")
	_ = u
}

func clearOIDCStateCookie(c *gin.Context) {
	c.SetCookie("oidc_state", "", -1, "/", "", cookieSecure(c), true)
}

func requestScheme(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = strings.TrimSpace(strings.Split(proto, ",")[0])
	}
	return scheme
}

func requestHost(c *gin.Context) string {
	host := c.Request.Host
	if fwd := c.GetHeader("X-Forwarded-Host"); fwd != "" {
		host = strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	return host
}

func cookieSecure(c *gin.Context) bool {
	return requestScheme(c) == "https"
}

func oidcRedirectURI(c *gin.Context) string {
	return requestScheme(c) + "://" + requestHost(c) + AuthOIDCCallbackPath
}

func (h *AuthHandler) AuthStatus(c *gin.Context) {
	status := gin.H{
		"LoginEnabled":        h.loginEnabled,
		"RegistrationEnabled": h.registrationEnabled,
		"OIDCEnabled":         h.oidc != nil && h.oidc.Enabled(),
	}
	c.JSON(http.StatusOK, status)
}

func (h *AuthHandler) getUserFromToken(token string) (*domain.User, error) {
	claims, err := h.jwt.ValidateToken(token)
	if err != nil {
		return nil, err
	}
	return h.users.GetByID(claims.UserID)
}

func currentUser(c *gin.Context) *domain.User {
	u, exists := c.Get("user")
	if !exists {
		return nil
	}
	user, ok := u.(*domain.User)
	if !ok {
		return nil
	}
	return user
}
