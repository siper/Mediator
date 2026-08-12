package handlers

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeLibraryPath(t *testing.T) {
	root := "/"
	if runtime.GOOS == "windows" {
		root = `C:\`
	}
	abs := func(parts ...string) string {
		return filepath.Join(append([]string{root}, parts...)...)
	}

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"absolute simple", abs("data", "movies"), abs("data", "movies"), false},
		{"absolute nested", abs("data", "movies", "hd"), abs("data", "movies", "hd"), false},
		{"absolute cleaned", abs("data", "movies", "..", "series"), abs("data", "series"), false},
		{"absolute trailing slash", abs("data", "movies") + string(filepath.Separator), abs("data", "movies"), false},
		{"relative rejected", "movies", "", true},
		{"relative nested rejected", filepath.Join("movies", "hd"), "", true},
		{"relative escape rejected", "..", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeLibraryPath(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
