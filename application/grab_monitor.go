package application

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"stersh.ru/mediator/domain"
)

type GrabMonitor struct {
	mu          sync.RWMutex
	grabbers    []domain.Grabber
	queueRepo   domain.QueueRepository
	historyRepo domain.HistoryRepository
	OnCompleted func(ctx context.Context, item domain.QueueItem, status domain.GrabStatus) error
}

func NewGrabMonitor(grabbers []domain.Grabber, queueRepo domain.QueueRepository, historyRepo domain.HistoryRepository) *GrabMonitor {
	return &GrabMonitor{grabbers: grabbers, queueRepo: queueRepo, historyRepo: historyRepo}
}

func (m *GrabMonitor) SetGrabbers(grabbers []domain.Grabber) {
	m.mu.Lock()
	m.grabbers = grabbers
	m.mu.Unlock()
}

func (m *GrabMonitor) Tick(ctx context.Context) error {
	active, err := m.queueRepo.ListActive()
	if err != nil {
		return err
	}
	slog.Debug("grab monitor: tick", "active_count", len(active))
	for _, item := range active {
		if err := m.tickOne(ctx, item); err != nil {
			slog.Warn("grab monitor tick failed", "job", item.JobID, "err", err)
		}
	}
	return nil
}

func (m *GrabMonitor) tickOne(ctx context.Context, item domain.QueueItem) error {
	slog.Debug("grab monitor: tickOne", "job_id", item.JobID, "download_id", item.DownloadID, "grabber", item.GrabberName, "current_state", item.State)
	g := m.findGrabber(item.GrabberName)
	if g == nil {
		slog.Warn("grab monitor: grabber not found", "job_id", item.JobID, "grabber", item.GrabberName)
		return domain.ErrNoGrabber
	}
	lookupID := item.DownloadID
	if lookupID == "" {
		lookupID = item.JobID
	}
	status, err := g.Status(ctx, domain.GrabHandle{GrabberName: item.GrabberName, JobID: lookupID, Name: item.ReleaseTitle})
	if err != nil {
		slog.Warn("grab monitor: status check failed", "job_id", item.JobID, "download_id", item.DownloadID, "err", err)
		return err
	}
	if status.ResolvedID != "" && item.DownloadID == "" {
		slog.Debug("grab monitor: resolved download id", "job_id", item.JobID, "download_id", status.ResolvedID, "release", item.ReleaseTitle)
		item.DownloadID = status.ResolvedID
	}
	slog.Debug("grab monitor: status", "job_id", item.JobID, "state", status.State, "progress", status.Progress, "error", status.Error)

	prevState := item.State
	prevProgress := item.Progress
	prevDownloadID := item.DownloadID
	item.Progress = status.Progress

	switch status.State {
	case domain.GrabCompleted:
		slog.Info("grab monitor: job completed", "job_id", item.JobID, "media_id", item.MediaId, "output_files", status.OutputFiles)
		item.State = domain.GrabCompleted
		if m.OnCompleted != nil {
			if importErr := m.OnCompleted(ctx, item, status); importErr != nil {
				if isFatalImportError(importErr) {
					slog.Warn("grab monitor: import failed permanently", "job_id", item.JobID, "media_id", item.MediaId, "err", importErr)
					item.State = domain.GrabFailed
					if err := m.queueRepo.Update(&item); err != nil {
						return err
					}
					return m.historyRepo.Add(&domain.History{
						MediaId:      item.MediaId,
						EventType:    domain.HistoryFailed,
						ReleaseTitle: item.ReleaseTitle,
						Data:         importErr.Error(),
						CreatedAt:    time.Now().UTC(),
					})
				}
				slog.Warn("grab monitor: import failed, will retry", "job_id", item.JobID, "media_id", item.MediaId, "err", importErr)
				item.State = domain.GrabRunning
			}
		}
		return m.queueRepo.Update(&item)
	case domain.GrabFailed:
		slog.Warn("grab monitor: job failed", "job_id", item.JobID, "media_id", item.MediaId, "error", status.Error)
		item.State = domain.GrabFailed
		if err := m.queueRepo.Update(&item); err != nil {
			return err
		}
		return m.historyRepo.Add(&domain.History{
			MediaId:      item.MediaId,
			EventType:    domain.HistoryFailed,
			ReleaseTitle: item.ReleaseTitle,
			Data:         status.Error,
			CreatedAt:    time.Now().UTC(),
		})
	default:
		if status.State == domain.GrabRunning && prevState != domain.GrabRunning {
			slog.Debug("grab monitor: job started running", "job_id", item.JobID, "media_id", item.MediaId)
			item.State = domain.GrabRunning
			return m.queueRepo.Update(&item)
		}
	}

	if prevState != item.State || prevProgress != item.Progress || prevDownloadID != item.DownloadID {
		if err := m.queueRepo.Update(&item); err != nil {
			slog.Warn("grab monitor: progress update failed", "job_id", item.JobID, "err", err)
			return err
		}
	}
	return nil
}

func isFatalImportError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, domain.ErrPartNotFound) ||
		errors.Is(err, domain.ErrLibraryRequired) ||
		errors.Is(err, domain.ErrLibraryNotFound) ||
		errors.Is(err, domain.ErrMediaNotFound) {
		return true
	}
	return strings.Contains(err.Error(), "no output files")
}

func (m *GrabMonitor) findGrabber(name string) domain.Grabber {
	m.mu.RLock()
	defer m.mu.RUnlock()
	slog.Debug("grab monitor: findGrabber called", "name", name, "grabber_count", len(m.grabbers))
	for _, g := range m.grabbers {
		if g.Name() == name {
			slog.Debug("grab monitor: grabber found", "name", name, "grabber", g.Name())
			return g
		}
	}
	slog.Warn("grab monitor: grabber not found", "name", name, "registered_grabbers", func() []string {
		names := make([]string, 0, len(m.grabbers))
		for _, g := range m.grabbers {
			names = append(names, g.Name())
		}
		return names
	}())
	return nil
}
