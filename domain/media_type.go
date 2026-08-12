package domain

type MediaType uint8

const (
	MediaTypeBook MediaType = iota
	MediaTypeSeries
	MediaTypeMovie
	MediaTypeMusicAlbum
)

func (t MediaType) String() string {
	switch t {
	case MediaTypeBook:
		return "book"
	case MediaTypeSeries:
		return "series"
	case MediaTypeMovie:
		return "movie"
	case MediaTypeMusicAlbum:
		return "music_album"
	default:
		return "unknown"
	}
}
