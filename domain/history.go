package domain

import "time"

type HistoryEventType string

const (
	HistoryGrabbed  HistoryEventType = "grabbed"
	HistoryImported HistoryEventType = "imported"
	HistoryFailed   HistoryEventType = "failed"
)

type History struct {
	Id           ID
	MediaId      ID
	PartId       *ID
	EventType    HistoryEventType
	ReleaseTitle string
	Data         string
	CreatedAt    time.Time
}

type HistoryRepository interface {
	Add(h *History) error
	GetByMediaId(mediaId ID) ([]History, error)
	List(page int, limit int) ([]History, error)
}
