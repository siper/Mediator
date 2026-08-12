package sqlite

import (
	"database/sql"
	"encoding/json"

	"stersh.ru/mediator/domain"
)

type SQLiteQualityProfileRepository struct {
	db DBTX
}

func NewSQLiteQualityProfileRepository(db DBTX) *SQLiteQualityProfileRepository {
	return &SQLiteQualityProfileRepository{db: db}
}

func (r *SQLiteQualityProfileRepository) Add(p *domain.QualityProfile) error {
	allowed, err := marshalQualityNames(p.Allowed)
	if err != nil {
		return err
	}
	res, err := r.db.Exec(
		"INSERT INTO quality_profiles (name, type, allowed, cutoff) VALUES (?, ?, ?, ?)",
		p.Name, p.Type, allowed, p.Cutoff.Name,
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	p.Id = domain.ID(id)
	return nil
}

func (r *SQLiteQualityProfileRepository) GetById(id domain.ID) (*domain.QualityProfile, error) {
	row := r.db.QueryRow(
		"SELECT id, name, type, allowed, cutoff FROM quality_profiles WHERE id = ?", id,
	)
	p, err := scanProfile(row.Scan)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrMediaNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *SQLiteQualityProfileRepository) List() ([]domain.QualityProfile, error) {
	return r.list("SELECT id, name, type, allowed, cutoff FROM quality_profiles ORDER BY id")
}

func (r *SQLiteQualityProfileRepository) ListByType(t domain.MediaType) ([]domain.QualityProfile, error) {
	return r.list(
		"SELECT id, name, type, allowed, cutoff FROM quality_profiles WHERE type = ? ORDER BY id", t,
	)
}

func (r *SQLiteQualityProfileRepository) list(query string, args ...any) ([]domain.QualityProfile, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []domain.QualityProfile{}
	for rows.Next() {
		p, err := scanProfile(rows.Scan)
		if err != nil {
			return nil, err
		}
		result = append(result, *p)
	}
	return result, rows.Err()
}

func (r *SQLiteQualityProfileRepository) Update(p *domain.QualityProfile) error {
	allowed, err := marshalQualityNames(p.Allowed)
	if err != nil {
		return err
	}
	res, err := r.db.Exec(
		"UPDATE quality_profiles SET name = ?, type = ?, allowed = ?, cutoff = ? WHERE id = ?",
		p.Name, p.Type, allowed, p.Cutoff.Name, p.Id,
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

func (r *SQLiteQualityProfileRepository) Remove(id domain.ID) error {
	res, err := r.db.Exec("DELETE FROM quality_profiles WHERE id = ?", id)
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

func marshalQualityNames(qs []domain.Quality) (string, error) {
	names := make([]string, len(qs))
	for i, q := range qs {
		names[i] = q.Name
	}
	b, err := json.Marshal(names)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func scanProfile(scan scanFn) (*domain.QualityProfile, error) {
	var id int64
	var name string
	var t int
	var allowedJSON string
	var cutoffName string
	if err := scan(&id, &name, &t, &allowedJSON, &cutoffName); err != nil {
		return nil, err
	}

	mediaType := domain.MediaType(t)
	kind, _ := domain.QualityKindFor(mediaType)

	var names []string
	if allowedJSON != "" {
		if err := json.Unmarshal([]byte(allowedJSON), &names); err != nil {
			return nil, err
		}
	}
	allowed := make([]domain.Quality, 0, len(names))
	for _, n := range names {
		allowed = append(allowed, domain.Quality{Kind: kind, Name: n})
	}

	var cutoff domain.Quality
	if cutoffName != "" {
		cutoff = domain.Quality{Kind: kind, Name: cutoffName}
	}

	return &domain.QualityProfile{
		Id:      domain.ID(id),
		Name:    name,
		Type:    mediaType,
		Allowed: allowed,
		Cutoff:  cutoff,
	}, nil
}
