package domain

type OIDCIdentity struct {
	Id     ID
	UserId ID
	Issuer string
	Subject string
}

type OIDCIdentityRepository interface {
	GetByIssuerSubject(issuer, subject string) (*OIDCIdentity, error)
	Add(i *OIDCIdentity) error
	RemoveByUserID(userId ID) error
}
