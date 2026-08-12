package musicbrainz

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

func newTestProvider(t *testing.T, mux *http.ServeMux) *MusicBrainzProvider {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	origBaseURL := baseURL
	t.Cleanup(func() { baseURL = origBaseURL })
	baseURL = srv.URL
	return &MusicBrainzProvider{
		http: srv.Client(),
	}
}

func TestMusicBrainzProvider_Name(t *testing.T) {
	p := NewProvider(nil)
	assert.Equal(t, "musicbrainz", p.Name())
}

func TestMusicBrainzProvider_Search_ReturnsAlbums(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/release", func(w http.ResponseWriter, r *http.Request) {
		assert.NotEmpty(t, r.Header.Get("User-Agent"))
		q := r.URL.Query().Get("query")
		assert.Equal(t, "queen", q)
		assert.Equal(t, "25", r.URL.Query().Get("limit"))
		assert.Equal(t, "0", r.URL.Query().Get("offset"))
		json.NewEncoder(w).Encode(mbSearchResponse{
			Count:  1,
			Offset: 0,
			Releases: []mbSearchRelease{
				{
					ID:    "release-uuid",
					Title: "A Night at the Opera",
					ArtistCredit: []mbArtistCreditEntry{
						{Name: "Queen", Artist: &mbArtistRef{ID: "artist-uuid", Name: "Queen"}},
					},
					CoverArt: &mbCoverArt{Artwork: true, Available: true},
				},
			},
		})
	})
	p := newTestProvider(t, mux)

	results, hasMore, err := p.Search("queen", nil, 1, 25)
	require.NoError(t, err)
	assert.False(t, hasMore)
	require.Len(t, results, 1)
	assert.Equal(t, "musicbrainz", results[0].ProviderName)
	assert.Equal(t, "release-uuid", results[0].ExternalID)
	assert.Equal(t, "Queen - A Night at the Opera", results[0].Title)
	assert.Equal(t, "https://coverartarchive.org/release/release-uuid/front", results[0].CoverURL)
	assert.Equal(t, domain.MediaTypeMusicAlbum, results[0].MediaType)
}

func TestMusicBrainzProvider_Search_Paginates(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/release", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "10", r.URL.Query().Get("limit"))
		assert.Equal(t, "20", r.URL.Query().Get("offset"))
		json.NewEncoder(w).Encode(mbSearchResponse{
			Count:  50,
			Offset: 20,
			Releases: []mbSearchRelease{
				{ID: "r1", Title: "Album", ArtistCredit: []mbArtistCreditEntry{{Name: "A"}}},
			},
		})
	})
	p := newTestProvider(t, mux)

	results, hasMore, err := p.Search("lorna shore", nil, 3, 10)
	require.NoError(t, err)
	assert.True(t, hasMore)
	require.Len(t, results, 1)
}

func TestMusicBrainzProvider_Search_DedupsReleaseGroup(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/release", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(mbSearchResponse{
			Count:  2,
			Offset: 0,
			Releases: []mbSearchRelease{
				{
					ID:           "rel-a",
					Title:        "Everblack",
					ArtistCredit: []mbArtistCreditEntry{{Name: "Lorna Shore"}},
					CoverArt:     &mbCoverArt{Available: false},
					ReleaseGroup: &mbReleaseGroup{ID: "rg-1"},
				},
				{
					ID:           "rel-b",
					Title:        "Everblack",
					ArtistCredit: []mbArtistCreditEntry{{Name: "Lorna Shore"}},
					CoverArt:     &mbCoverArt{Available: true, Front: true},
					ReleaseGroup: &mbReleaseGroup{ID: "rg-1"},
				},
			},
		})
	})
	p := newTestProvider(t, mux)

	results, hasMore, err := p.Search("lorna", nil, 1, 25)
	require.NoError(t, err)
	assert.False(t, hasMore)
	require.Len(t, results, 1)
	assert.Equal(t, "rel-b", results[0].ExternalID)
}

func TestMusicBrainzProvider_Search_FiltersByMediaType(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/release", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(mbSearchResponse{Releases: nil})
	})
	p := newTestProvider(t, mux)

	mt := domain.MediaTypeMovie
	results, _, err := p.Search("test", &mt, 1, 25)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestMusicBrainzProvider_Search_NoCoverArt(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/release", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(mbSearchResponse{
			Releases: []mbSearchRelease{
				{ID: "no-cover", Title: "No Cover", ArtistCredit: nil, CoverArt: &mbCoverArt{Available: false}},
			},
		})
	})
	p := newTestProvider(t, mux)

	results, _, err := p.Search("test", nil, 1, 25)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "https://coverartarchive.org/release/no-cover/front", results[0].CoverURL)
}

func TestMusicBrainzProvider_Search_EmptyResults(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/release", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(mbSearchResponse{Releases: nil})
	})
	p := newTestProvider(t, mux)

	results, _, err := p.Search("nonexistent", nil, 1, 25)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestMusicBrainzProvider_GetMedia_ReturnsReleaseWithTracks(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/release/", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(mbRelease{
			ID:   "release-uuid",
			Title: "A Night at the Opera",
			ArtistCredit: []mbArtistCreditEntry{
				{Name: "Queen", Artist: &mbArtistRef{Name: "Queen"}},
			},
			LastUpdated: "2024-01-01T00:00:00",
			MediumList: []mbMedium{
				{
					Position: 1,
					TrackList: []mbTrack{
						{Title: "You're My Best Friend", Number: "1"},
						{Title: "Love of My Life", Number: "2"},
					},
				},
			},
			CoverArt: &mbCoverArt{Artwork: true, Available: true},
		})
	})
	p := newTestProvider(t, mux)

	pm, err := p.GetMedia("release-uuid", domain.MediaTypeMusicAlbum)
	require.NoError(t, err)
	require.NotNil(t, pm)
	assert.Equal(t, "release-uuid", pm.ExternalID)
	assert.Equal(t, "Queen - A Night at the Opera", pm.Title)
	assert.Equal(t, domain.MediaTypeMusicAlbum, pm.MediaType)
	assert.Equal(t, domain.MediaStatusCompleted, pm.Status)
	assert.Equal(t, "2024-01-01T00:00:00", pm.LastModified)
	assert.Equal(t, "https://coverartarchive.org/release/release-uuid/front", pm.CoverURL)
	require.Len(t, pm.Groups, 1)
	assert.Equal(t, "Album", pm.Groups[0].Name)
	require.Len(t, pm.Parts, 2)
	assert.Equal(t, "You're My Best Friend", *pm.Parts[0].Name)
	assert.Equal(t, "Love of My Life", *pm.Parts[1].Name)
}

func TestMusicBrainzProvider_GetMedia_MultiDisc(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/release/", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(mbRelease{
			ID:   "multi-uuid",
			Title: "Double Album",
			ArtistCredit: []mbArtistCreditEntry{
				{Artist: &mbArtistRef{Name: "Some Artist"}},
			},
			MediumList: []mbMedium{
				{
					Position: 1,
					TrackList: []mbTrack{
						{Title: "Disc 1 Track 1", Number: "1"},
					},
				},
				{
					Position: 2,
					TrackList: []mbTrack{
						{Title: "Disc 2 Track 1", Number: "1"},
					},
				},
			},
		})
	})
	p := newTestProvider(t, mux)

	pm, err := p.GetMedia("multi-uuid", domain.MediaTypeMusicAlbum)
	require.NoError(t, err)
	require.Len(t, pm.Groups, 2)
	assert.Equal(t, "Disc 1", pm.Groups[0].Name)
	assert.Equal(t, 1, pm.Groups[0].Order)
	assert.Equal(t, "Disc 2", pm.Groups[1].Name)
	assert.Equal(t, 2, pm.Groups[1].Order)
	require.Len(t, pm.Parts, 2)
}

func TestMusicBrainzProvider_GetMedia_NoTracksCreatesNoParts(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/release/", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(mbRelease{
			ID:   "no-tracks",
			Title: "No Tracks Album",
			ArtistCredit: []mbArtistCreditEntry{
				{Artist: &mbArtistRef{Name: "Unknown"}},
			},
			MediumList: nil,
		})
	})
	p := newTestProvider(t, mux)

	pm, err := p.GetMedia("no-tracks", domain.MediaTypeMusicAlbum)
	require.NoError(t, err)
	assert.Empty(t, pm.Parts)
	assert.Empty(t, pm.Groups)
}

func TestMusicBrainzProvider_GetMedia_ReturnsErrorForUnsupportedType(t *testing.T) {
	p := NewProvider(nil)
	pm, err := p.GetMedia("1", domain.MediaTypeMovie)
	require.Error(t, err)
	assert.Nil(t, pm)
}

func TestMusicBrainzProvider_GetMedia_ReturnsErrorOnNonOKStatus(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/release/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	p := newTestProvider(t, mux)

	pm, err := p.GetMedia("999", domain.MediaTypeMusicAlbum)
	require.Error(t, err)
	assert.Nil(t, pm)
}

func TestMusicBrainzProvider_FetchLastModified(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/release/", func(w http.ResponseWriter, r *http.Request) {
		assert.NotEmpty(t, r.Header.Get("User-Agent"))
		json.NewEncoder(w).Encode(mbRelease{
			ID:          "release-uuid",
			Title:       "Test",
			LastUpdated: "2024-06-15T12:00:00",
		})
	})
	p := newTestProvider(t, mux)

	result, err := p.FetchLastModified(context.Background(), "release-uuid", domain.MediaTypeMusicAlbum)
	require.NoError(t, err)
	assert.Equal(t, "2024-06-15T12:00:00", result)
}

func TestMusicBrainzProvider_FetchLastModified_ReturnsEmptyForNonMusic(t *testing.T) {
	p := NewProvider(nil)
	result, err := p.FetchLastModified(nil, "1", domain.MediaTypeMovie)
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestMusicBrainzProvider_TestConnection_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/release", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "a", r.URL.Query().Get("query"))
		assert.NotEmpty(t, r.Header.Get("User-Agent"))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"count":0,"offset":0,"releases":[]}`))
	})
	p := newTestProvider(t, mux)

	err := p.TestConnection(context.Background())
	assert.NoError(t, err)
}

func TestMusicBrainzProvider_TestConnection_ErrorsOnNonOKStatus(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/release", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	p := newTestProvider(t, mux)

	err := p.TestConnection(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "429")
}

func TestMusicBrainzProvider_Search_ReturnsErrorOnNonOKStatus(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/release", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	p := newTestProvider(t, mux)

	results, _, err := p.Search("test", nil, 1, 25)
	require.Error(t, err)
	assert.Nil(t, results)
}
