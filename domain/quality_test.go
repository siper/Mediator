package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQualityKindFor(t *testing.T) {
	tests := []struct {
		name   string
		t      MediaType
		want   QualityKind
		wantOk bool
	}{
		{"movie", MediaTypeMovie, QualityKindVideoMovie, true},
		{"series", MediaTypeSeries, QualityKindVideoSeries, true},
		{"book", MediaTypeBook, QualityKindBook, true},
		{"album", MediaTypeMusicAlbum, QualityKindAudio, true},
		{"unknown", MediaType(99), "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := QualityKindFor(tt.t)
			assert.Equal(t, tt.wantOk, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestQuality_RankAndValid(t *testing.T) {
	assert.Equal(t, 0, Quality{QualityKindVideoMovie, "CAM"}.Rank())
	assert.True(t, Quality{QualityKindVideoMovie, "1080p WEB-DL"}.Valid())
	assert.False(t, Quality{QualityKindVideoMovie, "8k"}.Valid())
	assert.True(t, Quality{QualityKindBook, "epub"}.Valid())
	assert.True(t, Quality{QualityKindAudio, "flac"}.Valid())
	assert.False(t, Quality{QualityKindBook, "1080p"}.Valid())
}

func TestQuality_RankOrderedByResolutionThenSource(t *testing.T) {
	cam := Quality{QualityKindVideoMovie, "CAM"}.Rank()
	web720 := Quality{QualityKindVideoMovie, "720p WEB-DL"}.Rank()
	br720 := Quality{QualityKindVideoMovie, "720p BluRay"}.Rank()
	web1080 := Quality{QualityKindVideoMovie, "1080p WEB-DL"}.Rank()
	assert.Less(t, cam, web720, "CAM should rank below 720p WEB-DL")
	assert.Less(t, web720, br720, "720p WEB-DL below 720p BluRay")
	assert.Less(t, br720, web1080, "720p BluRay below 1080p WEB-DL")
}

func TestQuality_AtLeast(t *testing.T) {
	tests := []struct {
		name   string
		q      Quality
		cutoff Quality
		want   bool
	}{
		{"equal", q(QualityKindVideoMovie, "1080p WEB-DL"), q(QualityKindVideoMovie, "1080p WEB-DL"), true},
		{"higher", q(QualityKindVideoMovie, "2160p WEB-DL"), q(QualityKindVideoMovie, "1080p WEB-DL"), true},
		{"lower", q(QualityKindVideoMovie, "720p WEB-DL"), q(QualityKindVideoMovie, "1080p WEB-DL"), false},
		{"unknown quality vs cutoff", q(QualityKindVideoMovie, "8k"), q(QualityKindVideoMovie, "1080p WEB-DL"), false},
		{"cross-kind ignored", q(QualityKindBook, "epub"), q(QualityKindVideoMovie, "CAM"), false},
		{"cross-video-kind ignored", q(QualityKindVideoMovie, "1080p WEB-DL"), q(QualityKindVideoSeries, "1080p WEB-DL"), false},
		{"book epub vs fb2", q(QualityKindBook, "epub"), q(QualityKindBook, "fb2"), true},
		{"audio flac vs mp3", q(QualityKindAudio, "flac"), q(QualityKindAudio, "mp3"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.q.AtLeast(tt.cutoff))
		})
	}
}

func TestQualities_CatalogCompleteness(t *testing.T) {
	for _, kind := range []QualityKind{QualityKindVideoMovie, QualityKindVideoSeries, QualityKindBook, QualityKindAudio} {
		qs := Qualities(kind)
		assert.NotEmpty(t, qs)
		for _, qq := range qs {
			assert.True(t, qq.Valid(), "%s/%s should be valid", kind, qq.Name)
		}
	}
}

func q(kind QualityKind, name string) Quality { return Quality{Kind: kind, Name: name} }
