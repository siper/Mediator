package sqlite

import (
	"database/sql"

	"stersh.ru/mediator/domain"
)

type SQLiteMediaRepository struct {
	db DBTX
}

func NewSQLiteMediaRepository(db DBTX) *SQLiteMediaRepository {
	return &SQLiteMediaRepository{db: db}
}

const mediaSelectCols = "id, name, original_name, folder, cover, type, library_id, provider_id, external_id, status, last_modified, quality_profile_id"

func scanMedia(row interface {
	Scan(dest ...any) error
}) (*domain.Media, error) {
	m := &domain.Media{}
	var (
		cover     sql.NullString
		folder    sql.NullString
		libID     sql.NullInt64
		profileID sql.NullInt64
	)
	err := row.Scan(&m.Id, &m.Name, &m.OriginalName, &folder, &cover, &m.Type, &libID, &m.ProviderID, &m.ExternalID, &m.Status, &m.LastModified, &profileID)
	if err == sql.ErrNoRows {
		return nil, domain.ErrMediaNotFound
	}
	if err != nil {
		return nil, err
	}
	if folder.Valid {
		m.Folder = &folder.String
	}
	if cover.Valid {
		m.Cover = &cover.String
	}
	if libID.Valid {
		v := domain.ID(libID.Int64)
		m.LibraryID = &v
	}
	if profileID.Valid {
		v := domain.ID(profileID.Int64)
		m.QualityProfileID = &v
	}
	return m, nil
}

func (r *SQLiteMediaRepository) GetById(id domain.ID) (*domain.Media, error) {
	row := r.db.QueryRow(
		"SELECT "+mediaSelectCols+" FROM media WHERE id = ?", id,
	)
	return scanMedia(row)
}

func (r *SQLiteMediaRepository) Create(name string, originalName string, folder *string, cover *string, mediaType domain.MediaType, libraryID *domain.ID, providerID string, externalID string, profileID *domain.ID) (*domain.Media, error) {
	var libArg any
	if libraryID != nil {
		libArg = int64(*libraryID)
	}
	var profileArg any
	if profileID != nil {
		profileArg = int64(*profileID)
	}
	res, err := r.db.Exec(
		"INSERT INTO media (name, original_name, folder, cover, type, library_id, provider_id, external_id, quality_profile_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		name, originalName, folder, cover, mediaType, libArg, providerID, externalID, profileArg,
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &domain.Media{
		Id:               domain.ID(id),
		Name:             name,
		OriginalName:     originalName,
		Folder:           folder,
		Cover:            cover,
		Type:             mediaType,
		LibraryID:        libraryID,
		ProviderID:       providerID,
		ExternalID:       externalID,
		QualityProfileID: profileID,
	}, nil
}

func (r *SQLiteMediaRepository) Update(id domain.ID, name string, originalName string, cover *string) error {
	res, err := r.db.Exec(
		"UPDATE media SET name = ?, original_name = ?, cover = ? WHERE id = ?",
		name, originalName, cover, id,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrMediaNotFound
	}
	return nil
}

func (r *SQLiteMediaRepository) UpdateProfile(id domain.ID, profileID *domain.ID) error {
	var arg any
	if profileID != nil {
		arg = int64(*profileID)
	}
	res, err := r.db.Exec(
		"UPDATE media SET quality_profile_id = ? WHERE id = ?",
		arg, id,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrMediaNotFound
	}
	return nil
}

func (r *SQLiteMediaRepository) UpdateProviderMeta(id domain.ID, status domain.MediaStatus, lastModified string) error {
	res, err := r.db.Exec(
		"UPDATE media SET status = ?, last_modified = ? WHERE id = ?",
		string(status), lastModified, id,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrMediaNotFound
	}
	return nil
}

func (r *SQLiteMediaRepository) Remove(id domain.ID) error {
	res, err := r.db.Exec("DELETE FROM media WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrMediaNotFound
	}
	return nil
}

func (r *SQLiteMediaRepository) GetByProviderExternal(provider string, externalID string) (*domain.Media, error) {
	row := r.db.QueryRow(
		"SELECT "+mediaSelectCols+" FROM media WHERE provider_id = ? AND external_id = ?",
		provider, externalID,
	)
	return scanMedia(row)
}

func (r *SQLiteMediaRepository) GetPaged(page int, limit int, mediaType *domain.MediaType) ([]domain.Media, error) {
	offset := (page - 1) * limit

	var rows *sql.Rows
	var err error

	if mediaType != nil {
		rows, err = r.db.Query(
			"SELECT "+mediaSelectCols+" FROM media WHERE type = ? ORDER BY id LIMIT ? OFFSET ?",
			*mediaType, limit, offset,
		)
	} else {
		rows, err = r.db.Query(
			"SELECT "+mediaSelectCols+" FROM media ORDER BY id LIMIT ? OFFSET ?",
			limit, offset,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []domain.Media{}
	for rows.Next() {
		m, err := scanMedia(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *m)
	}
	return result, rows.Err()
}
