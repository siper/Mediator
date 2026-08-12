package sqlite

import (
	"database/sql"
	"encoding/json"

	"stersh.ru/mediator/domain"
)

type SQLiteQueueRepository struct {
	db DBTX
}

func NewSQLiteQueueRepository(db DBTX) *SQLiteQueueRepository {
	return &SQLiteQueueRepository{db: db}
}

func (r *SQLiteQueueRepository) Add(q *domain.QueueItem) error {
	res, err := r.db.Exec(
		"INSERT INTO queue (media_id, part_ids, grabber_name, job_id, download_id, release_title, state, progress, added_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		q.MediaId, encodePartIds(q.PartIds), q.GrabberName, q.JobID, q.DownloadID, q.ReleaseTitle, q.State, q.Progress, formatTime(q.AddedAt),
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	q.Id = domain.ID(id)
	return nil
}

func (r *SQLiteQueueRepository) GetById(id domain.ID) (*domain.QueueItem, error) {
	row := r.db.QueryRow(
		"SELECT id, media_id, part_ids, grabber_name, job_id, download_id, release_title, state, progress, added_at FROM queue WHERE id = ?", id,
	)
	q, err := scanQueueItem(row.Scan)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrQueueNotFound
		}
		return nil, err
	}
	return q, nil
}

func (r *SQLiteQueueRepository) List(page int, limit int) ([]domain.QueueItem, error) {
	offset := (page - 1) * limit
	rows, err := r.db.Query(
		"SELECT id, media_id, part_ids, grabber_name, job_id, download_id, release_title, state, progress, added_at FROM queue ORDER BY id DESC LIMIT ? OFFSET ?",
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanQueueItems(rows)
}

func (r *SQLiteQueueRepository) ListActive() ([]domain.QueueItem, error) {
	rows, err := r.db.Query(
		"SELECT id, media_id, part_ids, grabber_name, job_id, download_id, release_title, state, progress, added_at FROM queue WHERE state IN (?, ?) ORDER BY id",
		string(domain.GrabQueued), string(domain.GrabRunning),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanQueueItems(rows)
}

func (r *SQLiteQueueRepository) Update(q *domain.QueueItem) error {
	res, err := r.db.Exec(
		"UPDATE queue SET media_id = ?, part_ids = ?, grabber_name = ?, job_id = ?, download_id = ?, release_title = ?, state = ?, progress = ?, added_at = ? WHERE id = ?",
		q.MediaId, encodePartIds(q.PartIds), q.GrabberName, q.JobID, q.DownloadID, q.ReleaseTitle, q.State, q.Progress, formatTime(q.AddedAt), q.Id,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrQueueNotFound
	}
	return nil
}

func (r *SQLiteQueueRepository) Remove(id domain.ID) error {
	res, err := r.db.Exec("DELETE FROM queue WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrQueueNotFound
	}
	return nil
}

func scanQueueItem(scan scanFn) (*domain.QueueItem, error) {
	var id, mediaId int64
	var partIDs, grabberName, jobID, releaseTitle, state, addedAt string
	var downloadID sql.NullString
	var progress float64
	if err := scan(&id, &mediaId, &partIDs, &grabberName, &jobID, &downloadID, &releaseTitle, &state, &progress, &addedAt); err != nil {
		return nil, err
	}
	t, err := parseTime(addedAt)
	if err != nil {
		return nil, err
	}
	return &domain.QueueItem{
		Id:           domain.ID(id),
		MediaId:      domain.ID(mediaId),
		PartIds:      decodePartIds(partIDs),
		GrabberName:  grabberName,
		JobID:        jobID,
		DownloadID:   downloadID.String,
		ReleaseTitle: releaseTitle,
		State:        domain.GrabState(state),
		Progress:     progress,
		AddedAt:      t,
	}, nil
}

func scanQueueItems(rows *sql.Rows) ([]domain.QueueItem, error) {
	result := []domain.QueueItem{}
	for rows.Next() {
		q, err := scanQueueItem(rows.Scan)
		if err != nil {
			return nil, err
		}
		result = append(result, *q)
	}
	return result, rows.Err()
}

func encodePartIds(ids []domain.ID) string {
	if len(ids) == 0 {
		return ""
	}
	b, err := json.Marshal(ids)
	if err != nil {
		return ""
	}
	return string(b)
}

func decodePartIds(s string) []domain.ID {
	if s == "" {
		return nil
	}
	var raw []uint64
	if err := json.Unmarshal([]byte(s), &raw); err != nil {
		return nil
	}
	out := make([]domain.ID, len(raw))
	for i, v := range raw {
		out[i] = domain.ID(v)
	}
	return out
}
