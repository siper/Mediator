package delivery

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"stersh.ru/mediator/domain"
	"stersh.ru/mediator/infrastructure/auth"
)

type AuthMiddleware struct {
	jwt   *auth.JWTService
	users domain.UserRepository
}

func NewAuthMiddleware(j *auth.JWTService, u domain.UserRepository) *AuthMiddleware {
	return &AuthMiddleware{jwt: j, users: u}
}

func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr, err := c.Cookie("access_token")
		if err != nil || tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		claims, err := m.jwt.ValidateToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		u, err := m.users.GetByID(claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}

		c.Set("user_id", u.Id)
		c.Set("user_role", string(u.Role))
		c.Set("user", u)
		c.Next()
	}
}

func (m *AuthMiddleware) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("user_role")
		if !exists || role != string(domain.RoleAdmin) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
		c.Next()
	}
}

func CurrentUser(c *gin.Context) *domain.User {
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
