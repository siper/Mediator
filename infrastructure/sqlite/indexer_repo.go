package sqlite

import (
	"database/sql"

	"stersh.ru/mediator/domain"
)

type SQLiteIndexerRepository struct {
	db DBTX
}

func NewSQLiteIndexerRepository(db DBTX) *SQLiteIndexerRepository {
	return &SQLiteIndexerRepository{db: db}
}

func (r *SQLiteIndexerRepository) Add(i *domain.Indexer) error {
	res, err := r.db.Exec(
		"INSERT INTO indexers (name, type, settings, enabled) VALUES (?, ?, ?, ?)",
		i.Name, i.Type, encodeSettings(i.Settings), boolToInt(i.Enabled),
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

func (r *SQLiteIndexerRepository) GetById(id domain.ID) (*domain.Indexer, error) {
	row := r.db.QueryRow(
		"SELECT id, name, type, settings, enabled FROM indexers WHERE id = ?", id,
	)
	i, err := scanIndexer(row.Scan)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrIndexerNotFound
		}
		return nil, err
	}
	return i, nil
}

func (r *SQLiteIndexerRepository) List() ([]domain.Indexer, error) {
	rows, err := r.db.Query("SELECT id, name, type, settings, enabled FROM indexers ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []domain.Indexer{}
	for rows.Next() {
		i, err := scanIndexer(rows.Scan)
		if err != nil {
			return nil, err
		}
		result = append(result, *i)
	}
	return result, rows.Err()
}

func (r *SQLiteIndexerRepository) Update(i *domain.Indexer) error {
	res, err := r.db.Exec(
		"UPDATE indexers SET name = ?, type = ?, settings = ?, enabled = ? WHERE id = ?",
		i.Name, i.Type, encodeSettings(i.Settings), boolToInt(i.Enabled), i.Id,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrIndexerNotFound
	}
	return nil
}

func (r *SQLiteIndexerRepository) Remove(id domain.ID) error {
	res, err := r.db.Exec("DELETE FROM indexers WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrIndexerNotFound
	}
	return nil
}

func scanIndexer(scan scanFn) (*domain.Indexer, error) {
	var id int64
	var name, typ, settingsJSON string
	var enabled int
	if err := scan(&id, &name, &typ, &settingsJSON, &enabled); err != nil {
		return nil, err
	}
	return &domain.Indexer{
		Id:       domain.ID(id),
		Name:     name,
		Type:     typ,
		Settings: decodeSettings(settingsJSON),
		Enabled:  enabled != 0,
	}, nil
}
