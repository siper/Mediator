package domain

type QualityProfile struct {
	Id      ID
	Name    string
	Type    MediaType
	Allowed []Quality
	Cutoff  Quality
}

func (p *QualityProfile) Validate() error {
	if p.Name == "" {
		return ErrEmptyName
	}
	if p.Type != MediaTypeMovie && p.Type != MediaTypeSeries && p.Type != MediaTypeMusicAlbum {
		return ErrProfileTypeNotAllowed
	}
	kind, ok := QualityKindFor(p.Type)
	if !ok {
		return ErrInvalidMediaType
	}
	for _, q := range p.Allowed {
		if q.Kind != kind || !q.Valid() {
			return ErrInvalidQuality
		}
	}
	if p.Cutoff != (Quality{}) {
		if p.Cutoff.Kind != kind || !p.Cutoff.Valid() {
			return ErrInvalidQuality
		}
	}
	return nil
}

type QualityProfileRepository interface {
	Add(p *QualityProfile) error
	GetById(id ID) (*QualityProfile, error)
	List() ([]QualityProfile, error)
	ListByType(t MediaType) ([]QualityProfile, error)
	Update(p *QualityProfile) error
	Remove(id ID) error
}
