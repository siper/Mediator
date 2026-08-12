package sqlite

import (
	"database/sql"

	"stersh.ru/mediator/domain"
)

type SQLiteOIDCIdentityRepository struct {
	db DBTX
}

func NewSQLiteOIDCIdentityRepository(db DBTX) *SQLiteOIDCIdentityRepository {
	return &SQLiteOIDCIdentityRepository{db: db}
}

func (r *SQLiteOIDCIdentityRepository) GetByIssuerSubject(issuer, subject string) (*domain.OIDCIdentity, error) {
	row := r.db.QueryRow(
		"SELECT id, user_id, issuer, subject FROM oidc_identities WHERE issuer = ? AND subject = ?",
		issuer, subject,
	)
	var (
		id      int64
		userId  int64
		iss     string
		sub     string
	)
	if err := row.Scan(&id, &userId, &iss, &sub); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrOIDCIdentityNotFound
		}
		return nil, err
	}
	return &domain.OIDCIdentity{
		Id:     domain.ID(id),
		UserId: domain.ID(userId),
		Issuer: iss,
		Subject: sub,
	}, nil
}

func (r *SQLiteOIDCIdentityRepository) Add(i *domain.OIDCIdentity) error {
	res, err := r.db.Exec(
		"INSERT INTO oidc_identities (user_id, issuer, subject) VALUES (?, ?, ?)",
		i.UserId, i.Issuer, i.Subject,
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	i.Id = domain.ID(id)
	return nil
}

func (r *SQLiteOIDCIdentityRepository) RemoveByUserID(userId domain.ID) error {
	_, err := r.db.Exec("DELETE FROM oidc_identities WHERE user_id = ?", userId)
	return err
}
