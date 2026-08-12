package application

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNamingService_Build(t *testing.T) {
	n := NewNamingService()

	assert.Equal(t,
		filepath.Join("/lib/movies", "My Movie", "My Movie - 1080p.mkv"),
		n.Build("/lib/movies", "", "My Movie", "1080p", ".mkv"),
	)

	assert.Equal(t,
		filepath.Join("/lib/books", "Dune", "Dune.epub"),
		n.Build("/lib/books", "", "Dune", "", "epub"),
	)

	assert.Equal(t,
		filepath.Join("/lib", "Clean Title", "Clean Title.mkv"),
		n.Build("/lib", "", "Clean: Title?", "", ".mkv"),
	)
}

func TestNamingService_BuildCustomFolder(t *testing.T) {
	n := NewNamingService()

	assert.Equal(t,
		filepath.Join("/lib/movies", "Custom Folder", "My Movie - 1080p.mkv"),
		n.Build("/lib/movies", "Custom Folder", "My Movie", "1080p", ".mkv"),
	)

	assert.Equal(t,
		filepath.Join("/lib/books", "Custom_Folder", "Dune.epub"),
		n.Build("/lib/books", "Custom_Folder", "Dune", "", "epub"),
	)

	assert.Equal(t,
		filepath.Join("/lib", "Dir", "Clean Title.mkv"),
		n.Build("/lib", "Dir", "Clean: Title?", "", ".mkv"),
	)
}

func TestNamingService_BuildEpisodeCustomFolder(t *testing.T) {
	n := NewNamingService()
	name := ptr("Pilot")

	assert.Equal(t,
		filepath.Join("/lib/series", "Custom Show", "Season 1", "Show S01E01 - Pilot.mkv"),
		n.BuildEpisode("/lib/series", "Custom Show", "Show", 1, 1, name, "", "", ".mkv"),
	)
}

func TestNamingService_BuildEpisode(t *testing.T) {
	n := NewNamingService()
	name := ptr("Pilot")

	assert.Equal(t,
		filepath.Join("/lib/series", "Foo", "Season 1", "Foo S01E01 - Pilot - 1080p.mkv"),
		n.BuildEpisode("/lib/series", "", "Foo", 1, 1, name, "", "1080p", ".mkv"),
	)

	assert.Equal(t,
		filepath.Join("/lib/series", "Foo", "Season 12", "Foo S12E03.mkv"),
		n.BuildEpisode("/lib/series", "", "Foo", 12, 3, nil, "", "", ".mkv"),
	)

	assert.Equal(t,
		filepath.Join("/lib/series", "Foo", "S01", "Foo S01E01 - Pilot.mkv"),
		n.BuildEpisode("/lib/series", "", "Foo", 1, 1, name, "S%02d", "", "mkv"),
	)

	assert.Equal(t,
		filepath.Join("/lib/series", "Foo", "Season 1", "Foo S01E01 - Pilot.mkv"),
		n.BuildEpisode("/lib/series", "", "Foo", 1, 1, name, "Season %d", "", "mkv"),
	)
}

func TestNamingService_BuildAlbum(t *testing.T) {
	n := NewNamingService()

	assert.Equal(t,
		filepath.Join("/lib/music", "Queen", "A Night at the Opera", "01 - Bohemian Rhapsody.flac"),
		n.BuildAlbum("/lib/music", "Queen", "A Night at the Opera", 1, "Bohemian Rhapsody", "", ".flac"),
	)

	assert.Equal(t,
		filepath.Join("/lib/music", "Queen", "A Night at the Opera", "03 - You're My Best Friend - flac.flac"),
		n.BuildAlbum("/lib/music", "Queen", "A Night at the Opera", 3, "You're My Best Friend", "flac", "flac"),
	)

	assert.Equal(t,
		filepath.Join("/lib/music", "Clean Artist", "Clean Album", "02 - Clean Title.mp3"),
		n.BuildAlbum("/lib/music", "Clean: Artist?", "Clean: Album?", 2, "Clean: Title?", "", ".mp3"),
	)

	assert.Equal(t,
		filepath.Join("/lib/music", "Only Album", "01 - Track.flac"),
		n.BuildAlbum("/lib/music", "", "Only Album", 1, "Track", "", ".flac"),
	)
}

func TestFormatSeason(t *testing.T) {
	assert.Equal(t, "Season 1", formatSeason("", 1))
	assert.Equal(t, "Season 12", formatSeason("", 12))
	assert.Equal(t, "S01", formatSeason("S%02d", 1))
	assert.Equal(t, "Season 1", formatSeason("Season %d", 1))
	assert.Equal(t, "Specials", formatSeason("Specials", 0))
}

func ptr[T any](v T) *T { return &v }
