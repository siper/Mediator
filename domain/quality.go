package domain

type QualityKind string

const (
	QualityKindVideoMovie  QualityKind = "video_movie"
	QualityKindVideoSeries QualityKind = "video_series"
	QualityKindBook        QualityKind = "book"
	QualityKindAudio       QualityKind = "audio"
)

type Quality struct {
	Kind QualityKind
	Name string
}

var qualityCatalog = map[QualityKind][]string{
	QualityKindVideoMovie: {
		"CAM", "TS", "TC", "SCR",
		"DVD",
		"480p", "480p WEB-DL", "480p BluRay",
		"720p", "720p HDTV", "720p WEBRip", "720p WEB-DL", "720p BluRay",
		"1080p", "1080p HDTV", "1080p WEBRip", "1080p WEB-DL", "1080p BluRay",
		"2160p", "2160p WEBRip", "2160p WEB-DL", "2160p BluRay",
	},
	QualityKindVideoSeries: {
		"720p", "720p HDTV", "720p WEBRip", "720p WEB-DL", "720p BluRay",
		"1080p", "1080p HDTV", "1080p WEBRip", "1080p WEB-DL", "1080p BluRay",
		"2160p", "2160p WEBRip", "2160p WEB-DL", "2160p BluRay",
	},
	QualityKindBook:  {"txt", "pdf", "djvu", "fb2", "epub", "mobi", "azw3", "docx"},
	QualityKindAudio: {"mp3", "aac", "ogg", "opus", "alac", "flac", "wav"},
}

func QualityKindFor(t MediaType) (QualityKind, bool) {
	switch t {
	case MediaTypeMovie:
		return QualityKindVideoMovie, true
	case MediaTypeSeries:
		return QualityKindVideoSeries, true
	case MediaTypeBook:
		return QualityKindBook, true
	case MediaTypeMusicAlbum:
		return QualityKindAudio, true
	}
	return "", false
}

func Qualities(kind QualityKind) []Quality {
	names := qualityCatalog[kind]
	out := make([]Quality, 0, len(names))
	for _, n := range names {
		out = append(out, Quality{Kind: kind, Name: n})
	}
	return out
}

func (q Quality) Valid() bool {
	for _, n := range qualityCatalog[q.Kind] {
		if n == q.Name {
			return true
		}
	}
	return false
}

func (q Quality) Rank() int {
	for i, n := range qualityCatalog[q.Kind] {
		if n == q.Name {
			return i
		}
	}
	return -1
}

func (q Quality) AtLeast(cutoff Quality) bool {
	if q.Kind != cutoff.Kind {
		return false
	}
	return q.Rank() >= cutoff.Rank()
}
