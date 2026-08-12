package sqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

func TestProxy_RepoRoundtrip(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteProxyRepository(db)

	p := &domain.Proxy{
		Name:     "office",
		Type:     domain.ProxyHTTP,
		Endpoint: "http://proxy.local:8080",
		Enabled:  true,
	}
	require.NoError(t, repo.Add(p))
	assert.NotZero(t, p.Id)

	got, err := repo.GetByID(p.Id)
	require.NoError(t, err)
	assert.Equal(t, "office", got.Name)
	assert.Equal(t, domain.ProxyHTTP, got.Type)
	assert.Equal(t, "http://proxy.local:8080", got.Endpoint)
	assert.True(t, got.Enabled)

	all, err := repo.List()
	require.NoError(t, err)
	require.Len(t, all, 1)

	got.Endpoint = "http://proxy.local:9090"
	got.Enabled = false
	require.NoError(t, repo.Update(got))

	after, err := repo.GetByID(p.Id)
	require.NoError(t, err)
	assert.Equal(t, "http://proxy.local:9090", after.Endpoint)
	assert.False(t, after.Enabled)

	enabled, err := repo.ListEnabled()
	require.NoError(t, err)
	require.Empty(t, enabled)

	require.NoError(t, repo.Remove(p.Id))
	_, err = repo.GetByID(p.Id)
	assert.ErrorIs(t, err, domain.ErrProxyNotFound)
}

func TestProxy_Validate(t *testing.T) {
	tests := []struct {
		name    string
		proxy   *domain.Proxy
		wantErr error
	}{
		{"empty name", &domain.Proxy{Name: "", Type: domain.ProxyHTTP, Endpoint: "h:1"}, domain.ErrEmptyName},
		{"bad type", &domain.Proxy{Name: "x", Type: "ftp", Endpoint: "h:1"}, domain.ErrInvalidProxyType},
		{"empty endpoint", &domain.Proxy{Name: "x", Type: domain.ProxyHTTP, Endpoint: ""}, domain.ErrEmptyPath},
		{"valid", &domain.Proxy{Name: "x", Type: domain.ProxySOCKS5, Endpoint: "h:1"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.proxy.Validate()
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestProxy_GetByID_NotFound(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteProxyRepository(db)
	_, err := repo.GetByID(999)
	assert.ErrorIs(t, err, domain.ErrProxyNotFound)
}
