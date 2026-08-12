package authortoday

type chapterInfo struct {
	ID          int64  `json:"id"`
	IsAvailable bool   `json:"isAvailable"`
	Title       string `json:"title"`
	SortOrder   int    `json:"sortOrder"`
	TextLength  int    `json:"textLength"`
}

type workDetails struct {
	ID                   int64         `json:"id"`
	Title                string        `json:"title"`
	Annotation           string        `json:"annotation"`
	CoverURL             string        `json:"coverUrl"`
	AuthorFIO            string        `json:"authorFIO"`
	CoAuthorFIO          string        `json:"coAuthorFIO"`
	IsFinished           bool          `json:"isFinished"`
	IsPurchased          bool          `json:"isPurchased"`
	AllowDownloads       bool          `json:"allowDownloads"`
	LastModificationTime string        `json:"lastModificationTime"`
	SeriesTitle          string        `json:"seriesTitle"`
	SeriesOrder          int           `json:"seriesOrder"`
	Chapters             []chapterInfo `json:"chapters"`
}

type scrapeResult struct {
	WorkID     int64
	Title      string
	Annotation string
	CoverURL   string
	AuthorID   string
	AuthorName string
	Completed  bool
	SeriesID   int64
	Series     string
}
