package domain

import (
	"context"
	"fmt"
	"strings"
)

type SourceType string

const (
	SourceTMDB        SourceType = "tmdb"
	SourceAuthorToday SourceType = "author_today"
	SourceMusicBrainz SourceType = "musicbrainz"
	SourceOpenLibrary SourceType = "openlibrary"
)

var validSourceTypes = map[SourceType]bool{
	SourceTMDB:        true,
	SourceAuthorToday: true,
	SourceMusicBrainz: true,
	SourceOpenLibrary: true,
}

type Source struct {
	Id          ID
	Type        string
	Name        string
	Settings    map[string]string
	Enabled     bool
	ProxyID     *ID
}

func (s *Source) Get(key string) string {
	if s == nil || s.Settings == nil {
		return ""
	}
	return s.Settings[key]
}

func (s *Source) ApplyDefaultName() {
	if s == nil {
		return
	}
	s.Name = strings.TrimSpace(s.Name)
	if s.Name != "" {
		return
	}
	switch SourceType(s.Type) {
	case SourceTMDB:
		s.Name = "TMDB"
	case SourceAuthorToday:
		s.Name = "Author.today"
	case SourceMusicBrainz:
		s.Name = "MusicBrainz"
	case SourceOpenLibrary:
		s.Name = "Open Library"
	default:
		s.Name = s.Type
	}
}

func (s *Source) Validate() error {
	s.ApplyDefaultName()
	if s.Name == "" {
		return ErrEmptyName
	}
	if !validSourceTypes[SourceType(s.Type)] {
		return fmt.Errorf("%w: %s", ErrInvalidSourceType, s.Type)
	}
	switch SourceType(s.Type) {
	case SourceTMDB:
		if s.Get("api_key") == "" {
			return fmt.Errorf("tmdb: api_key is required")
		}
	case SourceAuthorToday:
		return nil
	case SourceMusicBrainz:
		return nil
	case SourceOpenLibrary:
		return nil
	}
	return nil
}

type SourceRepository interface {
	Add(s *Source) error
	GetByID(id ID) (*Source, error)
	List() ([]Source, error)
	Update(s *Source) error
	Remove(id ID) error
}

type MetadataSource interface {
	Active() []MediaProvider
}

type SourceReloader interface {
	Reload() error
}

type SourceTester interface {
	Test(ctx context.Context, s Source) error
}
