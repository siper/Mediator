package sqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

func TestSource_ProxyIDRoundtrip(t *testing.T) {
	db := newTestDB(t)
	sourceRepo := NewSQLiteSourceRepository(db)
	proxyRepo := NewSQLiteProxyRepository(db)

	px := &domain.Proxy{Name: "p", Type: domain.ProxyHTTP, Endpoint: "http://h:1", Enabled: true}
	require.NoError(t, proxyRepo.Add(px))

	s := &domain.Source{
		Type:     string(domain.SourceTMDB),
		Name:     "TMDB",
		Settings: map[string]string{"api_key": "k"},
		Enabled:  true,
		ProxyID:  &px.Id,
	}
	require.NoError(t, sourceRepo.Add(s))

	got, err := sourceRepo.GetByID(s.Id)
	require.NoError(t, err)
	require.NotNil(t, got.ProxyID)
	assert.Equal(t, px.Id, *got.ProxyID)

	all, err := sourceRepo.List()
	require.NoError(t, err)
	require.Len(t, all, 1)
	require.NotNil(t, all[0].ProxyID)
	assert.Equal(t, px.Id, *all[0].ProxyID)

	got.ProxyID = nil
	require.NoError(t, sourceRepo.Update(got))
	after, err := sourceRepo.GetByID(s.Id)
	require.NoError(t, err)
	assert.Nil(t, after.ProxyID)
}
