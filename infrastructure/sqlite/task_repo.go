package sqlite

import (
	"database/sql"
	"time"

	"stersh.ru/mediator/domain"
)

type SQLiteTaskRepository struct {
	db DBTX
}

func NewSQLiteTaskRepository(db DBTX) *SQLiteTaskRepository {
	return &SQLiteTaskRepository{db: db}
}

func (r *SQLiteTaskRepository) List() ([]domain.Task, error) {
	rows, err := r.db.Query(
		"SELECT name, interval, enabled, last_run, next_run, last_status, last_error, settings, data FROM tasks ORDER BY name",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []domain.Task{}
	for rows.Next() {
		t, err := scanTask(rows.Scan)
		if err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

func (r *SQLiteTaskRepository) Get(name string) (*domain.Task, error) {
	row := r.db.QueryRow(
		"SELECT name, interval, enabled, last_run, next_run, last_status, last_error, settings, data FROM tasks WHERE name = ?", name,
	)
	t, err := scanTask(row.Scan)
	if err == sql.ErrNoRows {
		return nil, domain.ErrTaskNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *SQLiteTaskRepository) Create(task domain.Task) error {
	data := task.Data
	if data == "" {
		data = "{}"
	}
	_, err := r.db.Exec(
		"INSERT INTO tasks (name, interval, enabled, last_status, last_error, settings, data) VALUES (?, ?, ?, ?, '', ?, ?)",
		task.Name, task.Interval, boolToInt(task.Enabled), string(domain.TaskIdle), encodeSettings(task.Settings), data,
	)
	return err
}

func (r *SQLiteTaskRepository) Update(name string, interval string, enabled bool, settings map[string]string) error {
	next := time.Now().UTC().Add(parseInterval(interval))
	res, err := r.db.Exec(
		"UPDATE tasks SET interval = ?, enabled = ?, next_run = ?, settings = ? WHERE name = ?",
		interval, boolToInt(enabled), next.UTC().Format(time.RFC3339), encodeSettings(settings), name,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrTaskNotFound
	}
	return nil
}

func (r *SQLiteTaskRepository) UpdateData(name string, data string) error {
	if data == "" {
		data = "{}"
	}
	res, err := r.db.Exec("UPDATE tasks SET data = ? WHERE name = ?", data, name)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrTaskNotFound
	}
	return nil
}

func (r *SQLiteTaskRepository) UpdateRun(name string, status domain.TaskStatus, errMsg string, lastRun time.Time, nextRun time.Time) error {
	res, err := r.db.Exec(
		"UPDATE tasks SET last_status = ?, last_error = ?, last_run = ?, next_run = ? WHERE name = ?",
		string(status), errMsg, lastRun.UTC().Format(time.RFC3339), nextRun.UTC().Format(time.RFC3339), name,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrTaskNotFound
	}
	return nil
}

type taskScanFn func(dest ...any) error

func scanTask(scan taskScanFn) (domain.Task, error) {
	var t domain.Task
	var enabled int64
	var lastRun, nextRun sql.NullString
	var settingsJSON, data string
	if err := scan(&t.Name, &t.Interval, &enabled, &lastRun, &nextRun, &t.LastStatus, &t.LastError, &settingsJSON, &data); err != nil {
		return t, err
	}
	t.Enabled = enabled != 0
	t.Settings = decodeSettings(settingsJSON)
	t.Data = data
	if t.Data == "" {
		t.Data = "{}"
	}
	if lastRun.Valid {
		if parsed, err := time.Parse(time.RFC3339, lastRun.String); err == nil {
			t.LastRun = &parsed
		}
	}
	if nextRun.Valid {
		if parsed, err := time.Parse(time.RFC3339, nextRun.String); err == nil {
			t.NextRun = &parsed
		}
	}
	return t, nil
}

func parseInterval(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return 24 * time.Hour
	}
	return d
}
