package sqlite

import (
	"database/sql"

	"stersh.ru/mediator/domain"
)

type SQLiteProxyRepository struct {
	db DBTX
}

func NewSQLiteProxyRepository(db DBTX) *SQLiteProxyRepository {
	return &SQLiteProxyRepository{db: db}
}

func (r *SQLiteProxyRepository) Add(p *domain.Proxy) error {
	res, err := r.db.Exec(
		"INSERT INTO proxies (name, type, endpoint, enabled) VALUES (?, ?, ?, ?)",
		p.Name, p.Type, p.Endpoint, boolToInt(p.Enabled),
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

func (r *SQLiteProxyRepository) GetByID(id domain.ID) (*domain.Proxy, error) {
	row := r.db.QueryRow(
		"SELECT id, name, type, endpoint, enabled FROM proxies WHERE id = ?", id,
	)
	p, err := scanProxy(row.Scan)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrProxyNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *SQLiteProxyRepository) List() ([]domain.Proxy, error) {
	rows, err := r.db.Query("SELECT id, name, type, endpoint, enabled FROM proxies ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []domain.Proxy{}
	for rows.Next() {
		p, err := scanProxy(rows.Scan)
		if err != nil {
			return nil, err
		}
		result = append(result, *p)
	}
	return result, rows.Err()
}

func (r *SQLiteProxyRepository) ListEnabled() ([]domain.Proxy, error) {
	rows, err := r.db.Query("SELECT id, name, type, endpoint, enabled FROM proxies WHERE enabled = 1 ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []domain.Proxy{}
	for rows.Next() {
		p, err := scanProxy(rows.Scan)
		if err != nil {
			return nil, err
		}
		result = append(result, *p)
	}
	return result, rows.Err()
}

func (r *SQLiteProxyRepository) Update(p *domain.Proxy) error {
	res, err := r.db.Exec(
		"UPDATE proxies SET name = ?, type = ?, endpoint = ?, enabled = ? WHERE id = ?",
		p.Name, p.Type, p.Endpoint, boolToInt(p.Enabled), p.Id,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrProxyNotFound
	}
	return nil
}

func (r *SQLiteProxyRepository) Remove(id domain.ID) error {
	res, err := r.db.Exec("DELETE FROM proxies WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrProxyNotFound
	}
	return nil
}

func scanProxy(scan scanFn) (*domain.Proxy, error) {
	var id int64
	var name, typ, endpoint string
	var enabled int
	if err := scan(&id, &name, &typ, &endpoint, &enabled); err != nil {
		return nil, err
	}
	return &domain.Proxy{
		Id:       domain.ID(id),
		Name:     name,
		Type:     domain.ProxyType(typ),
		Endpoint: endpoint,
		Enabled:  enabled != 0,
	}, nil
}
