package domain

import "strings"

type Library struct {
	Id       ID
	Name     string
	Path     string
	Type     MediaType
	Settings map[string]string
}

func (l *Library) ApplyDefaultName() {
	if l == nil {
		return
	}
	l.Name = strings.TrimSpace(l.Name)
	if l.Name != "" {
		return
	}
	switch l.Type {
	case MediaTypeMovie:
		l.Name = "Movies"
	case MediaTypeSeries:
		l.Name = "Series"
	case MediaTypeBook:
		l.Name = "Books"
	case MediaTypeMusicAlbum:
		l.Name = "Music"
	default:
		l.Name = l.Type.String()
	}
}

func (l *Library) Validate() error {
	l.ApplyDefaultName()
	if l.Name == "" {
		return ErrEmptyName
	}
	if l.Path == "" {
		return ErrEmptyPath
	}
	switch l.Type {
	case MediaTypeBook, MediaTypeSeries, MediaTypeMovie, MediaTypeMusicAlbum:
		return nil
	default:
		return ErrInvalidMediaType
	}
}

type LibraryRepository interface {
	Add(l *Library) error
	GetById(id ID) (*Library, error)
	List() ([]Library, error)
	ListByType(t MediaType) ([]Library, error)
	Update(l *Library) error
	Remove(id ID) error
}
