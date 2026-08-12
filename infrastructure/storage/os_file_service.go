package storage

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"

	"stersh.ru/mediator/domain"
)

type OSService struct{}

func NewOSService() *OSService { return &OSService{} }

func (s *OSService) Move(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	} else if !errors.Is(err, syscall.EXDEV) {
		return err
	}
	copyErr := copyFile(src, dst)
	if copyErr != nil {
		return copyErr
	}
	if rmErr := os.Remove(src); rmErr != nil {
		slog.Warn("file service: source file could not be removed after cross-device copy", "src", src, "err", rmErr)
	}
	return nil
}

func (s *OSService) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (s *OSService) Size(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (s *OSService) Remove(path string) error {
	return os.Remove(path)
}

func (s *OSService) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (s *OSService) ListFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}
	return out.Close()
}

var _ domain.FileService = (*OSService)(nil)
