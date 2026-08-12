package sqlite

import (
	"stersh.ru/mediator/domain"
)

type SQLitePartGroupRepository struct {
	db DBTX
}

func NewSQLitePartGroupRepository(db DBTX) *SQLitePartGroupRepository {
	return &SQLitePartGroupRepository{db: db}
}

func (r *SQLitePartGroupRepository) Add(group *domain.PartGroup) error {
	res, err := r.db.Exec(
		"INSERT INTO part_groups (name, order_num, media_id) VALUES (?, ?, ?)",
		group.Name, group.Order, group.MediaId,
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	group.Id = domain.ID(id)
	return nil
}

func (r *SQLitePartGroupRepository) GetByMediaID(mediaID domain.ID) ([]domain.PartGroup, error) {
	rows, err := r.db.Query(
		"SELECT id, name, order_num, media_id FROM part_groups WHERE media_id = ? ORDER BY order_num, id",
		mediaID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []domain.PartGroup{}
	for rows.Next() {
		var g domain.PartGroup
		if err := rows.Scan(&g.Id, &g.Name, &g.Order, &g.MediaId); err != nil {
			return nil, err
		}
		result = append(result, g)
	}
	return result, rows.Err()
}

func (r *SQLitePartGroupRepository) Remove(id domain.ID) error {
	res, err := r.db.Exec("DELETE FROM part_groups WHERE id = ?", id)
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
