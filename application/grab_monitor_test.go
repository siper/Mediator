package application

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

func TestIsFatalImportError(t *testing.T) {
	assert.True(t, isFatalImportError(domain.ErrPartNotFound))
	assert.True(t, isFatalImportError(domain.ErrLibraryRequired))
	assert.True(t, isFatalImportError(fmt.Errorf("import: no output files")))
	assert.False(t, isFatalImportError(fmt.Errorf("import move failed: permission denied")))
}

func TestGrabMonitor_FatalImportMarksFailed(t *testing.T) {
	queue := new(mockQueueRepo)
	history := new(mockHistoryRepo)
	grabber := &fakeGrabber{
		name:    "torrent",
		support: true,
		status: map[string]domain.GrabStatus{
			"abc": {State: domain.GrabCompleted, Progress: 1, OutputFiles: []string{"/tmp/a.mkv"}},
		},
	}

	item := domain.QueueItem{
		Id: 1, MediaId: 6, JobID: "abc", GrabberName: "torrent",
		ReleaseTitle: "The Good Doctor S03", State: domain.GrabRunning, Progress: 1,
	}
	queue.On("ListActive").Return([]domain.QueueItem{item}, nil)
	queue.On("Update", mock.MatchedBy(func(q *domain.QueueItem) bool {
		return q.State == domain.GrabFailed
	})).Return(nil)
	history.On("Add", mock.MatchedBy(func(h *domain.History) bool {
		return h.EventType == domain.HistoryFailed
	})).Return(nil)

	mon := NewGrabMonitor([]domain.Grabber{grabber}, queue, history)
	mon.OnCompleted = func(ctx context.Context, item domain.QueueItem, status domain.GrabStatus) error {
		return domain.ErrPartNotFound
	}
	require.NoError(t, mon.Tick(context.Background()))
	queue.AssertExpectations(t)
	history.AssertExpectations(t)
}

func TestCoveredPartIDs(t *testing.T) {
	ep := func(n int) *int { return &n }
	parts := []domain.Part{
		{Id: 1, GroupOrder: ep(1)},
		{Id: 2, GroupOrder: ep(2)},
		{Id: 3, GroupOrder: ep(3)},
	}
	ids := coveredPartIDs(parts, domain.ParsedRelease{Episodes: []int{1, 2}})
	assert.Equal(t, []domain.ID{1, 2}, ids)

	ids = coveredPartIDs(parts, domain.ParsedRelease{})
	assert.Equal(t, []domain.ID{1, 2, 3}, ids)
}

func TestRankByCoverageDropsZero(t *testing.T) {
	scored := []ScoredRelease{
		{Release: domain.Release{Title: "a", Seeders: 5}, Parsed: domain.ParsedRelease{Episodes: []int{9}}},
		{Release: domain.Release{Title: "b", Seeders: 5}, Parsed: domain.ParsedRelease{Episodes: []int{1, 2}}},
	}
	ranked := rankByCoverage(scored, []int{1, 2})
	require.Len(t, ranked, 1)
	assert.Equal(t, "b", ranked[0].Release.Title)
}
