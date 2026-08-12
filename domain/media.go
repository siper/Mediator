package domain

type MediaStatus string

const (
	MediaStatusContinuing MediaStatus = "continuing"
	MediaStatusCompleted  MediaStatus = "completed"
)

type Media struct {
	Id               ID
	Name             string
	OriginalName     string
	Folder           *string
	Cover            *string
	Type             MediaType
	LibraryID        *ID
	ProviderID       string
	ExternalID       string
	Status           MediaStatus
	LastModified     string
	QualityProfileID *ID
}

func (m *Media) Validate() error {
	if m.Name == "" {
		return ErrEmptyName
	}
	switch m.Type {
	case MediaTypeBook, MediaTypeSeries, MediaTypeMovie, MediaTypeMusicAlbum:
		return nil
	default:
		return ErrInvalidMediaType
	}
}
