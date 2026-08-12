package torznab

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:torznab="http://torznab.com/specified/1.0">
  <channel>
    <item>
      <title>Movie.1999.1080p.WEB-DL.x264-GRP</title>
      <link>http://tracker/get/1</link>
      <enclosure url="http://tracker/get/1.torrent" length="1500000000" type="application/x-bittorrent" />
      <pubDate>Mon, 02 Jan 2023 15:04:05 +0000</pubDate>
      <torznab:attr name="magneturl" value="magnet:?xt=urn:btih:abc" />
      <torznab:attr name="seeders" value="42" />
      <torznab:attr name="size" value="1500000000" />
    </item>
    <item>
      <title>Movie.1999.720p.WEBRip-OTHER</title>
      <enclosure url="http://tracker/get/2.torrent" length="800000000" type="application/x-bittorrent" />
      <pubDate>Mon, 02 Jan 2023 16:04:05 +0000</pubDate>
      <torznab:attr name="infohash" value="deadbeef0123456789abcdef0123456789abcdef" />
      <torznab:attr name="seeders" value="5" />
    </item>
  </channel>
</rss>`

func TestClient_Search(t *testing.T) {
	var lastQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(sampleRSS))
	}))
	defer srv.Close()

	c := New("jackett", srv.URL, "secret")
	releases, err := c.Search(context.Background(), "movie 1999", []int{2000, 2040})
	require.NoError(t, err)
	require.Len(t, releases, 2)

	r1 := releases[0]
	assert.Equal(t, "Movie.1999.1080p.WEB-DL.x264-GRP", r1.Title)
	assert.Equal(t, "http://tracker/get/1.torrent", r1.DownloadURL)
	assert.Equal(t, "magnet:?xt=urn:btih:abc", r1.MagnetURI)
	assert.Equal(t, 42, r1.Seeders)
	assert.Equal(t, int64(1500000000), r1.Size)
	assert.Equal(t, "jackett", r1.Indexer)
	assert.False(t, r1.PublishDate.IsZero())

	r2 := releases[1]
	assert.Equal(t, "Movie.1999.720p.WEBRip-OTHER", r2.Title)
	assert.Equal(t, "deadbeef0123456789abcdef0123456789abcdef", r2.InfoHash)
	assert.Equal(t, "magnet:?xt=urn:btih:deadbeef0123456789abcdef0123456789abcdef", r2.MagnetURI)

	q, err := url.ParseQuery(lastQuery)
	require.NoError(t, err)
	assert.Equal(t, "search", q.Get("t"))
	assert.Equal(t, "movie 1999", q.Get("q"))
	assert.Equal(t, "2000,2040", q.Get("cat"))
	assert.Equal(t, "secret", q.Get("apikey"))
}

func TestClient_Search_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New("jackett", srv.URL, "")
	_, err := c.Search(context.Background(), "x", nil)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "HTTP 500"))
}

func TestParseResults_Empty(t *testing.T) {
	releases, err := parseResults([]byte(`<?xml version="1.0"?><rss><channel></channel></rss>`), "x")
	require.NoError(t, err)
	assert.Empty(t, releases)
}

func TestClient_RSS(t *testing.T) {
	var lastQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(sampleRSS))
	}))
	defer srv.Close()

	c := New("prowlarr", srv.URL, "secret")
	releases, err := c.RSS(context.Background(), []int{5000, 5030}, time.Unix(1700000000, 0))
	require.NoError(t, err)
	require.Len(t, releases, 2)

	q, err := url.ParseQuery(lastQuery)
	require.NoError(t, err)
	assert.Equal(t, "rss", q.Get("t"))
	assert.Equal(t, "5000,5030", q.Get("cat"))
	assert.Equal(t, "1700000000", q.Get("since"))
	assert.Equal(t, "secret", q.Get("apikey"))
}

func TestClient_RSS_NoSince(t *testing.T) {
	var lastQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(sampleRSS))
	}))
	defer srv.Close()

	c := New("prowlarr", srv.URL, "")
	_, err := c.RSS(context.Background(), nil, time.Time{})
	require.NoError(t, err)

	q, err := url.ParseQuery(lastQuery)
	require.NoError(t, err)
	assert.Equal(t, "rss", q.Get("t"))
	assert.Equal(t, "", q.Get("since"))
}

const sampleCaps = `<?xml version="1.0" encoding="UTF-8"?>
<caps>
  <server version="1.0" title="Jackett" />
  <searching>
    <search available="yes" supportedParams="q" />
    <tv-search available="yes" supportedParams="q,season,ep" />
  </searching>
  <categories />
</caps>`

func TestClient_TestConnection_OK(t *testing.T) {
	var lastQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(sampleCaps))
	}))
	defer srv.Close()

	c := New("jackett", srv.URL, "secret")
	require.NoError(t, c.TestConnection(context.Background()))

	q, err := url.ParseQuery(lastQuery)
	require.NoError(t, err)
	assert.Equal(t, "caps", q.Get("t"))
	assert.Equal(t, "secret", q.Get("apikey"))
}

func TestClient_TestConnection_ErrorElement(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<?xml version="1.0"?><error code="100" description="Invalid API Key (Unauthorized)" />`))
	}))
	defer srv.Close()

	c := New("jackett", srv.URL, "bad")
	err := c.TestConnection(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid API Key")
}

func TestClient_TestConnection_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := New("jackett", srv.URL, "")
	err := c.TestConnection(context.Background())
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "HTTP 503"))
}
