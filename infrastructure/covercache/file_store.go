package covercache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"stersh.ru/mediator/domain"
)

const httpTimeout = 15 * time.Second

type FileCoverStore struct {
	root       string
	urlPrefix  string
	httpClient *http.Client
}

func NewFileCoverStore(root, urlPrefix string) *FileCoverStore {
	return &FileCoverStore{
		root:       root,
		urlPrefix:  strings.TrimRight(urlPrefix, "/"),
		httpClient: &http.Client{Timeout: httpTimeout},
	}
}

func (s *FileCoverStore) Store(ctx context.Context, sourceURL string) (string, error) {
	if sourceURL == "" {
		return "", nil
	}

	rel, err := s.relativePath(sourceURL)
	if err != nil {
		return "", err
	}

	full := filepath.Join(s.root, rel)
	if _, err := os.Stat(full); err == nil {
		return s.servedURL(rel), nil
	}

	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cover download failed: %s", resp.Status)
	}

	tmp, err := os.CreateTemp(filepath.Dir(full), ".tmp-*")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return "", err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return "", err
	}

	if err := os.Rename(tmpPath, full); err != nil {
		os.Remove(tmpPath)
		return "", err
	}

	return s.servedURL(rel), nil
}

func (s *FileCoverStore) relativePath(sourceURL string) (string, error) {
	ext := filepath.Ext(sourceURL)
	if i := strings.IndexAny(ext, "?#"); i >= 0 {
		ext = ext[:i]
	}
	if ext == "" {
		ext = ".jpg"
	}

	sum := sha256.Sum256([]byte(sourceURL))
	h := hex.EncodeToString(sum[:])
	return filepath.Join(h[:2], h[2:4], h+ext), nil
}

func (s *FileCoverStore) servedURL(rel string) string {
	return s.urlPrefix + "/" + filepath.ToSlash(rel)
}

var _ domain.CoverStore = (*FileCoverStore)(nil)
