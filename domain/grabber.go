package domain

import "context"

type GrabState string

const (
	GrabQueued    GrabState = "queued"
	GrabRunning   GrabState = "running"
	GrabCompleted GrabState = "completed"
	GrabFailed    GrabState = "failed"
	GrabCanceled  GrabState = "canceled"
)

func (s GrabState) Active() bool {
	return s == GrabQueued || s == GrabRunning
}

type GrabTarget struct {
	MediaID      ID
	PartIds      []ID
	Release      *Release
	URL          string
	Profile      *QualityProfile
	ProviderName string
}

type GrabHandle struct {
	GrabberName string
	JobID       string
	Name        string
}

type GrabStatus struct {
	State       GrabState
	Progress    float64
	OutputFiles []string
	Error       string
	ResolvedID  string
}

type Grabber interface {
	Name() string
	Supports(target GrabTarget) bool
	SupportsProvider(providerName string) bool
	Submit(ctx context.Context, target GrabTarget) (GrabHandle, error)
	Status(ctx context.Context, handle GrabHandle) (GrabStatus, error)
	Cancel(ctx context.Context, handle GrabHandle) error
}

type GrabberSink interface {
	SetGrabbers(grabbers []Grabber)
}

type ClientReloader interface {
	Reload() error
}
