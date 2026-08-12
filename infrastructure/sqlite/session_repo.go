package sqlite

import (
	"database/sql"

	"stersh.ru/mediator/domain"
)

type SQLiteSessionRepository struct {
	db DBTX
}

func NewSQLiteSessionRepository(db DBTX) *SQLiteSessionRepository {
	return &SQLiteSessionRepository{db: db}
}

func (r *SQLiteSessionRepository) Add(s *domain.Session) error {
	res, err := r.db.Exec(
		"INSERT INTO user_sessions (user_id, refresh_token_hash, expires_at, created_at, revoked) VALUES (?, ?, ?, ?, ?)",
		s.UserId, s.RefreshHash, s.ExpiresAt, s.CreatedAt, boolToInt(s.Revoked),
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	s.Id = domain.ID(id)
	return nil
}

func (r *SQLiteSessionRepository) GetByRefreshHash(hash string) (*domain.Session, error) {
	row := r.db.QueryRow(
		"SELECT id, user_id, refresh_token_hash, expires_at, created_at, revoked FROM user_sessions WHERE refresh_token_hash = ?",
		hash,
	)
	var (
		id        int64
		userId    int64
		rth       string
		expiresAt string
		createdAt string
		revoked   int
	)
	if err := row.Scan(&id, &userId, &rth, &expiresAt, &createdAt, &revoked); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrSessionNotFound
		}
		return nil, err
	}
	return &domain.Session{
		Id:          domain.ID(id),
		UserId:      domain.ID(userId),
		RefreshHash: rth,
		ExpiresAt:   expiresAt,
		CreatedAt:   createdAt,
		Revoked:     revoked != 0,
	}, nil
}

func (r *SQLiteSessionRepository) RevokeByTokenHash(hash string) error {
	res, err := r.db.Exec("UPDATE user_sessions SET revoked = 1 WHERE refresh_token_hash = ?", hash)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrSessionNotFound
	}
	return nil
}

func (r *SQLiteSessionRepository) DeleteExpired() error {
	_, err := r.db.Exec("DELETE FROM user_sessions WHERE expires_at < datetime('now')")
	return err
}

func (r *SQLiteSessionRepository) RevokeAllForUser(userId domain.ID) error {
	_, err := r.db.Exec("UPDATE user_sessions SET revoked = 1 WHERE user_id = ?", userId)
	return err
}
