package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYearFromDate(t *testing.T) {
	t.Run("full date", func(t *testing.T) {
		y := YearFromDate("1999-03-31")
		require.NotNil(t, y)
		assert.Equal(t, 1999, *y)
	})
	t.Run("year only", func(t *testing.T) {
		y := YearFromDate("1975")
		require.NotNil(t, y)
		assert.Equal(t, 1975, *y)
	})
	t.Run("empty", func(t *testing.T) {
		assert.Nil(t, YearFromDate(""))
		assert.Nil(t, YearFromDate("   "))
	})
	t.Run("invalid", func(t *testing.T) {
		assert.Nil(t, YearFromDate("abcd"))
		assert.Nil(t, YearFromDate("99"))
		assert.Nil(t, YearFromDate("0999-01-01"))
	})
}

func TestYearPtr(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		y := YearPtr(1954)
		require.NotNil(t, y)
		assert.Equal(t, 1954, *y)
	})
	t.Run("out of range", func(t *testing.T) {
		assert.Nil(t, YearPtr(0))
		assert.Nil(t, YearPtr(999))
		assert.Nil(t, YearPtr(3001))
	})
}
