package domain

import "time"

type QueueItem struct {
	Id            ID
	MediaId       ID
	PartIds       []ID
	GrabberName   string
	JobID         string
	DownloadID    string
	ReleaseTitle  string
	State         GrabState
	Progress      float64
	AddedAt       time.Time
}

type QueueListItem struct {
	QueueItem
	Media     Media
	PartLabel *string
}

type QueueRepository interface {
	Add(q *QueueItem) error
	GetById(id ID) (*QueueItem, error)
	List(page int, limit int) ([]QueueItem, error)
	ListActive() ([]QueueItem, error)
	Update(q *QueueItem) error
	Remove(id ID) error
}
