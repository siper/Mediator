package sqlite

import (
	"database/sql"

	"stersh.ru/mediator/domain"
)

type SQLiteLibraryRepository struct {
	db DBTX
}

func NewSQLiteLibraryRepository(db DBTX) *SQLiteLibraryRepository {
	return &SQLiteLibraryRepository{db: db}
}

func (r *SQLiteLibraryRepository) Add(l *domain.Library) error {
	res, err := r.db.Exec("INSERT INTO libraries (name, path, type, settings) VALUES (?, ?, ?, ?)", l.Name, l.Path, l.Type, encodeSettings(l.Settings))
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	l.Id = domain.ID(id)
	return nil
}

func (r *SQLiteLibraryRepository) GetById(id domain.ID) (*domain.Library, error) {
	row := r.db.QueryRow("SELECT id, name, path, type, settings FROM libraries WHERE id = ?", id)
	l, err := scanLibrary(row.Scan)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrLibraryNotFound
		}
		return nil, err
	}
	return l, nil
}

func (r *SQLiteLibraryRepository) List() ([]domain.Library, error) {
	rows, err := r.db.Query("SELECT id, name, path, type, settings FROM libraries ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []domain.Library{}
	for rows.Next() {
		l, err := scanLibrary(rows.Scan)
		if err != nil {
			return nil, err
		}
		result = append(result, *l)
	}
	return result, rows.Err()
}

func (r *SQLiteLibraryRepository) ListByType(t domain.MediaType) ([]domain.Library, error) {
	rows, err := r.db.Query("SELECT id, name, path, type, settings FROM libraries WHERE type = ? ORDER BY id", t)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []domain.Library{}
	for rows.Next() {
		l, err := scanLibrary(rows.Scan)
		if err != nil {
			return nil, err
		}
		result = append(result, *l)
	}
	return result, rows.Err()
}

func (r *SQLiteLibraryRepository) Update(l *domain.Library) error {
	res, err := r.db.Exec("UPDATE libraries SET name = ?, path = ?, type = ?, settings = ? WHERE id = ?", l.Name, l.Path, l.Type, encodeSettings(l.Settings), l.Id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrLibraryNotFound
	}
	return nil
}

func (r *SQLiteLibraryRepository) Remove(id domain.ID) error {
	res, err := r.db.Exec("DELETE FROM libraries WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrLibraryNotFound
	}
	return nil
}

func scanLibrary(scan scanFn) (*domain.Library, error) {
	var (
		id          int64
		name        string
		path        string
		t           int
		settingsJSON string
	)
	if err := scan(&id, &name, &path, &t, &settingsJSON); err != nil {
		return nil, err
	}
	return &domain.Library{Id: domain.ID(id), Name: name, Path: path, Type: domain.MediaType(t), Settings: decodeSettings(settingsJSON)}, nil
}
