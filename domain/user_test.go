package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUser_Validate(t *testing.T) {
	tests := []struct {
		name    string
		user    *User
		wantErr error
	}{
		{
			name:    "valid admin",
			user:    &User{Name: "Admin", Email: "admin@test.com", PasswordHash: "hashed", Role: RoleAdmin},
			wantErr: nil,
		},
		{
			name:    "valid user",
			user:    &User{Name: "User", Email: "user@test.com", PasswordHash: "hashed", Role: RoleUser},
			wantErr: nil,
		},
		{
			name:    "empty name",
			user:    &User{Email: "a@b.c", PasswordHash: "x", Role: RoleUser},
			wantErr: ErrEmptyName,
		},
		{
			name:    "empty email",
			user:    &User{Name: "A", PasswordHash: "x", Role: RoleUser},
			wantErr: ErrEmptyEmail,
		},
		{
			name:    "empty password hash",
			user:    &User{Name: "A", Email: "a@b.c", Role: RoleUser},
			wantErr: ErrPasswordRequired,
		},
		{
			name:    "invalid role",
			user:    &User{Name: "A", Email: "a@b.c", PasswordHash: "x", Role: "superadmin"},
			wantErr: ErrInvalidRole,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.Validate()
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestUser_IsAdmin(t *testing.T) {
	assert.True(t, (&User{Role: RoleAdmin}).IsAdmin())
	assert.False(t, (&User{Role: RoleUser}).IsAdmin())
	assert.False(t, (&User{}).IsAdmin())
	assert.False(t, (*User)(nil).IsAdmin())
}
