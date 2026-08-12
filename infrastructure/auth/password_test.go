package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashAndVerify(t *testing.T) {
	h, err := HashPassword("secret123")
	require.NoError(t, err)
	assert.NotEqual(t, "secret123", h)
	assert.NotEmpty(t, h)
	assert.True(t, VerifyPassword("secret123", h))
	assert.False(t, VerifyPassword("wrong", h))
}

func TestHashPassword_DifferentEachTime(t *testing.T) {
	h1, _ := HashPassword("same")
	h2, _ := HashPassword("same")
	assert.NotEqual(t, h1, h2)
	assert.True(t, VerifyPassword("same", h1))
	assert.True(t, VerifyPassword("same", h2))
}
