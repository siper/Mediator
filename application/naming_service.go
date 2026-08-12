package application

import (
	"fmt"
	"path/filepath"
	"strings"
)

type NamingService struct{}

func NewNamingService() *NamingService { return &NamingService{} }

func (n *NamingService) Build(rootPath, folderName, mediaName, quality, ext string) string {
	folderBase := folderName
	if folderBase == "" {
		folderBase = mediaName
	}
	folderBase = sanitize(folderBase)
	base := sanitize(mediaName)
	folder := filepath.Join(rootPath, folderBase)
	name := base
	if quality != "" {
		name = base + " - " + quality
	}
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return filepath.Join(folder, name+ext)
}

func (n *NamingService) BuildEpisode(rootPath, folderName, seriesName string, season, episode int, episodeName *string, seasonFormat, quality, ext string) string {
	folderBase := folderName
	if folderBase == "" {
		folderBase = seriesName
	}
	folderBase = sanitize(folderBase)
	base := sanitize(seriesName)
	seasonDir := formatSeason(seasonFormat, season)
	ep := fmt.Sprintf("%s S%02dE%02d", base, season, episode)
	if episodeName != nil && *episodeName != "" {
		ep = ep + " - " + sanitize(*episodeName)
	}
	if quality != "" {
		ep = ep + " - " + quality
	}
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return filepath.Join(rootPath, folderBase, seasonDir, ep+ext)
}

func (n *NamingService) BuildAlbum(rootPath, artist, album string, trackNo int, title, quality, ext string) string {
	artist = sanitize(artist)
	album = sanitize(album)
	trackTitle := sanitize(title)
	name := fmt.Sprintf("%02d - %s", trackNo, trackTitle)
	if quality != "" {
		name = name + " - " + quality
	}
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	if artist != "" {
		return filepath.Join(rootPath, artist, album, name+ext)
	}
	return filepath.Join(rootPath, album, name+ext)
}

func formatSeason(seasonFormat string, season int) string {
	if seasonFormat == "" {
		seasonFormat = "Season %d"
	}
	if strings.Contains(seasonFormat, "%") {
		return sanitize(fmt.Sprintf(seasonFormat, season))
	}
	return sanitize(seasonFormat)
}

var unsafeReplacer = strings.NewReplacer(
	"/", " ", "\\", " ", ":", " ", "*", " ",
	"?", " ", "\"", " ", "<", " ", ">", " ", "|", " ",
)

func sanitize(s string) string {
	s = unsafeReplacer.Replace(s)
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if r == ' ' {
			if !prevSpace {
				b.WriteRune(r)
			}
			prevSpace = true
			continue
		}
		b.WriteRune(r)
		prevSpace = false
	}
	return strings.TrimSpace(b.String())
}
