package sqlite

import (
	"database/sql"

	"stersh.ru/mediator/domain"
)

type SQLitePartRepository struct {
	db DBTX
}

func NewSQLitePartRepository(db DBTX) *SQLitePartRepository {
	return &SQLitePartRepository{db: db}
}

func (r *SQLitePartRepository) Add(part *domain.Part) error {
	res, err := r.db.Exec(
		"INSERT INTO parts (name, group_order, group_id, media_id, path, monitored) VALUES (?, ?, ?, ?, ?, ?)",
		part.Name, part.GroupOrder, part.GroupId, part.MediaId, part.Path, boolToInt(part.Monitored),
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	part.Id = domain.ID(id)
	return nil
}

func (r *SQLitePartRepository) GetById(id domain.ID) (*domain.Part, error) {
	row := r.db.QueryRow(
		"SELECT id, name, group_order, group_id, media_id, path, monitored FROM parts WHERE id = ?", id,
	)
	p := &domain.Part{}
	if err := scanPart(row.Scan, p); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrPartNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *SQLitePartRepository) GetByMediaId(mediaId domain.ID) ([]domain.Part, error) {
	rows, err := r.db.Query(
		"SELECT id, name, group_order, group_id, media_id, path, monitored FROM parts WHERE media_id = ? ORDER BY group_order, id",
		mediaId,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []domain.Part{}
	for rows.Next() {
		var p domain.Part
		if err := scanPart(rows.Scan, &p); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r *SQLitePartRepository) GetWanted(page int, limit int) ([]domain.Part, error) {
	offset := (page - 1) * limit
	rows, err := r.db.Query(
		"SELECT id, name, group_order, group_id, media_id, path, monitored FROM parts WHERE monitored = 1 AND path IS NULL ORDER BY id LIMIT ? OFFSET ?",
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []domain.Part{}
	for rows.Next() {
		var p domain.Part
		if err := scanPart(rows.Scan, &p); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r *SQLitePartRepository) Remove(id domain.ID) error {
	res, err := r.db.Exec("DELETE FROM parts WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrPartNotFound
	}
	return nil
}

func (r *SQLitePartRepository) Update(part *domain.Part) error {
	res, err := r.db.Exec(
		"UPDATE parts SET name = ?, group_order = ?, group_id = ?, media_id = ?, path = ?, monitored = ? WHERE id = ?",
		part.Name, part.GroupOrder, part.GroupId, part.MediaId, part.Path, boolToInt(part.Monitored), part.Id,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrPartNotFound
	}
	return nil
}

type scanFn func(dest ...any) error

func scanPart(scan scanFn, p *domain.Part) error {
	var name, path sql.NullString
	var groupOrder sql.NullInt64
	var groupID sql.NullInt64
	var monitored int64

	if err := scan(&p.Id, &name, &groupOrder, &groupID, &p.MediaId, &path, &monitored); err != nil {
		return err
	}
	if name.Valid {
		p.Name = &name.String
	}
	if path.Valid {
		p.Path = &path.String
	}
	if groupOrder.Valid {
		v := int(groupOrder.Int64)
		p.GroupOrder = &v
	}
	if groupID.Valid {
		v := domain.ID(groupID.Int64)
		p.GroupId = &v
	}
	p.Monitored = monitored != 0
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
