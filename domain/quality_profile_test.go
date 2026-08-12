package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQualityProfile_Validate(t *testing.T) {
	tests := []struct {
		name    string
		p       *QualityProfile
		wantErr error
	}{
		{
			name:    "movie profile allowed",
			p:       &QualityProfile{Name: "Movies", Type: MediaTypeMovie, Allowed: []Quality{{Kind: QualityKindVideoMovie, Name: "1080p WEB-DL"}}},
			wantErr: nil,
		},
		{
			name:    "series profile allowed",
			p:       &QualityProfile{Name: "Series", Type: MediaTypeSeries, Allowed: []Quality{{Kind: QualityKindVideoSeries, Name: "1080p WEB-DL"}}},
			wantErr: nil,
		},
		{
			name:    "book profile rejected",
			p:       &QualityProfile{Name: "Books", Type: MediaTypeBook, Allowed: []Quality{{Kind: QualityKindBook, Name: "epub"}}},
			wantErr: ErrProfileTypeNotAllowed,
		},
		{
			name:    "music profile allowed",
			p:       &QualityProfile{Name: "Music", Type: MediaTypeMusicAlbum, Allowed: []Quality{{Kind: QualityKindAudio, Name: "flac"}}},
			wantErr: nil,
		},
		{
			name:    "empty name rejected",
			p:       &QualityProfile{Name: "", Type: MediaTypeMovie},
			wantErr: ErrEmptyName,
		},
		{
			name:    "wrong kind in allowed rejected",
			p:       &QualityProfile{Name: "Movies", Type: MediaTypeMovie, Allowed: []Quality{{Kind: QualityKindVideoSeries, Name: "1080p WEB-DL"}}},
			wantErr: ErrInvalidQuality,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.p.Validate()
			assert.Equal(t, tt.wantErr, err)
		})
	}
}
