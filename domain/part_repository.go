package domain

type PartRepository interface {
	GetById(id ID) (*Part, error)
	Add(part *Part) error
	GetByMediaId(mediaId ID) ([]Part, error)
	GetWanted(page int, limit int) ([]Part, error)
	Remove(id ID) error
	Update(part *Part) error
}
