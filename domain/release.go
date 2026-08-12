package domain

import "time"

type Release struct {
	Title       string
	DownloadURL string
	MagnetURI   string
	InfoHash    string
	Size        int64
	Seeders     int
	Indexer     string
	PublishDate time.Time
}
