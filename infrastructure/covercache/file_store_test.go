package covercache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func relativePathFor(t *testing.T, sourceURL string) string {
	t.Helper()
	ext := filepath.Ext(sourceURL)
	if i := strings.IndexAny(ext, "?#"); i >= 0 {
		ext = ext[:i]
	}
	if ext == "" {
		ext = ".jpg"
	}
	sum := sha256.Sum256([]byte(sourceURL))
	h := hex.EncodeToString(sum[:])
	return filepath.Join(h[:2], h[2:4], h+ext)
}

func TestFileCoverStore_Store(t *testing.T) {
	t.Run("empty url returns empty without error", func(t *testing.T) {
		s := NewFileCoverStore(t.TempDir(), "/covers")
		got, err := s.Store(context.Background(), "")
		assert.NoError(t, err)
		assert.Equal(t, "", got)
	})

	t.Run("downloads and stores, returns served url", func(t *testing.T) {
		var hits int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&hits, 1)
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, "binary-bytes")
		}))
		defer srv.Close()

		root := t.TempDir()
		s := NewFileCoverStore(root, "/covers")

		got, err := s.Store(context.Background(), srv.URL+"/img.jpg")
		require.NoError(t, err)

		rel := relativePathFor(t, srv.URL+"/img.jpg")
		assert.Equal(t, "/covers/"+filepath.ToSlash(rel), got)

		info, err := os.Stat(filepath.Join(root, rel))
		require.NoError(t, err)
		assert.Equal(t, int64(len("binary-bytes")), info.Size())
		assert.Equal(t, int32(1), atomic.LoadInt32(&hits))
	})

	t.Run("idempotent: second call skips download", func(t *testing.T) {
		var hits int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&hits, 1)
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, "binary-bytes")
		}))
		defer srv.Close()

		s := NewFileCoverStore(t.TempDir(), "/covers")
		url := srv.URL + "/img.jpg"

		first, err := s.Store(context.Background(), url)
		require.NoError(t, err)

		second, err := s.Store(context.Background(), url)
		require.NoError(t, err)

		assert.Equal(t, first, second)
		assert.Equal(t, int32(1), atomic.LoadInt32(&hits))
	})

	t.Run("non-200 returns error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		s := NewFileCoverStore(t.TempDir(), "/covers")
		_, err := s.Store(context.Background(), srv.URL+"/missing.jpg")
		assert.Error(t, err)
	})

	t.Run("server unreachable returns error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		srv.Close()

		s := NewFileCoverStore(t.TempDir(), "/covers")
		_, err := s.Store(context.Background(), srv.URL+"/img.jpg")
		assert.Error(t, err)
	})

	t.Run("query params stripped from extension", func(t *testing.T) {
		var hits int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&hits, 1)
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, "binary-bytes")
		}))
		defer srv.Close()

		root := t.TempDir()
		s := NewFileCoverStore(root, "/covers")
		url := srv.URL + "/img.jpg?width=153&height=200"

		got, err := s.Store(context.Background(), url)
		require.NoError(t, err)

		assert.True(t, strings.HasSuffix(got, ".jpg"), "served url should end with .jpg, got %q", got)
		assert.NotContains(t, got, "?", "served url must not contain query string")

		rel := relativePathFor(t, url)
		assert.Equal(t, "/covers/"+filepath.ToSlash(rel), got)
		_, err = os.Stat(filepath.Join(root, rel))
		require.NoError(t, err)
	})
}
