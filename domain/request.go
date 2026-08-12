package domain

import "fmt"

type MediaRequestStatus string

const (
	MediaRequestPending  MediaRequestStatus = "pending"
	MediaRequestApproved MediaRequestStatus = "approved"
	MediaRequestRejected MediaRequestStatus = "rejected"
	MediaRequestCanceled MediaRequestStatus = "canceled"
)

type MediaRequest struct {
	Id               ID
	UserId           ID
	Provider         string
	ExternalID       string
	Title            string
	Cover            string
	Type             MediaType
	LibraryID        ID
	QualityProfileID *ID
	Folder           *string
	Status           MediaRequestStatus
	CreatedAt        string
	ApprovedAt       *string
	ApprovedBy       *ID
	RejectedAt       *string
	RejectedBy       *ID
	CanceledAt       *string
	Notes            *string
	MediaID          *ID
}

func (r *MediaRequest) CanEdit() bool {
	return r != nil && r.Status == MediaRequestPending
}

func (r *MediaRequest) Validate() error {
	if r.Provider == "" {
		return ErrEmptyName
	}
	if r.ExternalID == "" {
		return fmt.Errorf("external id cannot be empty")
	}
	if r.LibraryID == 0 {
		return ErrLibraryRequired
	}
	return nil
}

type MediaRequestRepository interface {
	Add(r *MediaRequest) error
	GetByID(id ID) (*MediaRequest, error)
	ListByUser(userID ID, page int, limit int) ([]MediaRequest, error)
	ListAll(page int, limit int) ([]MediaRequest, error)
	Update(r *MediaRequest) error
	Remove(id ID) error
	ExistsByProviderExternal(provider string, externalID string) (bool, error)
}
