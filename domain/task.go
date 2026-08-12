package domain

import "time"

type TaskStatus string

const (
	TaskIdle    TaskStatus = "idle"
	TaskRunning TaskStatus = "running"
	TaskOK      TaskStatus = "ok"
	TaskError   TaskStatus = "error"
)

const (
	TaskCleanupStuck   = "cleanup-stuck"
	TaskSettingPenalty = "penalty"
)

type Task struct {
	Name       string
	Interval   string
	Enabled    bool
	LastRun    *time.Time
	NextRun    *time.Time
	LastStatus TaskStatus
	LastError  string
	Settings   map[string]string
	Data       string
}

type TaskRepository interface {
	List() ([]Task, error)
	Get(name string) (*Task, error)
	Create(task Task) error
	Update(name string, interval string, enabled bool, settings map[string]string) error
	UpdateData(name string, data string) error
	UpdateRun(name string, status TaskStatus, errMsg string, lastRun time.Time, nextRun time.Time) error
}
