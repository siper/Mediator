package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

type memSettings struct {
	data map[string]string
}

func (m *memSettings) Get(key string) (string, error) {
	v, ok := m.data[key]
	if !ok {
		return "", domain.ErrSettingNotFound
	}
	return v, nil
}

func (m *memSettings) Set(key, value string) error {
	if m.data == nil {
		m.data = map[string]string{}
	}
	m.data[key] = value
	return nil
}

func (m *memSettings) List() ([]domain.Setting, error) {
	out := make([]domain.Setting, 0, len(m.data))
	for k, v := range m.data {
		out = append(out, domain.Setting{Key: k, Value: v})
	}
	return out, nil
}

func TestLoadOrCreateJWTSecret_CreatesOnce(t *testing.T) {
	s := &memSettings{}
	first, err := LoadOrCreateJWTSecret(s)
	require.NoError(t, err)
	assert.NotEmpty(t, first)

	second, err := LoadOrCreateJWTSecret(s)
	require.NoError(t, err)
	assert.Equal(t, first, second)
}
