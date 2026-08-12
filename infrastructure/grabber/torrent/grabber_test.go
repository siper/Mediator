package torrent

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

type fakeRunner struct {
	addErr    error
	getTask   *domain.DownloadTask
	getErr    error
	listTasks []domain.DownloadTask
	removedID string
}

func (f *fakeRunner) Name() string { return "fake" }

func (f *fakeRunner) Add(ctx context.Context, r domain.Release, category string) (*domain.DownloadTask, error) {
	if f.addErr != nil {
		return nil, f.addErr
	}
	return &domain.DownloadTask{ID: "h1", Name: r.Title, Status: domain.DownloadQueued}, nil
}

func (f *fakeRunner) Get(ctx context.Context, id string) (*domain.DownloadTask, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.getTask, nil
}

func (f *fakeRunner) List(ctx context.Context) ([]domain.DownloadTask, error) {
	return f.listTasks, nil
}

func (f *fakeRunner) Remove(ctx context.Context, id string, deleteData bool) error {
	f.removedID = id
	return nil
}

func TestTorrentGrabber_SupportsAndSubmit(t *testing.T) {
	r := &fakeRunner{}
	g := New("torrent", r, nil, "media")

	assert.True(t, g.Supports(domain.GrabTarget{Release: &domain.Release{MagnetURI: "magnet:?x"}}))
	assert.False(t, g.Supports(domain.GrabTarget{URL: "http://x/file"}))
	assert.False(t, g.Supports(domain.GrabTarget{Release: &domain.Release{}}))

	handle, err := g.Submit(context.Background(), domain.GrabTarget{Release: &domain.Release{Title: "M", MagnetURI: "magnet:?x"}})
	require.NoError(t, err)
	assert.Equal(t, "h1", handle.JobID)
	assert.Equal(t, "torrent", handle.GrabberName)
}

func TestTorrentGrabber_StatusMapping(t *testing.T) {
	tests := []struct {
		name   string
		status domain.DownloadStatus
		want   domain.GrabState
		output bool
	}{
		{"queued", domain.DownloadQueued, domain.GrabQueued, false},
		{"downloading", domain.Downloading, domain.GrabRunning, false},
		{"completed", domain.DownloadCompleted, domain.GrabCompleted, true},
		{"failed", domain.DownloadFailed, domain.GrabFailed, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &fakeRunner{getTask: &domain.DownloadTask{ID: "h1", Status: tt.status, OutputPath: "/o/file"}}
			g := New("torrent", r, nil, "")
			st, err := g.Status(context.Background(), domain.GrabHandle{JobID: "h1"})
			require.NoError(t, err)
			assert.Equal(t, tt.want, st.State)
			if tt.output {
				assert.Len(t, st.OutputFiles, 1)
			} else {
				assert.Empty(t, st.OutputFiles)
			}
		})
	}
}

func TestTorrentGrabber_SubmitError(t *testing.T) {
	r := &fakeRunner{addErr: errors.New("nope")}
	g := New("torrent", r, nil, "")
	_, err := g.Submit(context.Background(), domain.GrabTarget{Release: &domain.Release{MagnetURI: "magnet:?x"}})
	assert.Error(t, err)
}

func TestTorrentGrabber_Cancel(t *testing.T) {
	r := &fakeRunner{}
	g := New("torrent", r, nil, "")
	require.NoError(t, g.Cancel(context.Background(), domain.GrabHandle{JobID: "h9"}))
	assert.Equal(t, "h9", r.removedID)
}

func TestTorrentGrabber_StatusFallbackByName(t *testing.T) {
	r := &fakeRunner{
		getErr: domain.ErrPartNotFound,
		listTasks: []domain.DownloadTask{
			{ID: "abc123", Name: "Spider-Man Brand New Day 2026", Status: domain.Downloading, Progress: 0.6, OutputPath: "/data/Movie.mkv"},
		},
	}
	g := New("torrent", r, nil, "media")
	st, err := g.Status(context.Background(), domain.GrabHandle{
		JobID: "http://tracker/download?id=1",
		Name:  "Spider-Man: Brand New Day 2026",
	})
	require.NoError(t, err)
	assert.Equal(t, domain.GrabRunning, st.State)
	assert.InDelta(t, 0.6, st.Progress, 1e-9)
	assert.Equal(t, "abc123", st.ResolvedID)
}

type fakeFS struct {
	files map[string][]string
}

func (f *fakeFS) Move(src, dst string) error      { return nil }
func (f *fakeFS) Exists(path string) bool         { return true }
func (f *fakeFS) Size(path string) (int64, error) { return 0, nil }
func (f *fakeFS) Remove(path string) error        { return nil }
func (f *fakeFS) RemoveAll(path string) error     { return nil }
func (f *fakeFS) ListFiles(dir string) ([]string, error) {
	return f.files[dir], nil
}

func TestTorrentGrabber_EnumeratesDirectory(t *testing.T) {
	r := &fakeRunner{getTask: &domain.DownloadTask{ID: "h1", Status: domain.DownloadCompleted, OutputPath: "/dl/Show.S01"}}
	fs := &fakeFS{files: map[string][]string{
		"/dl/Show.S01": {
			"/dl/Show.S01/Show.S01E01.mkv",
			"/dl/Show.S01/Show.S01E02.mkv",
			"/dl/Show.S01/cover.jpg",
			"/dl/Show.S01/subs.srt",
		},
	}}
	g := New("torrent", r, fs, "")
	st, err := g.Status(context.Background(), domain.GrabHandle{JobID: "h1"})
	require.NoError(t, err)
	assert.Equal(t, domain.GrabCompleted, st.State)
	require.Len(t, st.OutputFiles, 2)
	assert.Equal(t, "/dl/Show.S01/Show.S01E01.mkv", st.OutputFiles[0])
	assert.Equal(t, "/dl/Show.S01/Show.S01E02.mkv", st.OutputFiles[1])
}
