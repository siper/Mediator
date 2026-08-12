package application

import (
	"errors"
	"fmt"

	"stersh.ru/mediator/domain"
)

type QueueService struct {
	queueRepo domain.QueueRepository
	mediaRepo domain.MediaRepository
	partRepo  domain.PartRepository
	groupRepo domain.PartGroupRepository
}

func NewQueueService(
	queueRepo domain.QueueRepository,
	mediaRepo domain.MediaRepository,
	partRepo domain.PartRepository,
	groupRepo domain.PartGroupRepository,
) *QueueService {
	return &QueueService{
		queueRepo: queueRepo,
		mediaRepo: mediaRepo,
		partRepo:  partRepo,
		groupRepo: groupRepo,
	}
}

func (s *QueueService) ListItems(page, limit int) ([]domain.QueueListItem, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	items, err := s.queueRepo.List(page, limit)
	if err != nil {
		return nil, err
	}

	mediaByID := make(map[domain.ID]domain.Media)
	groupsByMedia := make(map[domain.ID][]domain.PartGroup)
	out := make([]domain.QueueListItem, 0, len(items))

	for _, item := range items {
		media, ok := mediaByID[item.MediaId]
		if !ok {
			m, err := s.mediaRepo.GetById(item.MediaId)
			if err != nil {
				if errors.Is(err, domain.ErrMediaNotFound) {
					media = domain.Media{Id: item.MediaId}
				} else {
					return nil, err
				}
			} else {
				media = *m
			}
			mediaByID[item.MediaId] = media
		}

		entry := domain.QueueListItem{QueueItem: item, Media: media}
		if label := s.partLabel(item, media, groupsByMedia); label != "" {
			entry.PartLabel = &label
		}
		out = append(out, entry)
	}
	return out, nil
}

func (s *QueueService) partLabel(
	item domain.QueueItem,
	media domain.Media,
	groupsByMedia map[domain.ID][]domain.PartGroup,
) string {
	if s.partRepo == nil || len(item.PartIds) == 0 {
		return ""
	}
	part, err := s.partRepo.GetById(item.PartIds[0])
	if err != nil || part == nil {
		return ""
	}

	if media.Type != domain.MediaTypeSeries {
		if part.Name != nil {
			return *part.Name
		}
		return ""
	}

	var season *int
	if part.GroupId != nil && s.groupRepo != nil {
		groups, ok := groupsByMedia[part.MediaId]
		if !ok {
			groups, err = s.groupRepo.GetByMediaID(part.MediaId)
			if err != nil {
				groups = nil
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

	ep := part.GroupOrder
	var code string
	if season != nil && ep != nil {
		code = fmt.Sprintf("S%02dE%02d", *season, *ep)
	} else if ep != nil {
		code = fmt.Sprintf("E%02d", *ep)
	}
	if code != "" && part.Name != nil && *part.Name != "" {
		return code + " · " + *part.Name
	}
	if code != "" {
		return code
	}
	if part.Name != nil {
		return *part.Name
	}
	return ""
}
