package domain

type Part struct {
	Id         ID
	Name       *string
	GroupOrder *int
	GroupId    *ID
	MediaId    ID
	Path       *string
	Monitored  bool
}

func (p *Part) Validate() error {
	if p.MediaId == 0 {
		return ErrMediaNotFound
	}
	if p.Name != nil && *p.Name == "" {
		return ErrEmptyName
	}
	return nil
}

type PartGroup struct {
	Id      ID
	Name    string
	Order   int
	MediaId ID
}
