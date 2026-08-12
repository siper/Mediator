package application

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"stersh.ru/mediator/domain"
)

type ReleaseParser struct{}

func NewReleaseParser() *ReleaseParser { return &ReleaseParser{} }

var (
	reResolution = regexp.MustCompile(`(?i)\b(480|720|1080|2160|4320)p\b`)
	re4K         = regexp.MustCompile(`(?i)\b(4k|uhd)\b`)
	reSource     = regexp.MustCompile(`(?i)\b(web-?dl|webdl|webrip|web\.rip|bluray|blu-?ray|br-?rip|bdrip|hdtv|dvdrip|dvdscr|cam|ts|tc|telecine|scr|screener)\b`)
	reCodec      = regexp.MustCompile(`(?i)\b(x264|h\.?264|x265|h\.?265|hevc|xvid|divx|av1)\b`)
	reYear       = regexp.MustCompile(`(?i)\b(19\d{2}|20[0-3]\d)\b`)
	reRepack     = regexp.MustCompile(`(?i)\b(repack|proper|rerip)\b`)

	reSeason          = regexp.MustCompile(`(?i)S(\d{1,2})(?:\D|$)`)
	reSeasonWord      = regexp.MustCompile(`(?i)season[._\s-]?(\d{1,2})(?:\D|$)`)
	reSeasonRangeEp = regexp.MustCompile(`(?i)S(\d{1,2})-S?(\d{1,2})E(\d{1,3})-E?(\d{1,3})`)
	reSeasonRange     = regexp.MustCompile(`(?i)S(\d{1,2})-S?(\d{1,2})(?:\D|$)`)
	reSeasonWordRange = regexp.MustCompile(`(?i)season[._\s-]?(\d{1,2})-(\d{1,2})(?:\D|$)`)
	reEpRange         = regexp.MustCompile(`(?i)E(\d{1,3})-E?(\d{1,3})(?:\D|$)`)
	reEpisode         = regexp.MustCompile(`(?i)E(\d{1,3})`)
	reXFormat         = regexp.MustCompile(`(?i)(\d{1,2})x(\d{1,3})(?:\D|$)`)
	reComplete        = regexp.MustCompile(`(?i)\b(complete|full)\b`)
)

func (p *ReleaseParser) Parse(title string, t domain.MediaType) (domain.ParsedRelease, error) {
	if title == "" {
		return domain.ParsedRelease{}, fmt.Errorf("empty release title")
	}
	pr := domain.ParsedRelease{Group: extractGroup(title)}

	kind, _ := domain.QualityKindFor(t)
	switch kind {
	case domain.QualityKindVideoMovie, domain.QualityKindVideoSeries:
		p.parseVideo(title, &pr)
		pr.Quality = videoQuality(kind, pr.Resolution, pr.Source)
		if t == domain.MediaTypeSeries {
			parseSeasonEpisode(title, &pr)
		}
	case domain.QualityKindBook, domain.QualityKindAudio:
		pr.Quality = detectFormat(title, kind)
	}

	return pr, nil
}

func (p *ReleaseParser) parseVideo(title string, pr *domain.ParsedRelease) {
	pr.Resolution = resolutionOf(title)
	if m := reSource.FindStringSubmatch(title); m != nil {
		pr.Source = normalizeSource(m[1])
	}
	if m := reCodec.FindStringSubmatch(title); m != nil {
		pr.Codec = strings.ToLower(strings.ReplaceAll(m[1], ".", ""))
	}
	if m := reYear.FindStringSubmatch(title); m != nil {
		if y, err := strconv.Atoi(m[1]); err == nil {
			pr.Year = &y
		}
	}
	if reRepack.MatchString(title) {
		pr.IsRepack = true
	}
}

func videoQuality(kind domain.QualityKind, resolution, source string) domain.Quality {
	name := compositeVideoName(resolution, source)
	if name == "" {
		return domain.Quality{}
	}
	q := domain.Quality{Kind: kind, Name: name}
	if !q.Valid() {
		return domain.Quality{}
	}
	return q
}

func compositeVideoName(resolution, source string) string {
	switch source {
	case "TELECINE":
		source = "TC"
	case "SCREENER", "DVDSCR":
		source = "SCR"
	}
	switch source {
	case "CAM", "TS", "TC", "SCR", "DVD":
		return source
	}
	if resolution == "" {
		return ""
	}
	if source == "" {
		return resolution
	}
	return resolution + " " + source
}

func resolutionOf(title string) string {
	if m := reResolution.FindStringSubmatch(title); m != nil {
		return strings.ToLower(m[1]) + "p"
	}
	if re4K.MatchString(title) {
		return "2160p"
	}
	return ""
}

func normalizeSource(raw string) string {
	lower := strings.ToLower(strings.ReplaceAll(raw, "-", ""))
	switch lower {
	case "webdl", "webrip":
		if strings.Contains(strings.ToLower(raw), "rip") {
			return "WEBRip"
		}
		return "WEB-DL"
	case "bluray", "brrip", "bdrip":
		return "BluRay"
	case "hdtv":
		return "HDTV"
	case "dvdrip":
		return "DVD"
	default:
		return strings.ToUpper(lower)
	}
}

var bookFormats = domain.Qualities(domain.QualityKindBook)
var audioFormats = domain.Qualities(domain.QualityKindAudio)

func detectFormat(title string, kind domain.QualityKind) domain.Quality {
	catalog := domain.Qualities(kind)
	lower := strings.ToLower(title)
	tokens := strings.FieldsFunc(lower, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	})
	rank := make(map[string]int, len(catalog))
	for i, q := range catalog {
		rank[q.Name] = i
	}
	best := -1
	var bestName string
	for _, tok := range tokens {
		if r, ok := rank[tok]; ok && r > best {
			best = r
			bestName = tok
		}
	}
	if bestName == "" {
		return domain.Quality{}
	}
	return domain.Quality{Kind: kind, Name: bestName}
}

func extractGroup(title string) string {
	s := title
	if i := strings.LastIndex(s, "."); i != -1 {
		ext := s[i+1:]
		if len(ext) <= 4 && isAlpha(ext) {
			s = s[:i]
		}
	}
	if i := strings.LastIndex(s, "-"); i != -1 {
		return strings.TrimSpace(s[i+1:])
	}
	return ""
}

func isAlpha(s string) bool {
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
			return false
		}
	}
	return s != ""
}

func parseSeasonEpisode(title string, pr *domain.ParsedRelease) {
	if m := reXFormat.FindStringSubmatch(title); m != nil {
		season, _ := strconv.Atoi(m[1])
		ep, _ := strconv.Atoi(m[2])
		pr.Season = &season
		pr.Seasons = []int{season}
		pr.Episodes = []int{ep}
		return
	}

	if m := reSeasonRangeEp.FindStringSubmatch(title); m != nil {
		start, _ := strconv.Atoi(m[1])
		end, _ := strconv.Atoi(m[2])
		epStart, _ := strconv.Atoi(m[3])
		epEnd, _ := strconv.Atoi(m[4])
		pr.Season = &start
		pr.Seasons = seasonRange(start, end)
		for ep := epStart; ep <= epEnd; ep++ {
			pr.Episodes = append(pr.Episodes, ep)
		}
		return
	}

	if m := reSeasonRange.FindStringSubmatch(title); m != nil {
		start, _ := strconv.Atoi(m[1])
		end, _ := strconv.Atoi(m[2])
		pr.Season = &start
		pr.Seasons = seasonRange(start, end)
		return
	}
	if m := reSeasonWordRange.FindStringSubmatch(title); m != nil {
		start, _ := strconv.Atoi(m[1])
		end, _ := strconv.Atoi(m[2])
		pr.Season = &start
		pr.Seasons = seasonRange(start, end)
		return
	}

	season := -1
	if sm := reSeason.FindStringSubmatch(title); sm != nil {
		season, _ = strconv.Atoi(sm[1])
	} else if sm := reSeasonWord.FindStringSubmatch(title); sm != nil {
		season, _ = strconv.Atoi(sm[1])
	}
	if season >= 0 {
		pr.Season = &season
		pr.Seasons = []int{season}

		if m := reEpRange.FindStringSubmatch(title); m != nil {
			start, _ := strconv.Atoi(m[1])
			end, _ := strconv.Atoi(m[2])
			for ep := start; ep <= end; ep++ {
				pr.Episodes = append(pr.Episodes, ep)
			}
			return
		}
		for _, m := range reEpisode.FindAllStringSubmatch(title, -1) {
			ep, _ := strconv.Atoi(m[1])
			pr.Episodes = append(pr.Episodes, ep)
		}
		return
	}

	if reComplete.MatchString(title) {
		pr.Complete = true
	}
}

func seasonRange(start, end int) []int {
	if start > end {
		start, end = end, start
	}
	out := make([]int, 0, end-start+1)
	for s := start; s <= end; s++ {
		out = append(out, s)
	}
	return out
}
