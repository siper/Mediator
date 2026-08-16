package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

func TestReleaseParser_Video(t *testing.T) {
	p := NewReleaseParser()

	tests := []struct {
		name      string
		title     string
		wantQ     string
		wantSrc   string
		wantCodec string
		wantYear  int
		wantGroup string
		wantRep   bool
	}{
		{"1080p webdl", "The.Matrix.1999.1080p.WEB-DL.x264-GROUP", "1080p WEB-DL", "WEB-DL", "x264", 1999, "GROUP", false},
		{"2160p bluray hevc", "Movie.Name.2023.2160p.BluRay.x265-NOGRP", "2160p BluRay", "BluRay", "x265", 2023, "NOGRP", false},
		{"720p webrip", "Some.Movie.720p.WEBRip.XviD-XXX", "720p WEBRip", "WEBRip", "xvid", 0, "XXX", false},
		{"4k alias", "Inception.2010.4K.WEB-DL-G", "2160p WEB-DL", "WEB-DL", "", 2010, "G", false},
		{"cam junk", "Some.Movie.CAM.XviD-LOL", "CAM", "CAM", "xvid", 0, "LOL", false},
		{"repack", "Show.S01E01.REPACK.1080p.WEB-DL-GRP", "1080p WEB-DL", "WEB-DL", "", 0, "GRP", true},
		{"h264 dotted", "Film.2010.1080p.BluRay.H.264-RG", "1080p BluRay", "BluRay", "h264", 2010, "RG", false},
		{"no quality", "Some.Random.Title-RG", "", "", "", 0, "RG", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr, err := p.Parse(tt.title, domain.MediaTypeMovie)
			require.NoError(t, err)
			assert.Equal(t, tt.wantQ, pr.Quality.Name)
			assert.Equal(t, tt.wantSrc, pr.Source)
			assert.Equal(t, tt.wantCodec, pr.Codec)
			assert.Equal(t, tt.wantGroup, pr.Group)
			assert.Equal(t, tt.wantRep, pr.IsRepack)
			if tt.wantYear != 0 {
				require.NotNil(t, pr.Year)
				assert.Equal(t, tt.wantYear, *pr.Year)
			}
			if tt.wantQ != "" {
				assert.True(t, pr.Quality.Valid())
			}
		})
	}
}

func TestReleaseParser_VideoKindSeparated(t *testing.T) {
	p := NewReleaseParser()
	movie, err := p.Parse("Show.1080p.WEB-DL-X", domain.MediaTypeMovie)
	require.NoError(t, err)
	assert.Equal(t, domain.QualityKindVideoMovie, movie.Quality.Kind)
	series, err := p.Parse("Show.1080p.WEB-DL-X", domain.MediaTypeSeries)
	require.NoError(t, err)
	assert.Equal(t, domain.QualityKindVideoSeries, series.Quality.Kind)
}

func TestReleaseParser_BookAudio(t *testing.T) {
	p := NewReleaseParser()

	t.Run("book detects epub over lower-ranked tokens", func(t *testing.T) {
		pr, err := p.Parse("Author.Book.Title.epub", domain.MediaTypeBook)
		require.NoError(t, err)
		assert.Equal(t, domain.Quality{Kind: domain.QualityKindBook, Name: "epub"}, pr.Quality)
	})

	t.Run("book picks highest-ranked format when multiple", func(t *testing.T) {
		pr, err := p.Parse("Book.fb2.and.also.epub", domain.MediaTypeBook)
		require.NoError(t, err)
		assert.Equal(t, "epub", pr.Quality.Name)
	})

	t.Run("audio detects flac", func(t *testing.T) {
		pr, err := p.Parse("Artist.Album.FLAC", domain.MediaTypeMusicAlbum)
		require.NoError(t, err)
		assert.Equal(t, domain.Quality{Kind: domain.QualityKindAudio, Name: "flac"}, pr.Quality)
	})

	t.Run("no format leaves quality empty", func(t *testing.T) {
		pr, err := p.Parse("Unknown.Release", domain.MediaTypeBook)
		require.NoError(t, err)
		assert.False(t, pr.Quality.Valid())
	})
}

func TestReleaseParser_EmptyTitle(t *testing.T) {
	p := NewReleaseParser()
	_, err := p.Parse("", domain.MediaTypeMovie)
	assert.Error(t, err)
}

func TestReleaseParser_Series(t *testing.T) {
	p := NewReleaseParser()

	intVal := func(v int) *int { return &v }

	tests := []struct {
		name         string
		title        string
		wantSeason   *int
		wantSeasons  []int
		wantEps      []int
		wantComplete bool
	}{
		{"single episode", "Show.S01E02.1080p.WEB-DL-GRP", intVal(1), []int{1}, []int{2}, false},
		{"repack episode", "Show.S01E01.REPACK.1080p.WEB-DL-GRP", intVal(1), []int{1}, []int{1}, false},
		{"multi discrete", "Show.S01E02E03.1080p.WEB-DL-GRP", intVal(1), []int{1}, []int{2, 3}, false},
		{"range dash", "Show.S01E02-E04.1080p.WEB-DL-GRP", intVal(1), []int{1}, []int{2, 3, 4}, false},
		{"range no E", "Show.S01E02-04.1080p.WEB-DL-GRP", intVal(1), []int{1}, []int{2, 3, 4}, false},
		{"xformat", "Show.1x02.1080p.WEB-DL-GRP", intVal(1), []int{1}, []int{2}, false},
		{"season pack complete", "Show.S01.Complete.1080p.WEB-DL-GRP", intVal(1), []int{1}, nil, false},
		{"season pack season keyword", "Show.Season.2.1080p.WEB-DL-GRP", intVal(2), []int{2}, nil, false},
		{"season 2 ep 10", "Show.S02E10.720p.HDTV-X", intVal(2), []int{2}, []int{10}, false},
		{"multi-season range", "Show.S01-S05.1080p.WEB-DL-GRP", intVal(1), []int{1, 2, 3, 4, 5}, nil, false},
		{"multi-season word range", "Show.Season.1-3.1080p.WEB-DL-GRP", intVal(1), []int{1, 2, 3}, nil, false},
		{"multi-season with ep range", "Dr. Stone - S1-4E1-70 - 2019-2025 DUB WEBRip 2160p", intVal(1), []int{1, 2, 3, 4}, epRange(1, 70), false},
		{"amphibia complete pack", "Amphibia - S1-3E1-58 - 2019-2022 DUB (Nevafilm), Sub WEBDL 1080p - RUSSIAN", intVal(1), []int{1, 2, 3}, epRange(1, 58), false},
		{"amphibia season 3 pack", "Amphibia - S3E1-18 - 2021-2022 MVO (AniMaunt) WEBDL 1080p - RUSSIAN", intVal(3), []int{3}, epRange(1, 18), false},
		{"amphibia season 2 pack", "Amphibia - S2E1-20 - 2020-2021 MVO (HDRezka Studio) WEBDL 1080p - RUSSIAN", intVal(2), []int{2}, epRange(1, 20), false},
		{"season ep range compact", "Dr. Stone: Stone Wars - S2E1-11 - 2021 MVO", intVal(2), []int{2}, epRange(1, 11), false},
		{"complete series no season", "Show.Complete.Series.1080p.WEB-DL-GRP", nil, nil, nil, true},
		{"no season info", "Some.Random.Show.1080p.WEB-DL-GRP", nil, nil, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr, err := p.Parse(tt.title, domain.MediaTypeSeries)
			require.NoError(t, err)
			if tt.wantSeason == nil {
				assert.Nil(t, pr.Season)
			} else {
				require.NotNil(t, pr.Season)
				assert.Equal(t, *tt.wantSeason, *pr.Season)
			}
			assert.Equal(t, tt.wantSeasons, pr.Seasons)
			assert.Equal(t, tt.wantEps, pr.Episodes)
			assert.Equal(t, tt.wantComplete, pr.Complete)
		})
	}
}

func epRange(start, end int) []int {
	out := make([]int, 0, end-start+1)
	for i := start; i <= end; i++ {
		out = append(out, i)
	}
	return out
}
