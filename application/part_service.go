package application

import "stersh.ru/mediator/domain"

type PartService struct {
	partRepo  domain.PartRepository
	mediaRepo domain.MediaRepository
	groupRepo domain.PartGroupRepository
}

func NewPartService(
	partRepo domain.PartRepository,
	mediaRepo domain.MediaRepository,
	groupRepo domain.PartGroupRepository,
) *PartService {
	return &PartService{partRepo: partRepo, mediaRepo: mediaRepo, groupRepo: groupRepo}
}

func (s *PartService) Add(part *domain.Part) error {
	if err := part.Validate(); err != nil {
		return err
	}
	_, err := s.mediaRepo.GetById(part.MediaId)
	if err != nil {
		return err
	}
	return s.partRepo.Add(part)
}

func (s *PartService) GetByID(id domain.ID) (*domain.Part, error) {
	return s.partRepo.GetById(id)
}

func (s *PartService) GetByMediaID(mediaID domain.ID) ([]domain.Part, error) {
	return s.partRepo.GetByMediaId(mediaID)
}

func (s *PartService) Remove(id domain.ID) error {
	return s.partRepo.Remove(id)
}

type PartPatch struct {
	Name       *string
	GroupOrder *int
	GroupId    *domain.ID
	Path       *string
	Monitored  *bool
}

func (s *PartService) Update(id domain.ID, patch PartPatch) (*domain.Part, error) {
	existing, err := s.partRepo.GetById(id)
	if err != nil {
		return nil, err
	}

	if patch.Name != nil {
		if *patch.Name == "" {
			return nil, domain.ErrEmptyName
		}
		existing.Name = patch.Name
	}
	if patch.GroupOrder != nil {
		existing.GroupOrder = patch.GroupOrder
	}
	if patch.GroupId != nil {
		existing.GroupId = patch.GroupId
	}
	if patch.Path != nil {
		existing.Path = patch.Path
	}
	if patch.Monitored != nil {
		existing.Monitored = *patch.Monitored
	}

	if err := existing.Validate(); err != nil {
		return nil, err
	}
	if err := s.partRepo.Update(existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *PartService) Wanted(page, limit int) ([]domain.Part, error) {
	page, limit = clampPageLimit(page, limit)
	return s.partRepo.GetWanted(page, limit)
}

func (s *PartService) WantedItems(page, limit int) ([]domain.WantedItem, error) {
	parts, err := s.Wanted(page, limit)
	if err != nil {
		return nil, err
	}

	mediaByID := make(map[domain.ID]*domain.Media)
	groupsByMedia := make(map[domain.ID][]domain.PartGroup)
	out := make([]domain.WantedItem, 0, len(parts))

	for _, part := range parts {
		media, ok := mediaByID[part.MediaId]
		if !ok {
			media, err = s.mediaRepo.GetById(part.MediaId)
			if err != nil {
				return nil, err
			}
			mediaByID[part.MediaId] = media
		}

		var season *int
		if part.GroupId != nil && s.groupRepo != nil {
			groups, ok := groupsByMedia[part.MediaId]
			if !ok {
				groups, err = s.groupRepo.GetByMediaID(part.MediaId)
				if err != nil {
					return nil, err
				}
				groupsByMedia[part.MediaId] = groups
			}
			for _, g := range groups {
				if g.Id == *part.GroupId {
					order := g.Order
					season = &order
					break
				}
			}
		}

		out = append(out, domain.WantedItem{
			Part:   part,
			Media:  *media,
			Season: season,
		})
	}
	return out, nil
}

func clampPageLimit(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}
