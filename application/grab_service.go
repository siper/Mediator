package application

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"stersh.ru/mediator/domain"
)

type GrabService struct {
	mu        sync.RWMutex
	grabbers  []domain.Grabber
	queueRepo domain.QueueRepository
	uow       domain.UnitOfWork
}

func NewGrabService(grabbers []domain.Grabber, queueRepo domain.QueueRepository, uow domain.UnitOfWork) *GrabService {
	return &GrabService{grabbers: grabbers, queueRepo: queueRepo, uow: uow}
}

func (s *GrabService) SetGrabbers(grabbers []domain.Grabber) {
	s.mu.Lock()
	s.grabbers = grabbers
	s.mu.Unlock()
}

func (s *GrabService) Grab(ctx context.Context, t domain.GrabTarget) (*domain.QueueItem, error) {
	slog.Debug("grab service: grab called", "media_id", t.MediaID, "provider_name", t.ProviderName, "url", t.URL)
	grabber, err := s.findGrabber(t)
	if err != nil {
		slog.Warn("grab service: no grabber found", "media_id", t.MediaID, "provider_name", t.ProviderName, "err", err)
		return nil, err
	}
	slog.Debug("grab service: grabber found", "media_id", t.MediaID, "grabber", grabber.Name())
	handle, err := grabber.Submit(ctx, t)
	if err != nil {
		slog.Warn("grab service: submit failed", "media_id", t.MediaID, "grabber", grabber.Name(), "err", err)
		return nil, fmt.Errorf("grab submit failed: %w", err)
	}
	slog.Debug("grab service: submit ok", "media_id", t.MediaID, "job_id", handle.JobID)

	now := time.Now().UTC()
	item := &domain.QueueItem{
		MediaId:      t.MediaID,
		PartIds:      t.PartIds,
		GrabberName:  handle.GrabberName,
		JobID:        handle.JobID,
		ReleaseTitle: grabTitle(t),
		State:        domain.GrabQueued,
		AddedAt:      now,
	}
	var grabbedPartId *domain.ID
	if len(t.PartIds) == 1 {
		pid := t.PartIds[0]
		grabbedPartId = &pid
	}

	err = s.uow.Run(ctx, func(repos *domain.Repos) error {
		if err := repos.Queue.Add(item); err != nil {
			slog.Warn("grab service: queue insert failed", "media_id", t.MediaID, "job_id", handle.JobID, "err", err)
			return fmt.Errorf("queue insert failed: %w", err)
		}
		if err := repos.History.Add(&domain.History{
			MediaId:      t.MediaID,
			PartId:       grabbedPartId,
			EventType:    domain.HistoryGrabbed,
			ReleaseTitle: item.ReleaseTitle,
			CreatedAt:    now,
		}); err != nil {
			slog.Warn("grab service: history insert failed", "media_id", t.MediaID, "job_id", handle.JobID, "err", err)
			return fmt.Errorf("history insert failed: %w", err)
		}
		return nil
	})
	if err != nil {
		if cancelErr := grabber.Cancel(ctx, handle); cancelErr != nil {
			slog.Warn("grab service: cancel after persist failure",
				"media_id", t.MediaID, "job_id", handle.JobID, "err", err, "cancel_err", cancelErr)
		}
		return nil, err
	}

	slog.Info("grab service: grab queued", "media_id", t.MediaID, "job_id", handle.JobID, "grabber", handle.GrabberName)
	return item, nil
}

func (s *GrabService) Remove(ctx context.Context, id domain.ID) error {
	item, err := s.queueRepo.GetById(id)
	if err != nil {
		return err
	}

	if !item.State.Active() {
		return s.queueRepo.Remove(item.Id)
	}

	if g := s.grabberByName(item.GrabberName); g != nil {
		lookupID := item.DownloadID
		if lookupID == "" {
			lookupID = item.JobID
		}
		if err := g.Cancel(ctx, domain.GrabHandle{
			GrabberName: item.GrabberName,
			JobID:       lookupID,
			Name:        item.ReleaseTitle,
		}); err != nil {
			slog.Warn("grab service: cancel failed", "queue_id", item.Id, "err", err)
		}
	} else {
		slog.Warn("grab service: grabber not found", "queue_id", item.Id, "grabber", item.GrabberName)
	}

	return s.uow.Run(ctx, func(repos *domain.Repos) error {
		if err := repos.History.Add(&domain.History{
			MediaId:      item.MediaId,
			EventType:    domain.HistoryFailed,
			ReleaseTitle: item.ReleaseTitle,
			Data:         "canceled",
			CreatedAt:    time.Now().UTC(),
		}); err != nil {
			return err
		}
		return repos.Queue.Remove(item.Id)
	})
}

func (s *GrabService) grabberByName(name string) domain.Grabber {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, g := range s.grabbers {
		if g.Name() == name {
			return g
		}
	}
	return nil
}

func (s *GrabService) findGrabber(t domain.GrabTarget) (domain.Grabber, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	slog.Debug("grab service: findGrabber called", "provider_name", t.ProviderName, "url", t.URL, "grabber_count", len(s.grabbers))
	for _, g := range s.grabbers {
		if g.Supports(t) {
			slog.Debug("grab service: grabber matched", "provider_name", t.ProviderName, "grabber", g.Name())
			return g, nil
		}
	}
	slog.Warn("grab service: no grabber matched", "provider_name", t.ProviderName, "url", t.URL)
	return nil, domain.ErrNoGrabber
}

func (s *GrabService) HasActiveGrab(mediaID domain.ID, partID domain.ID) bool {
	active, err := s.queueRepo.ListActive()
	if err != nil {
		return false
	}
	for _, item := range active {
		if item.MediaId != mediaID {
			continue
		}
		if len(item.PartIds) == 0 {
			return true
		}
		for _, pid := range item.PartIds {
			if pid == partID {
				return true
			}
		}
	}
	return false
}

func (s *GrabService) HasActiveReleaseTitle(mediaID domain.ID, title string) bool {
	if title == "" {
		return false
	}
	active, err := s.queueRepo.ListActive()
	if err != nil {
		return false
	}
	for _, item := range active {
		if item.MediaId == mediaID && item.ReleaseTitle == title {
			return true
		}
	}
	return false
}

func (s *GrabService) SupportsProvider(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, g := range s.grabbers {
		if g.SupportsProvider(name) {
			return true
		}
	}
	return false
}

func grabTitle(t domain.GrabTarget) string {
	if t.Release != nil && t.Release.Title != "" {
		return t.Release.Title
	}
	return t.URL
}
