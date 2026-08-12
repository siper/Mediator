package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOSService_MoveAndInspect(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "stage")
	dstDir := filepath.Join(dir, "lib", "Movie")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))

	src := filepath.Join(srcDir, "file.mkv")
	require.NoError(t, os.WriteFile(src, []byte("payload"), 0o644))

	s := NewOSService()
	dst := filepath.Join(dstDir, "Movie.mkv")

	require.NoError(t, s.Move(src, dst))

	assert.False(t, s.Exists(src))
	assert.True(t, s.Exists(dst))

	size, err := s.Size(dst)
	require.NoError(t, err)
	assert.EqualValues(t, len("payload"), size)

	require.NoError(t, s.Remove(dst))
	assert.False(t, s.Exists(dst))
}

func TestOSService_MoveMissingSource(t *testing.T) {
	s := NewOSService()
	assert.Error(t, s.Move(filepath.Join(t.TempDir(), "nope"), filepath.Join(t.TempDir(), "dst")))
}
