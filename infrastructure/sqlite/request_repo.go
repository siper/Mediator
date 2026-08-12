package sqlite

import (
	"database/sql"

	"stersh.ru/mediator/domain"
)

type SQLiteMediaRequestRepository struct {
	db DBTX
}

func NewSQLiteMediaRequestRepository(db DBTX) *SQLiteMediaRequestRepository {
	return &SQLiteMediaRequestRepository{db: db}
}

func (r *SQLiteMediaRequestRepository) Add(req *domain.MediaRequest) error {
	res, err := r.db.Exec(
		`INSERT INTO media_requests (user_id, provider, external_id, title, cover, type, library_id, quality_profile_id, folder, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.UserId, req.Provider, req.ExternalID, req.Title, req.Cover, req.Type,
		req.LibraryID, proxyIDValue(req.QualityProfileID), req.Folder, string(req.Status), req.CreatedAt,
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	req.Id = domain.ID(id)
	return nil
}

func (r *SQLiteMediaRequestRepository) GetByID(id domain.ID) (*domain.MediaRequest, error) {
	row := r.db.QueryRow(
		`SELECT id, user_id, provider, external_id, title, cover, type, library_id, quality_profile_id, folder, status, created_at, approved_at, approved_by, rejected_at, rejected_by, canceled_at, notes, media_id FROM media_requests WHERE id = ?`,
		id,
	)
	return scanRequest(row.Scan)
}

func (r *SQLiteMediaRequestRepository) ListByUser(userID domain.ID, page int, limit int) ([]domain.MediaRequest, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit
	rows, err := r.db.Query(
		`SELECT id, user_id, provider, external_id, title, cover, type, library_id, quality_profile_id, folder, status, created_at, approved_at, approved_by, rejected_at, rejected_by, canceled_at, notes, media_id
		 FROM media_requests WHERE user_id = ? ORDER BY id DESC LIMIT ? OFFSET ?`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRequests(rows)
}

func (r *SQLiteMediaRequestRepository) ListAll(page int, limit int) ([]domain.MediaRequest, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit
	rows, err := r.db.Query(
		`SELECT id, user_id, provider, external_id, title, cover, type, library_id, quality_profile_id, folder, status, created_at, approved_at, approved_by, rejected_at, rejected_by, canceled_at, notes, media_id
		 FROM media_requests ORDER BY id DESC LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRequests(rows)
}

func (r *SQLiteMediaRequestRepository) Update(req *domain.MediaRequest) error {
	res, err := r.db.Exec(
		`UPDATE media_requests SET user_id = ?, provider = ?, external_id = ?, title = ?, cover = ?, type = ?, library_id = ?, quality_profile_id = ?, folder = ?, status = ?, created_at = ?, approved_at = ?, approved_by = ?, rejected_at = ?, rejected_by = ?, canceled_at = ?, notes = ?, media_id = ? WHERE id = ?`,
		req.UserId, req.Provider, req.ExternalID, req.Title, req.Cover, req.Type,
		req.LibraryID, proxyIDValue(req.QualityProfileID), req.Folder, string(req.Status), req.CreatedAt,
		req.ApprovedAt, proxyIDValue(req.ApprovedBy), req.RejectedAt, proxyIDValue(req.RejectedBy),
		req.CanceledAt, req.Notes, proxyIDValue(req.MediaID), req.Id,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrRequestNotFound
	}
	return nil
}

func (r *SQLiteMediaRequestRepository) Remove(id domain.ID) error {
	res, err := r.db.Exec("DELETE FROM media_requests WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrRequestNotFound
	}
	return nil
}

func (r *SQLiteMediaRequestRepository) ExistsByProviderExternal(provider string, externalID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM media_requests WHERE provider = ? AND external_id = ? AND status IN ('pending', 'approved'))`,
		provider, externalID,
	).Scan(&exists)
	return exists, err
}

func scanRequest(scan scanFn) (*domain.MediaRequest, error) {
	req := &domain.MediaRequest{}
	if err := scanRequestFields(scan, req); err != nil {
		return nil, err
	}
	return req, nil
}

func scanRequests(rows *sql.Rows) ([]domain.MediaRequest, error) {
	result := []domain.MediaRequest{}
	for rows.Next() {
		var req domain.MediaRequest
		if err := scanRequestFields(rows.Scan, &req); err != nil {
			return nil, err
		}
		result = append(result, req)
	}
	return result, rows.Err()
}

func scanRequestFields(scan scanFn, req *domain.MediaRequest) error {
	var (
		approvedAt  sql.NullString
		rejectedAt  sql.NullString
		canceledAt  sql.NullString
		notes       sql.NullString
		folder      sql.NullString
		title       sql.NullString
		cover    sql.NullString
		qualityProf sql.NullInt64
		mediaID     sql.NullInt64
		approvedBy  sql.NullInt64
		rejectedBy  sql.NullInt64
	)
	err := scan(
		&req.Id, &req.UserId, &req.Provider, &req.ExternalID, &title, &cover,
		&req.Type, &req.LibraryID, &qualityProf, &folder, &req.Status, &req.CreatedAt,
		&approvedAt, &approvedBy, &rejectedAt, &rejectedBy, &canceledAt,
		&notes, &mediaID,
	)
	if err == sql.ErrNoRows {
		return domain.ErrRequestNotFound
	}
	if err != nil {
		return err
	}
	if title.Valid {
		req.Title = title.String
	}
	if cover.Valid {
		req.Cover = cover.String
	}
	if folder.Valid {
		req.Folder = &folder.String
	}
	if approvedAt.Valid {
		req.ApprovedAt = &approvedAt.String
	}
	if rejectedAt.Valid {
		req.RejectedAt = &rejectedAt.String
	}
	if canceledAt.Valid {
		req.CanceledAt = &canceledAt.String
	}
	if notes.Valid {
		req.Notes = &notes.String
	}
	if qualityProf.Valid {
		v := domain.ID(qualityProf.Int64)
		req.QualityProfileID = &v
	}
	if mediaID.Valid {
		v := domain.ID(mediaID.Int64)
		req.MediaID = &v
	}
	if approvedBy.Valid {
		v := domain.ID(approvedBy.Int64)
		req.ApprovedBy = &v
	}
	if rejectedBy.Valid {
		v := domain.ID(rejectedBy.Int64)
		req.RejectedBy = &v
	}
	return nil
}
