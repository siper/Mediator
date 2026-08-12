package sqlite

import (
	"database/sql"

	"stersh.ru/mediator/domain"
)

type SQLiteDownloadClientRepository struct {
	db DBTX
}

func NewSQLiteDownloadClientRepository(db DBTX) *SQLiteDownloadClientRepository {
	return &SQLiteDownloadClientRepository{db: db}
}

func (r *SQLiteDownloadClientRepository) Add(c *domain.DownloadClient) error {
	res, err := r.db.Exec(
		"INSERT INTO download_clients (name, type, settings, enabled) VALUES (?, ?, ?, ?)",
		c.Name, c.Type, encodeSettings(c.Settings), boolToInt(c.Enabled),
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	c.Id = domain.ID(id)
	return nil
}

func (r *SQLiteDownloadClientRepository) GetById(id domain.ID) (*domain.DownloadClient, error) {
	row := r.db.QueryRow(
		"SELECT id, name, type, settings, enabled FROM download_clients WHERE id = ?", id,
	)
	c, err := scanDownloadClient(row.Scan)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrDownloadClientNotFound
		}
		return nil, err
	}
	return c, nil
}

func (r *SQLiteDownloadClientRepository) List() ([]domain.DownloadClient, error) {
	rows, err := r.db.Query("SELECT id, name, type, settings, enabled FROM download_clients ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []domain.DownloadClient{}
	for rows.Next() {
		c, err := scanDownloadClient(rows.Scan)
		if err != nil {
			return nil, err
		}
		result = append(result, *c)
	}
	return result, rows.Err()
}

func (r *SQLiteDownloadClientRepository) Update(c *domain.DownloadClient) error {
	res, err := r.db.Exec(
		"UPDATE download_clients SET name = ?, type = ?, settings = ?, enabled = ? WHERE id = ?",
		c.Name, c.Type, encodeSettings(c.Settings), boolToInt(c.Enabled), c.Id,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrDownloadClientNotFound
	}
	return nil
}

func (r *SQLiteDownloadClientRepository) Remove(id domain.ID) error {
	res, err := r.db.Exec("DELETE FROM download_clients WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrDownloadClientNotFound
	}
	return nil
}

func scanDownloadClient(scan scanFn) (*domain.DownloadClient, error) {
	var id int64
	var name, ctype, settingsJSON string
	var enabled int
	if err := scan(&id, &name, &ctype, &settingsJSON, &enabled); err != nil {
		return nil, err
	}
	return &domain.DownloadClient{
		Id:       domain.ID(id),
		Name:     name,
		Type:     domain.DownloadClientType(ctype),
		Settings: decodeSettings(settingsJSON),
		Enabled:  enabled != 0,
	}, nil
}
