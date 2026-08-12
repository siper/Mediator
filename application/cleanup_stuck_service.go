package application

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"stersh.ru/mediator/domain"
)

const defaultStuckPenalty = 3

type stallEntry struct {
	Progress float64 `json:"progress"`
	State    string  `json:"state"`
	Count    int     `json:"count"`
}

type CleanupStuckService struct {
	mu          sync.RWMutex
	grabbers    []domain.Grabber
	taskRepo    domain.TaskRepository
	queueRepo   domain.QueueRepository
	historyRepo domain.HistoryRepository
}

func NewCleanupStuckService(
	grabbers []domain.Grabber,
	taskRepo domain.TaskRepository,
	queueRepo domain.QueueRepository,
	historyRepo domain.HistoryRepository,
) *CleanupStuckService {
	return &CleanupStuckService{
		grabbers:    grabbers,
		taskRepo:    taskRepo,
		queueRepo:   queueRepo,
		historyRepo: historyRepo,
	}
}

func (s *CleanupStuckService) SetGrabbers(grabbers []domain.Grabber) {
	s.mu.Lock()
	s.grabbers = grabbers
	s.mu.Unlock()
}

func (s *CleanupStuckService) Run(ctx context.Context) error {
	task, err := s.taskRepo.Get(domain.TaskCleanupStuck)
	if err != nil {
		return err
	}
	penalty := parseStuckPenalty(task.Settings)
	stalls := decodeStallMap(task.Data)

	active, err := s.queueRepo.ListActive()
	if err != nil {
		return err
	}

	activeByID := make(map[string]domain.QueueItem, len(active))
	for _, item := range active {
		activeByID[strconv.FormatUint(uint64(item.Id), 10)] = item
	}

	for id := range stalls {
		if _, ok := activeByID[id]; !ok {
			delete(stalls, id)
		}
	}

	for id, item := range activeByID {
		entry, ok := stalls[id]
		if !ok {
			stalls[id] = stallEntry{Progress: item.Progress, State: string(item.State), Count: 0}
			continue
		}
		if entry.Progress == item.Progress && entry.State == string(item.State) {
			entry.Count++
			if entry.Count >= penalty {
				if err := s.removeStuck(ctx, item); err != nil {
					slog.Warn("cleanup-stuck: remove failed", "queue_id", item.Id, "err", err)
					stalls[id] = entry
					continue
				}
				delete(stalls, id)
				continue
			}
			stalls[id] = entry
			continue
		}
		stalls[id] = stallEntry{Progress: item.Progress, State: string(item.State), Count: 0}
	}

	return s.taskRepo.UpdateData(domain.TaskCleanupStuck, encodeStallMap(stalls))
}

func (s *CleanupStuckService) removeStuck(ctx context.Context, item domain.QueueItem) error {
	if g := s.findGrabber(item.GrabberName); g != nil {
		lookupID := item.DownloadID
		if lookupID == "" {
			lookupID = item.JobID
		}
		if err := g.Cancel(ctx, domain.GrabHandle{
			GrabberName: item.GrabberName,
			JobID:       lookupID,
			Name:        item.ReleaseTitle,
		}); err != nil {
			slog.Warn("cleanup-stuck: cancel failed", "queue_id", item.Id, "err", err)
		}
	} else {
		slog.Warn("cleanup-stuck: grabber not found", "queue_id", item.Id, "grabber", item.GrabberName)
	}

	if err := s.historyRepo.Add(&domain.History{
		MediaId:      item.MediaId,
		EventType:    domain.HistoryFailed,
		ReleaseTitle: item.ReleaseTitle,
		Data:         "stuck",
		CreatedAt:    time.Now().UTC(),
	}); err != nil {
		return err
	}
	return s.queueRepo.Remove(item.Id)
}

func (s *CleanupStuckService) findGrabber(name string) domain.Grabber {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, g := range s.grabbers {
		if g.Name() == name {
			return g
		}
	}
	return nil
}

func parseStuckPenalty(settings map[string]string) int {
	if settings == nil {
		return defaultStuckPenalty
	}
	raw, ok := settings[domain.TaskSettingPenalty]
	if !ok || raw == "" {
		return defaultStuckPenalty
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return defaultStuckPenalty
	}
	return n
}

func decodeStallMap(data string) map[string]stallEntry {
	out := map[string]stallEntry{}
	if data == "" || data == "{}" {
		return out
	}
	_ = json.Unmarshal([]byte(data), &out)
	if out == nil {
		return map[string]stallEntry{}
	}
	return out
}

func encodeStallMap(m map[string]stallEntry) string {
	if len(m) == 0 {
		return "{}"
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(b)
}
