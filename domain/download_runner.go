package domain

import "context"

type DownloadStatus string

const (
	DownloadQueued    DownloadStatus = "queued"
	Downloading       DownloadStatus = "downloading"
	DownloadCompleted DownloadStatus = "completed"
	DownloadSeeding   DownloadStatus = "seeding"
	DownloadFailed    DownloadStatus = "failed"
)

func (s DownloadStatus) Active() bool {
	return s == DownloadQueued || s == Downloading || s == DownloadSeeding
}

type DownloadTask struct {
	ID         string
	Name       string
	Status     DownloadStatus
	Progress   float64
	OutputPath string
}

type DownloadRunner interface {
	Name() string
	Add(ctx context.Context, r Release, category string) (*DownloadTask, error)
	Get(ctx context.Context, id string) (*DownloadTask, error)
	List(ctx context.Context) ([]DownloadTask, error)
	Remove(ctx context.Context, id string, deleteData bool) error
}
