package sqlite

import (
	"database/sql"

	"stersh.ru/mediator/domain"
)

type SQLiteSettingRepository struct {
	db DBTX
}

func NewSQLiteSettingRepository(db DBTX) *SQLiteSettingRepository {
	return &SQLiteSettingRepository{db: db}
}

func (r *SQLiteSettingRepository) Get(key string) (string, error) {
	var value string
	err := r.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", domain.ErrSettingNotFound
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

func (r *SQLiteSettingRepository) Set(key string, value string) error {
	_, err := r.db.Exec("INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value", key, value)
	return err
}

func (r *SQLiteSettingRepository) List() ([]domain.Setting, error) {
	rows, err := r.db.Query("SELECT key, value FROM settings ORDER BY key")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.Setting{}
	for rows.Next() {
		var s domain.Setting
		if err := rows.Scan(&s.Key, &s.Value); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}
