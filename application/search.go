package application

import (
	"strings"
	"unicode"

	"stersh.ru/mediator/domain"
)

func tokenize(s string) map[string]struct{} {
	out := make(map[string]struct{})
	var cur strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cur.WriteRune(r)
		} else {
			if cur.Len() >= 2 {
				out[strings.ToLower(cur.String())] = struct{}{}
			}
			cur.Reset()
		}
	}
	if cur.Len() >= 2 {
		out[strings.ToLower(cur.String())] = struct{}{}
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
	common := 0
	for w := range qt {
		if _, ok := tt[w]; ok {
			common++
		}
	}
	if common == len(qt) {
		return tierExact, 1.0
	}
	union := len(qt) + len(tt) - common
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
