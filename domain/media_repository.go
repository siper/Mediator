package domain

type MediaRepository interface {
	GetById(Id ID) (*Media, error)
	Create(Name string, OriginalName string, Folder *string, Cover *string, Type MediaType, LibraryID *ID, ProviderID string, ExternalID string, ProfileID *ID) (*Media, error)
	Update(Id ID, Name string, OriginalName string, Cover *string) error
	UpdateProfile(Id ID, ProfileID *ID) error
	UpdateProviderMeta(Id ID, Status MediaStatus, LastModified string) error
	Remove(Id ID) error
	GetPaged(Page int, Limit int, Type *MediaType) ([]Media, error)
	GetByProviderExternal(provider string, externalID string) (*Media, error)
}
