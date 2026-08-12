package sqlite

import (
	"database/sql"

	"stersh.ru/mediator/domain"
)

type SQLiteHistoryRepository struct {
	db DBTX
}

func NewSQLiteHistoryRepository(db DBTX) *SQLiteHistoryRepository {
	return &SQLiteHistoryRepository{db: db}
}

func (r *SQLiteHistoryRepository) Add(h *domain.History) error {
	res, err := r.db.Exec(
		"INSERT INTO history (media_id, part_id, event_type, release_title, data, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		h.MediaId, h.PartId, h.EventType, h.ReleaseTitle, h.Data, formatTime(h.CreatedAt),
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	h.Id = domain.ID(id)
	return nil
}

func (r *SQLiteHistoryRepository) GetByMediaId(mediaId domain.ID) ([]domain.History, error) {
	rows, err := r.db.Query(
		"SELECT id, media_id, part_id, event_type, release_title, data, created_at FROM history WHERE media_id = ? ORDER BY id DESC",
		mediaId,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHistory(rows)
}

func (r *SQLiteHistoryRepository) List(page int, limit int) ([]domain.History, error) {
	offset := (page - 1) * limit
	rows, err := r.db.Query(
		"SELECT id, media_id, part_id, event_type, release_title, data, created_at FROM history ORDER BY id DESC LIMIT ? OFFSET ?",
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHistory(rows)
}

func scanHistory(rows *sql.Rows) ([]domain.History, error) {
	result := []domain.History{}
	for rows.Next() {
		var id, mediaId int64
		var partID sql.NullInt64
		var eventType, releaseTitle, data, createdAt string
		if err := rows.Scan(&id, &mediaId, &partID, &eventType, &releaseTitle, &data, &createdAt); err != nil {
			return nil, err
		}
		t, err := parseTime(createdAt)
		if err != nil {
			return nil, err
		}
		h := domain.History{
			Id:           domain.ID(id),
			MediaId:      domain.ID(mediaId),
			EventType:    domain.HistoryEventType(eventType),
			ReleaseTitle: releaseTitle,
			Data:         data,
			CreatedAt:    t,
		}
		if partID.Valid {
			v := domain.ID(partID.Int64)
			h.PartId = &v
		}
		result = append(result, h)
	}
	return result, rows.Err()
}
