package application

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"stersh.ru/mediator/domain"
)

type MediaService struct {
	repo        domain.MediaRepository
	partRepo    domain.PartRepository
	libraryRepo domain.LibraryRepository
	fs          domain.FileService
}

func NewMediaService(
	repo domain.MediaRepository,
	partRepo domain.PartRepository,
	libraryRepo domain.LibraryRepository,
	fs domain.FileService,
) *MediaService {
	return &MediaService{
		repo:        repo,
		partRepo:    partRepo,
		libraryRepo: libraryRepo,
		fs:          fs,
	}
}

func (s *MediaService) Create(name string, originalName string, folder *string, cover *string, mediaType domain.MediaType, libraryID *domain.ID, providerID string, externalID string, profileID *domain.ID) (*domain.Media, error) {
	m := &domain.Media{Name: name, OriginalName: originalName, Folder: folder, Cover: cover, Type: mediaType, LibraryID: libraryID, ProviderID: providerID, ExternalID: externalID, QualityProfileID: profileID}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return s.repo.Create(name, originalName, folder, cover, mediaType, libraryID, providerID, externalID, profileID)
}

func (s *MediaService) GetByID(id domain.ID) (*domain.Media, error) {
	return s.repo.GetById(id)
}

func (s *MediaService) Update(id domain.ID, name string, originalName string, cover *string) error {
	return s.repo.Update(id, name, originalName, cover)
}

func (s *MediaService) UpdateProfile(id domain.ID, profileID *domain.ID) error {
	return s.repo.UpdateProfile(id, profileID)
}

func (s *MediaService) UpdateProviderMeta(id domain.ID, status domain.MediaStatus, lastModified string) error {
	return s.repo.UpdateProviderMeta(id, status, lastModified)
}

func (s *MediaService) Remove(id domain.ID, deleteFiles bool) error {
	if deleteFiles {
		if err := s.deleteMediaFiles(id); err != nil {
			return err
		}
	}
	return s.repo.Remove(id)
}

func (s *MediaService) deleteMediaFiles(id domain.ID) error {
	if s.partRepo == nil || s.fs == nil {
		return nil
	}
	parts, err := s.partRepo.GetByMediaId(id)
	if err != nil {
		return err
	}
	var paths []string
	for _, p := range parts {
		if p.Path == nil || *p.Path == "" {
			continue
		}
		paths = append(paths, filepath.Clean(*p.Path))
	}
	if len(paths) == 0 {
		return nil
	}

	if root := s.safeMediaRoot(id, paths); root != "" {
		if err := s.fs.RemoveAll(root); err != nil && !errors.Is(err, os.ErrNotExist) {
			slog.Warn("media service: failed to delete media folder", "path", root, "err", err)
		}
		return nil
	}

	for _, path := range paths {
		if err := s.fs.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			slog.Warn("media service: failed to delete file", "path", path, "err", err)
		}
	}
	return nil
}

func (s *MediaService) safeMediaRoot(id domain.ID, paths []string) string {
	root := commonMediaRoot(paths)
	if root == "" {
		return ""
	}
	if s.libraryRepo == nil {
		return ""
	}
	media, err := s.repo.GetById(id)
	if err != nil || media.LibraryID == nil {
		return ""
	}
	lib, err := s.libraryRepo.GetById(*media.LibraryID)
	if err != nil {
		return ""
	}
	libPath := filepath.Clean(lib.Path)
	if root == libPath || !pathHasPrefix(root, libPath) {
		return ""
	}
	return root
}

func commonMediaRoot(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	dirs := make([]string, len(paths))
	for i, p := range paths {
		dir := filepath.Clean(filepath.Dir(p))
		if dir == "." || dir == string(filepath.Separator) {
			return ""
		}
		dirs[i] = dir
	}
	common := dirs[0]
	for _, d := range dirs[1:] {
		for !pathHasPrefix(d, common) {
			parent := filepath.Dir(common)
			if parent == common || parent == "." || parent == string(filepath.Separator) {
				return ""
			}
			common = parent
		}
	}
	for _, p := range paths {
		rel, err := filepath.Rel(common, p)
		if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
			return ""
		}
	}
	return common
}

func pathHasPrefix(path, prefix string) bool {
	path = filepath.Clean(path)
	prefix = filepath.Clean(prefix)
	if path == prefix {
		return true
	}
	sep := string(filepath.Separator)
	return strings.HasPrefix(path, prefix+sep)
}

func (s *MediaService) GetPaged(page, limit int, mediaType *domain.MediaType) ([]domain.Media, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.GetPaged(page, limit, mediaType)
}
