package domain

type PartGroupRepository interface {
	Add(group *PartGroup) error
	GetByMediaID(mediaID ID) ([]PartGroup, error)
	Remove(id ID) error
}
