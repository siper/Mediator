package qbittorrent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

func newMockServer(t *testing.T, infoState string, progress float64) *httptest.Server {
	return newMockServerWithLogin(t, infoState, progress, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Ok."))
	}, http.StatusOK)
}

func newMockServerV5(t *testing.T, infoState string, progress float64) *httptest.Server {
	return newMockServerWithLogin(t, infoState, progress, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}, http.StatusAccepted)
}

func newMockServerWithLogin(t *testing.T, infoState string, progress float64, loginHandler http.HandlerFunc, addOKStatus int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/auth/login":
			loginHandler(w, r)
		case "/api/v2/torrents/add":
			assert.Equal(t, "magnet:?xt=urn:btih:abc123", r.FormValue("urls"))
			w.WriteHeader(addOKStatus)
			_, _ = w.Write([]byte("Ok."))
		case "/api/v2/torrents/info":
			items := []qbTorrent{{
				Hash: "abc123", Name: "Movie", State: infoState,
				Progress: progress, ContentPath: "/data/Movie.mkv",
				AddedOn: time.Now().Unix(),
			}}
			_ = json.NewEncoder(w).Encode(items)
		case "/api/v2/torrents/delete":
			w.WriteHeader(addOKStatus)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func newMockServerForURL(t *testing.T, infoState string, progress float64) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/auth/login":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Ok."))
		case "/api/v2/torrents/add":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Ok."))
		case "/api/v2/torrents/info":
			items := []qbTorrent{{
				Hash:    "deadbeef",
				Name:    "Spider-Man Brand New Day 2026",
				State:   infoState,
				Progress: progress,
				ContentPath: "/data/Spider-Man.Brand.New.Day.2026.avi",
				AddedOn: time.Now().Unix(),
			}}
			_ = json.NewEncoder(w).Encode(items)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestClient_AddAndGet(t *testing.T) {
	srv := newMockServer(t, "downloading", 0.5)
	defer srv.Close()

	c, err := New("qb", srv.URL, "admin", "pass")
	require.NoError(t, err)

	rel := domain.Release{
		Title:     "Movie",
		MagnetURI: "magnet:?xt=urn:btih:abc123",
	}
	task, err := c.Add(context.Background(), rel, "media")
	require.NoError(t, err)
	assert.Equal(t, "abc123", task.ID)
	assert.Equal(t, domain.DownloadQueued, task.Status)

	got, err := c.Get(context.Background(), "abc123")
	require.NoError(t, err)
	assert.Equal(t, "abc123", got.ID)
	assert.Equal(t, domain.Downloading, got.Status)
	assert.InDelta(t, 0.5, got.Progress, 1e-9)
	assert.Equal(t, "/data/Movie.mkv", got.OutputPath)
}

func TestClient_AddAndGet_V5(t *testing.T) {
	srv := newMockServerV5(t, "downloading", 0.5)
	defer srv.Close()

	c, err := New("qb", srv.URL, "admin", "temp_pass")
	require.NoError(t, err)

	rel := domain.Release{
		Title:     "Movie",
		MagnetURI: "magnet:?xt=urn:btih:abc123",
	}
	task, err := c.Add(context.Background(), rel, "media")
	require.NoError(t, err)
	assert.Equal(t, "abc123", task.ID)
	assert.Equal(t, domain.DownloadQueued, task.Status)

	got, err := c.Get(context.Background(), "abc123")
	require.NoError(t, err)
	assert.Equal(t, "abc123", got.ID)
	assert.Equal(t, domain.Downloading, got.Status)
	assert.InDelta(t, 0.5, got.Progress, 1e-9)
	assert.Equal(t, "/data/Movie.mkv", got.OutputPath)
}

func TestClient_AddURL_ResolvesInfoHash(t *testing.T) {
	srv := newMockServerForURL(t, "downloading", 0.4)
	defer srv.Close()

	c, err := New("qb", srv.URL, "admin", "pass")
	require.NoError(t, err)

	rel := domain.Release{
		Title:       "Spider-Man: Brand New Day 2026",
		DownloadURL: "https://tracker.example.com/download.php?id=123",
	}
	task, err := c.Add(context.Background(), rel, "media")
	require.NoError(t, err)
	assert.Equal(t, "deadbeef", task.ID)
	assert.Equal(t, domain.DownloadQueued, task.Status)
}

func TestClient_CompletedFromSeedingState(t *testing.T) {
	srv := newMockServer(t, "stalledUP", 1.0)
	defer srv.Close()

	c, err := New("qb", srv.URL, "admin", "pass")
	require.NoError(t, err)

	_, err = c.Add(context.Background(), domain.Release{MagnetURI: "magnet:?xt=urn:btih:abc123"}, "")
	require.NoError(t, err)

	got, err := c.Get(context.Background(), "abc123")
	require.NoError(t, err)
	assert.Equal(t, domain.DownloadCompleted, got.Status)
}

func TestClient_LoginFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/auth/login" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Fails."))
		}
	}))
	defer srv.Close()

	c, err := New("qb", srv.URL, "admin", "wrong")
	require.NoError(t, err)

	_, err = c.Add(context.Background(), domain.Release{MagnetURI: "magnet:?xt=urn:btih:abc"}, "")
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "login"))
}

func TestClient_LoginFailure_V5(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/auth/login" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("Unauthorized"))
		}
	}))
	defer srv.Close()

	c, err := New("qb", srv.URL, "admin", "wrong")
	require.NoError(t, err)

	_, err = c.Add(context.Background(), domain.Release{MagnetURI: "magnet:?xt=urn:btih:abc"}, "")
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "login"))
}

func TestInfohashFromMagnet(t *testing.T) {
	assert.Equal(t, "abc123", infohashFromMagnet("magnet:?xt=urn:btih:ABC123"))
	assert.Equal(t, "", infohashFromMagnet("not a magnet"))
	assert.Equal(t, "5d6d746c66ec54439cfb99a36338e22e7a313544",
		infohashFromMagnet("magnet:?xt=urn:btih:LVWXI3DG5RKEHHH3TGRWGOHCFZ5DCNKE"))
}

func TestClient_TestConnection_Success(t *testing.T) {
	srv := newMockServer(t, "downloading", 0.5)
	defer srv.Close()

	c, err := New("qb", srv.URL, "admin", "pass")
	require.NoError(t, err)

	err = c.TestConnection(context.Background())
	assert.NoError(t, err)
}

func TestClient_TestConnection_LoginFailure(t *testing.T) {
	srv := newMockServerWithLogin(t, "downloading", 0.5, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("Unauthorized"))
	}, http.StatusOK)
	defer srv.Close()

	c, err := New("qb", srv.URL, "admin", "wrong")
	require.NoError(t, err)

	err = c.TestConnection(context.Background())
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "login"))
}
