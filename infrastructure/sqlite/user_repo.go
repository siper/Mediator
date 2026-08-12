package sqlite

import (
	"database/sql"
	"strings"

	"stersh.ru/mediator/domain"
)

type SQLiteUserRepository struct {
	db DBTX
}

func NewSQLiteUserRepository(db DBTX) *SQLiteUserRepository {
	return &SQLiteUserRepository{db: db}
}

func (r *SQLiteUserRepository) Add(u *domain.User) error {
	res, err := r.db.Exec(
		"INSERT INTO users (name, email, password_hash, role, created_at) VALUES (?, ?, ?, ?, ?)",
		u.Name, u.Email, u.PasswordHash, string(u.Role), u.CreatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return domain.ErrDuplicateEmail
		}
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	u.Id = domain.ID(id)
	return nil
}

func (r *SQLiteUserRepository) GetByID(id domain.ID) (*domain.User, error) {
	row := r.db.QueryRow("SELECT id, name, email, password_hash, role, created_at FROM users WHERE id = ?", id)
	return scanUser(row.Scan)
}

func (r *SQLiteUserRepository) GetByEmail(email string) (*domain.User, error) {
	row := r.db.QueryRow("SELECT id, name, email, password_hash, role, created_at FROM users WHERE email = ?", email)
	return scanUser(row.Scan)
}

func (r *SQLiteUserRepository) List(page int, limit int) ([]domain.User, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit
	rows, err := r.db.Query("SELECT id, name, email, password_hash, role, created_at FROM users ORDER BY id LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.User{}
	for rows.Next() {
		u, err := scanUser(rows.Scan)
		if err != nil {
			return nil, err
		}
		result = append(result, *u)
	}
	return result, rows.Err()
}

func (r *SQLiteUserRepository) Update(u *domain.User) error {
	res, err := r.db.Exec(
		"UPDATE users SET name = ?, email = ?, password_hash = ?, role = ? WHERE id = ?",
		u.Name, u.Email, u.PasswordHash, string(u.Role), u.Id,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return domain.ErrDuplicateEmail
		}
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *SQLiteUserRepository) Remove(id domain.ID) error {
	res, err := r.db.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *SQLiteUserRepository) Count() (int, error) {
	var n int
	err := r.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&n)
	return n, err
}

func scanUser(scan scanFn) (*domain.User, error) {
	var (
		id           int64
		name         string
		email        string
		passwordHash string
		role         string
		createdAt    string
	)
	if err := scan(&id, &name, &email, &passwordHash, &role, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &domain.User{
		Id:           domain.ID(id),
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         domain.Role(role),
		CreatedAt:    createdAt,
	}, nil
}
