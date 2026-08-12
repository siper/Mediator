package domain

import (
	"testing"
)

func TestMediaType_String(t *testing.T) {
	tests := []struct {
		name      string
		mediaType MediaType
		want      string
	}{
		{"book", MediaTypeBook, "book"},
		{"series", MediaTypeSeries, "series"},
		{"movie", MediaTypeMovie, "movie"},
		{"music_album", MediaTypeMusicAlbum, "music_album"},
		{"unknown", MediaType(255), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mediaType.String(); got != tt.want {
				t.Errorf("MediaType.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMedia_Validate(t *testing.T) {
	category := MediaTypeBook
	emptyName := ""
	cover := "cover.jpg"

	tests := []struct {
		name    string
		media   *Media
		wantErr error
	}{
		{
			name:    "valid media",
			media:   &Media{Name: "Test", Type: MediaTypeMovie},
			wantErr: nil,
		},
		{
			name:    "empty name",
			media:   &Media{Name: "", Type: MediaTypeBook},
			wantErr: ErrEmptyName,
		},
		{
			name:    "invalid media type",
			media:   &Media{Name: "Test", Type: MediaType(99)},
			wantErr: ErrInvalidMediaType,
		},
		{
			name:    "with cover",
			media:   &Media{Name: "With Cover", Cover: &cover, Type: MediaTypeSeries},
			wantErr: nil,
		},
		{
			name:    "series type valid",
			media:   &Media{Name: "Series", Type: MediaTypeSeries},
			wantErr: nil,
		},
		{
			name:    "music album type valid",
			media:   &Media{Name: "Album", Type: MediaTypeMusicAlbum},
			wantErr: nil,
		},
		{
			name:    "book type valid",
			media:   &Media{Name: "Book", Type: category},
			wantErr: nil,
		},
		{
			name:    "empty name with valid type",
			media:   &Media{Name: emptyName, Type: MediaTypeMovie},
			wantErr: ErrEmptyName,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.media.Validate()
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("Media.Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("Media.Validate() unexpected error = %v", err)
			}
		})
	}
}
