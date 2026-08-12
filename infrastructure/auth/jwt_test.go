package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

func TestJWTService_GenerateAndValidate(t *testing.T) {
	svc := NewJWTService("test-secret", 15*time.Minute)
	u := &domain.User{Id: 42, Email: "a@b.c", Role: domain.RoleAdmin}

	token, err := svc.GenerateToken(u)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := svc.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, domain.ID(42), claims.UserID)
	assert.Equal(t, "admin", claims.Role)
}

func TestJWTService_InvalidToken(t *testing.T) {
	svc := NewJWTService("test-secret", 15*time.Minute)
	_, err := svc.ValidateToken("garbage")
	assert.Error(t, err)
}

func TestJWTService_WrongSecret(t *testing.T) {
	gen := NewJWTService("secret-a", 15*time.Minute)
	val := NewJWTService("secret-b", 15*time.Minute)

	u := &domain.User{Id: 1, Email: "x@y.z", Role: domain.RoleUser}
	token, err := gen.GenerateToken(u)
	require.NoError(t, err)

	_, err = val.ValidateToken(token)
	assert.Error(t, err)
}

func TestJWTService_ExpiredToken(t *testing.T) {
	svc := NewJWTService("secret", -1*time.Minute)
	u := &domain.User{Id: 1, Email: "x@y.z", Role: domain.RoleUser}
	token, err := svc.GenerateToken(u)
	require.NoError(t, err)

	_, err = svc.ValidateToken(token)
	assert.Error(t, err)
}
