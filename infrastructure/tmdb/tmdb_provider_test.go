package tmdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

func newTestProvider(t *testing.T, mux *http.ServeMux) *TMDBProvider {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	origBaseURL := baseURL
	t.Cleanup(func() { baseURL = origBaseURL })
	baseURL = srv.URL
	return &TMDBProvider{
		apiKey: "test-token",
		lang:   "en-US",
		http:   srv.Client(),
	}
}

func TestTMDBProvider_Search_ReturnsMoviesAndSeries(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search/multi", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.NotContains(t, r.URL.RawQuery, "api_key")
		json.NewEncoder(w).Encode(tmdbSearchResponse{
			Results: []tmdbSearchItem{
				{ID: 1, MediaType: "movie", Title: "Test Movie", Overview: "overview", PosterPath: "/poster.jpg", ReleaseDate: "1999-03-31"},
				{ID: 2, MediaType: "tv", Name: "Test Series", Overview: "overview", PosterPath: "/poster2.jpg", FirstAirDate: "2008-01-20"},
				{ID: 3, MediaType: "person", Title: "Actor", Overview: "overview"},
			},
		})
	})
	p := newTestProvider(t, mux)

	results, _, err := p.Search("test", nil, 1, 25)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, domain.MediaTypeMovie, results[0].MediaType)
	assert.Equal(t, "Test Movie", results[0].Title)
	assert.Equal(t, "https://image.tmdb.org/t/p/w500/poster.jpg", results[0].CoverURL)
	require.NotNil(t, results[0].Year)
	assert.Equal(t, 1999, *results[0].Year)
	assert.Equal(t, domain.MediaTypeSeries, results[1].MediaType)
	assert.Equal(t, "Test Series", results[1].Title)
	require.NotNil(t, results[1].Year)
	assert.Equal(t, 2008, *results[1].Year)
}

func TestTMDBProvider_Search_FiltersByMediaType(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search/multi", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(tmdbSearchResponse{
			Results: []tmdbSearchItem{
				{ID: 1, MediaType: "movie", Title: "Movie"},
				{ID: 2, MediaType: "tv", Name: "Series"},
			},
		})
	})
	p := newTestProvider(t, mux)

	mt := domain.MediaTypeMovie
	results, _, err := p.Search("test", &mt, 1, 25)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Movie", results[0].Title)
}

func TestTMDBProvider_Search_ReturnsErrorOnAuthFailure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search/multi", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status_code":7,"status_message":"Invalid API key","success":false}`))
	})
	p := newTestProvider(t, mux)

	results, _, err := p.Search("test", nil, 1, 25)
	require.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "401")
}

func TestTMDBProvider_Search_UsesBearerHeader(t *testing.T) {
	var receivedAuth string
	mux := http.NewServeMux()
	mux.HandleFunc("/search/multi", func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(tmdbSearchResponse{Results: nil})
	})
	p := newTestProvider(t, mux)

	_, _, _ = p.Search("test", nil, 1, 25)
	assert.Equal(t, "Bearer test-token", receivedAuth)
}

func TestTMDBProvider_GetMedia_Movie(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/movie/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		json.NewEncoder(w).Encode(tmdbMovie{
			Title:      "Test Movie",
			Overview:   "overview",
			PosterPath: "/poster.jpg",
		})
	})
	p := newTestProvider(t, mux)

	pm, err := p.GetMedia("123", domain.MediaTypeMovie)
	require.NoError(t, err)
	require.NotNil(t, pm)
	assert.Equal(t, "123", pm.ExternalID)
	assert.Equal(t, "Test Movie", pm.Title)
	assert.Equal(t, domain.MediaTypeMovie, pm.MediaType)
	assert.Equal(t, "https://image.tmdb.org/t/p/w500/poster.jpg", pm.CoverURL)
}

func TestTMDBProvider_GetMedia_TVSeries(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/tv/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		json.NewEncoder(w).Encode(tmdbTV{
			Name:       "Test Series",
			Overview:   "overview",
			PosterPath: "/poster.jpg",
			Seasons: []tmdbTVSeason{
				{SeasonNumber: 0, Name: "Specials"},
				{SeasonNumber: 1, Name: "Season 1"},
			},
		})
	})
	mux.HandleFunc("/tv/456/season/1", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(tmdbSeasonDetail{
			Name: "Season 1",
			Episodes: []tmdbEpisode{
				{EpisodeNumber: 1, Name: "Pilot"},
				{EpisodeNumber: 2, Name: "Episode 2"},
			},
		})
	})
	p := newTestProvider(t, mux)

	pm, err := p.GetMedia("456", domain.MediaTypeSeries)
	require.NoError(t, err)
	require.NotNil(t, pm)
	assert.Equal(t, "456", pm.ExternalID)
	assert.Equal(t, "Test Series", pm.Title)
	assert.Equal(t, domain.MediaTypeSeries, pm.MediaType)
	require.Len(t, pm.Groups, 1)
	assert.Equal(t, "Season 1", pm.Groups[0].Name)
	assert.Equal(t, 1, pm.Groups[0].Order)
	require.Len(t, pm.Parts, 2)
	assert.Equal(t, "Pilot", *pm.Parts[0].Name)
}

func TestTMDBProvider_GetMedia_ErrorsOnNonOKStatus(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/movie/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"status_code":34,"status_message":"The resource you requested was not found","success":false}`))
	})
	p := newTestProvider(t, mux)

	pm, err := p.GetMedia("999", domain.MediaTypeMovie)
	require.Error(t, err)
	assert.Nil(t, pm)
	assert.Contains(t, err.Error(), "404")
}

func TestTMDBProvider_GetMedia_ReturnsErrorForUnsupportedType(t *testing.T) {
	p := NewTMDBProvider("test-token", "en-US", nil)
	pm, err := p.GetMedia("1", domain.MediaTypeBook)
	require.Error(t, err)
	assert.Nil(t, pm)
}

func TestTMDBProvider_Name(t *testing.T) {
	p := NewTMDBProvider("key", "en-US", nil)
	assert.Equal(t, "tmdb", p.Name())
}

func TestTMDBProvider_BuildCoverURL(t *testing.T) {
	p := NewTMDBProvider("key", "en-US", nil)
	assert.Equal(t, "", p.buildCoverURL(""))
	assert.Equal(t, "https://image.tmdb.org/t/p/w500/poster.jpg", p.buildCoverURL("/poster.jpg"))
}

func TestTMDBProvider_Search_UsesConfigurableLanguage(t *testing.T) {
	var receivedLang string
	mux := http.NewServeMux()
	mux.HandleFunc("/search/multi", func(w http.ResponseWriter, r *http.Request) {
		receivedLang = r.URL.Query().Get("language")
		json.NewEncoder(w).Encode(tmdbSearchResponse{Results: nil})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	origBaseURL := baseURL
	t.Cleanup(func() { baseURL = origBaseURL })
	baseURL = srv.URL

	p := &TMDBProvider{
		apiKey: "test-token",
		lang:   "ru-RU",
		http:   srv.Client(),
	}

	_, _, _ = p.Search("test", nil, 1, 25)
	assert.Equal(t, "ru-RU", receivedLang)
}

func TestTMDBProvider_GetMedia_Movie_UsesConfigurableLanguage(t *testing.T) {
	var receivedLang string
	mux := http.NewServeMux()
	mux.HandleFunc("/movie/", func(w http.ResponseWriter, r *http.Request) {
		receivedLang = r.URL.Query().Get("language")
		json.NewEncoder(w).Encode(tmdbMovie{Title: "Test", Overview: "ov", PosterPath: "/p.jpg"})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	origBaseURL := baseURL
	t.Cleanup(func() { baseURL = origBaseURL })
	baseURL = srv.URL

	p := &TMDBProvider{
		apiKey: "test-token",
		lang:   "ru-RU",
		http:   srv.Client(),
	}
	_, err := p.GetMedia("123", domain.MediaTypeMovie)
	require.NoError(t, err)
	assert.Equal(t, "ru-RU", receivedLang)
}

func TestTMDBProvider_Search_NilMediaTypeReturnsAll(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search/multi", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(tmdbSearchResponse{
			Results: []tmdbSearchItem{
				{ID: 1, MediaType: "movie", Title: "Movie"},
				{ID: 2, MediaType: "tv", Name: "Series"},
			},
		})
	})
	p := newTestProvider(t, mux)

	results, _, err := p.Search("test", nil, 1, 25)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestTMDBProvider_Search_EmptyResults(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search/multi", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(tmdbSearchResponse{Results: nil})
	})
	p := newTestProvider(t, mux)

	results, _, err := p.Search("nonexistent", nil, 1, 25)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestTMDBProvider_Search_ReadsBodyFromResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search/multi", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results":[{"id":1,"media_type":"movie","title":"Foo","overview":"bar","poster_path":"/p.jpg"}]}`))
	})
	p := newTestProvider(t, mux)

	results, _, err := p.Search("foo", nil, 1, 25)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Foo", results[0].Title)
	assert.Equal(t, "https://image.tmdb.org/t/p/w500/p.jpg", results[0].CoverURL)
}

func TestTMDBProvider_TestConnection_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/configuration", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	})
	p := newTestProvider(t, mux)

	err := p.TestConnection(context.Background())
	assert.NoError(t, err)
}

func TestTMDBProvider_TestConnection_InvalidKey(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/configuration", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	p := newTestProvider(t, mux)

	err := p.TestConnection(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}
