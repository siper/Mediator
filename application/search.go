package application

import (
	"strconv"
	"strings"
	"unicode"

	"stersh.ru/mediator/domain"
)

func tokenize(s string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, tok := range tokenizeOrdered(s) {
		out[tok] = struct{}{}
	}
	return out
}

func tokenizeOrdered(s string) []string {
	out := make([]string, 0, 8)
	var cur strings.Builder
	flush := func() {
		if cur.Len() >= 2 {
			out = append(out, strings.ToLower(cur.String()))
		}
		cur.Reset()
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cur.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return out
}

var titleStopwords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "of": {}, "the": {},
	"le": {}, "la": {}, "les": {}, "el": {}, "los": {}, "las": {},
	"der": {}, "die": {}, "das": {}, "und": {},
}

var releaseNoiseTokens = map[string]struct{}{
	"1080p": {}, "2160p": {}, "720p": {}, "480p": {}, "4320p": {},
	"4k": {}, "uhd": {}, "hdr": {}, "hdr10": {}, "dv": {}, "dolby": {}, "atmos": {},
	"bluray": {}, "blu": {}, "ray": {}, "bdrip": {}, "brrip": {}, "remux": {},
	"webdl": {}, "webrip": {}, "web": {}, "hdtv": {}, "dvdrip": {}, "dvdscr": {},
	"cam": {}, "telecine": {}, "screener": {}, "scr": {}, "dl": {}, "rip": {},
	"x264": {}, "h264": {}, "x265": {}, "h265": {}, "hevc": {}, "av1": {}, "xvid": {}, "divx": {},
	"dts": {}, "aac": {}, "ac3": {}, "eac3": {}, "truehd": {}, "flac": {}, "mp3": {},
	"repack": {}, "proper": {}, "rerip": {}, "internal": {}, "limited": {},
	"multi": {}, "dual": {}, "audio": {}, "subs": {}, "sub": {}, "dub": {}, "dubbed": {},
	"vo": {}, "vf": {}, "vostfr": {},
	"russian": {}, "english": {}, "french": {}, "german": {}, "spanish": {}, "italian": {},
	"japanese": {}, "korean": {}, "chinese": {}, "portuguese": {}, "hindi": {},
	"rus": {}, "eng": {}, "fre": {}, "ger": {}, "spa": {}, "ita": {}, "jpn": {}, "kor": {},
	"complete": {}, "full": {}, "extended": {}, "unrated": {}, "directors": {}, "cut": {},
	"theatrical": {}, "hybrid": {}, "remastered": {},
}

func isYearToken(tok string) bool {
	if len(tok) != 4 {
		return false
	}
	y, err := strconv.Atoi(tok)
	if err != nil {
		return false
	}
	return y >= 1900 && y <= 2039
}

func isSeasonEpisodeToken(tok string) bool {
	if strings.HasPrefix(tok, "season") {
		rest := strings.TrimPrefix(tok, "season")
		if rest == "" {
			return true
		}
		_, err := strconv.Atoi(rest)
		return err == nil
	}
	if tok[0] == 's' || tok[0] == 'e' {
		rest := tok[1:]
		if rest == "" {
			return false
		}
		allDigits := true
		for _, r := range rest {
			if r < '0' || r > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return true
		}
	}
	if strings.Contains(tok, "x") {
		parts := strings.SplitN(tok, "x", 2)
		if len(parts) == 2 {
			_, err1 := strconv.Atoi(parts[0])
			_, err2 := strconv.Atoi(parts[1])
			return err1 == nil && err2 == nil
		}
	}
	if strings.HasPrefix(tok, "s") && strings.Contains(tok, "e") {
		i := strings.Index(tok, "e")
		if i > 1 {
			_, err1 := strconv.Atoi(tok[1:i])
			_, err2 := strconv.Atoi(tok[i+1:])
			return err1 == nil && err2 == nil
		}
	}
	return false
}

func significantQueryTokens(tokens map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{}, len(tokens))
	for tok := range tokens {
		if _, ok := titleStopwords[tok]; ok {
			continue
		}
		if _, ok := releaseNoiseTokens[tok]; ok {
			continue
		}
		if isYearToken(tok) {
			continue
		}
		out[tok] = struct{}{}
	}
	return out
}

func releaseTitleCore(title string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, tok := range tokenizeOrdered(title) {
		if isYearToken(tok) || isSeasonEpisodeToken(tok) {
			break
		}
		if _, ok := releaseNoiseTokens[tok]; ok {
			break
		}
		if _, ok := titleStopwords[tok]; ok {
			continue
		}
		out[tok] = struct{}{}
	}
	return out
}

const (
	tierExact = 0
	tierFuzzy = 1
	tierDrop  = -1
)

func matchScore(query, title string) (tier int, score float64) {
	qt := tokenize(query)
	if len(qt) == 0 {
		return tierExact, 1.0
	}
	tt := tokenize(title)
	if len(tt) == 0 {
		return tierDrop, 0
	}

	qSig := significantQueryTokens(qt)
	if len(qSig) == 0 {
		qSig = qt
	}
	tCore := releaseTitleCore(title)
	if len(tCore) == 0 {
		tCore = significantQueryTokens(tt)
		if len(tCore) == 0 {
			tCore = tt
		}
	}

	common := 0
	for w := range qSig {
		if _, ok := tCore[w]; ok {
			common++
		}
	}
	if common == len(qSig) {
		extra := len(tCore) - common
		coverage := float64(common) / float64(len(tCore))
		if extra <= 1 || coverage >= 0.5 {
			return tierExact, 1.0
		}
	}
	union := len(qSig) + len(tCore) - common
	if union == 0 {
		return tierDrop, 0
	}
	jaccard := float64(common) / float64(union)
	if jaccard >= 0.4 && common >= 2 {
		return tierFuzzy, jaccard
	}
	return tierDrop, 0
}

func sharesScript(a, b string) bool {
	return (hasScript(a, unicode.Latin) && hasScript(b, unicode.Latin)) ||
		(hasScript(a, unicode.Cyrillic) && hasScript(b, unicode.Cyrillic))
}

func hasScript(s string, rt *unicode.RangeTable) bool {
	for _, r := range s {
		if unicode.Is(rt, r) {
			return true
		}
	}
	return false
}

func dedupResults(results []domain.SearchResult) []domain.SearchResult {
	seen := make(map[string]struct{})
	out := make([]domain.SearchResult, 0, len(results))
	for _, r := range results {
		key := r.ProviderName + ":" + r.ExternalID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, r)
	}
	return out
}
