package direct

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

func waitFor(t *testing.T, g *Grabber, jobID string, timeout time.Duration) domain.GrabStatus {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		st, err := g.Status(context.Background(), domain.GrabHandle{JobID: jobID})
		require.NoError(t, err)
		if st.State == domain.GrabCompleted || st.State == domain.GrabFailed {
			return st
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for grab state")
	return domain.GrabStatus{}
}

func TestDirectGrabber_Download(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello-bytes"))
	}))
	defer srv.Close()

	g := New("http", t.TempDir())

	assert.True(t, g.Supports(domain.GrabTarget{URL: "http://x/f.mkv"}))
	assert.False(t, g.Supports(domain.GrabTarget{URL: "magnet:?x"}))

	handle, err := g.Submit(context.Background(), domain.GrabTarget{URL: srv.URL + "/movie.mkv"})
	require.NoError(t, err)
	assert.Equal(t, "http", handle.GrabberName)

	st := waitFor(t, g, handle.JobID, 2*time.Second)
	require.Equal(t, domain.GrabCompleted, st.State)
	require.Len(t, st.OutputFiles, 1)
	assert.Equal(t, 1.0, st.Progress)

	info, err := os.Stat(st.OutputFiles[0])
	require.NoError(t, err)
	assert.Equal(t, int64(len("hello-bytes")), info.Size())
	assert.Equal(t, ".mkv", filepath.Ext(st.OutputFiles[0]))
}

func TestDirectGrabber_Progress(t *testing.T) {
	const payloadSize = 256 * 1024
	payload := make([]byte, payloadSize)
	for i := range payload {
		payload[i] = byte(i)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", payloadSize))
		w.WriteHeader(http.StatusOK)
		w.Write(payload)
	}))
	defer srv.Close()

	g := New("http", t.TempDir())
	handle, err := g.Submit(context.Background(), domain.GrabTarget{URL: srv.URL + "/movie.mkv"})
	require.NoError(t, err)

	var seenProgress float64
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		st, err := g.Status(context.Background(), domain.GrabHandle{JobID: handle.JobID})
		require.NoError(t, err)
		if st.Progress > seenProgress {
			seenProgress = st.Progress
		}
		if st.State == domain.GrabCompleted {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	require.Greater(t, seenProgress, 0.0, "expected progress to advance above 0 during download")
}

func TestDirectGrabber_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	g := New("http", t.TempDir())
	handle, err := g.Submit(context.Background(), domain.GrabTarget{URL: srv.URL + "/missing"})
	require.NoError(t, err)

	st := waitFor(t, g, handle.JobID, 2*time.Second)
	assert.Equal(t, domain.GrabFailed, st.State)
	assert.Contains(t, st.Error, "404")
}
