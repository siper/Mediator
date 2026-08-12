package domain

type Session struct {
	Id          ID
	UserId      ID
	RefreshHash string
	ExpiresAt   string
	CreatedAt   string
	Revoked     bool
}

type SessionRepository interface {
	Add(s *Session) error
	GetByRefreshHash(hash string) (*Session, error)
	RevokeByTokenHash(hash string) error
	DeleteExpired() error
	RevokeAllForUser(userId ID) error
}
