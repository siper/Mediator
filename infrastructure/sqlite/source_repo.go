package sqlite

import (
	"database/sql"

	"stersh.ru/mediator/domain"
)

type SQLiteSourceRepository struct {
	db DBTX
}

func NewSQLiteSourceRepository(db DBTX) *SQLiteSourceRepository {
	return &SQLiteSourceRepository{db: db}
}

func proxyIDValue(id *domain.ID) any {
	if id == nil {
		return nil
	}
	return int64(*id)
}

func (r *SQLiteSourceRepository) Add(s *domain.Source) error {
	res, err := r.db.Exec(
		"INSERT INTO sources (type, name, settings, enabled, proxy_id) VALUES (?, ?, ?, ?, ?)",
		s.Type, s.Name, encodeSettings(s.Settings), boolToInt(s.Enabled), proxyIDValue(s.ProxyID),
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

func (r *SQLiteSourceRepository) GetByID(id domain.ID) (*domain.Source, error) {
	row := r.db.QueryRow(
		"SELECT id, type, name, settings, enabled, proxy_id FROM sources WHERE id = ?", id,
	)
	s, err := scanSource(row.Scan)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrSourceNotFound
		}
		return nil, err
	}
	return s, nil
}

func (r *SQLiteSourceRepository) List() ([]domain.Source, error) {
	rows, err := r.db.Query("SELECT id, type, name, settings, enabled, proxy_id FROM sources ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []domain.Source{}
	for rows.Next() {
		s, err := scanSource(rows.Scan)
		if err != nil {
			return nil, err
		}
		result = append(result, *s)
	}
	return result, rows.Err()
}

func (r *SQLiteSourceRepository) Update(s *domain.Source) error {
	res, err := r.db.Exec(
		"UPDATE sources SET type = ?, name = ?, settings = ?, enabled = ?, proxy_id = ? WHERE id = ?",
		s.Type, s.Name, encodeSettings(s.Settings), boolToInt(s.Enabled), proxyIDValue(s.ProxyID), s.Id,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrSourceNotFound
	}
	return nil
}

func (r *SQLiteSourceRepository) Remove(id domain.ID) error {
	res, err := r.db.Exec("DELETE FROM sources WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrSourceNotFound
	}
	return nil
}

func scanSource(scan scanFn) (*domain.Source, error) {
	var id int64
	var typ, name, settingsJSON string
	var enabled int
	var proxyID sql.NullInt64
	if err := scan(&id, &typ, &name, &settingsJSON, &enabled, &proxyID); err != nil {
		return nil, err
	}
	s := &domain.Source{
		Id:       domain.ID(id),
		Type:     typ,
		Name:     name,
		Settings: decodeSettings(settingsJSON),
		Enabled:  enabled != 0,
	}
	if proxyID.Valid {
		pid := domain.ID(proxyID.Int64)
		s.ProxyID = &pid
	}
	return s, nil
}
