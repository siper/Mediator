package application

import (
	"testing"

	"stersh.ru/mediator/domain"
)

func TestMatchScore(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		title     string
		wantTier  int
		wantDrop  bool
		wantFuzzy bool
	}{
		{
			name:     "reported garbage case dropped",
			query:    "Человек паук: новый день",
			title:    "Прощай жизнь дракона. Здравствуй жизнь человека",
			wantDrop: true,
		},
		{
			name:     "exact and-match with punctuation",
			query:    "Человек паук",
			title:    "Человек-паук: Новый день",
			wantTier: tierExact,
		},
		{
			name:     "case insensitive",
			query:    "человек паук",
			title:    "ЧЕЛОВЕК ПАУК",
			wantTier: tierExact,
		},
		{
			name:     "single token exact",
			query:    "Dune",
			title:    "Dune (2021)",
			wantTier: tierExact,
		},
		{
			name:      "fuzzy fallback when query has extra token",
			query:     "новый человек паук",
			title:     "Человек-паук",
			wantFuzzy: true,
		},
		{
			name:     "empty query not filtered",
			query:    "",
			title:    "Anything",
			wantTier: tierExact,
		},
		{
			name:     "query only punctuation not filtered",
			query:    ": -- ",
			title:    "Anything",
			wantTier: tierExact,
		},
		{
			name:     "empty title dropped",
			query:    "dune",
			title:    "???",
			wantDrop: true,
		},
		{
			name:     "common 1 below threshold dropped",
			query:    "Дюна фильм",
			title:    "Дюна",
			wantDrop: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tier, score := matchScore(tt.query, tt.title)
			if tt.wantDrop {
				if tier != tierDrop {
					t.Fatalf("expected drop, got tier=%d score=%v", tier, score)
				}
				return
			}
			if tt.wantFuzzy && tier != tierFuzzy {
				t.Fatalf("expected fuzzy, got tier=%d score=%v", tier, score)
			}
			if !tt.wantFuzzy && tt.wantTier != tier {
				t.Fatalf("expected tier=%d, got tier=%d score=%v", tt.wantTier, tier, score)
			}
		})
	}
}

func TestDedupResults(t *testing.T) {
	results := []domain.SearchResult{
		{ProviderName: "tmdb", ExternalID: "1", Title: "Человек-паук: Новый день"},
		{ProviderName: "tmdb", ExternalID: "1", Title: "Человек-паук: Новый день"},
		{ProviderName: "author_today", ExternalID: "9", Title: "Прощай жизнь дракона. Здравствуй жизнь человека"},
		{ProviderName: "author_today", ExternalID: "2", Title: "новый человек паук"},
	}
	got := dedupResults(results)
	if len(got) != 3 {
		t.Fatalf("expected 3 results after dedup, got %d: %+v", len(got), got)
	}
	if got[0].ExternalID != "1" || got[1].ExternalID != "9" || got[2].ExternalID != "2" {
		t.Fatalf("expected order preserved after dedup, got %+v", got)
	}
}

func TestDedupResults_CrossScriptKept(t *testing.T) {
	results := []domain.SearchResult{
		{ProviderName: "openlibrary", ExternalID: "OL82563W", Title: "J. K. Rowling — Harry Potter and the Philosopher's Stone"},
		{ProviderName: "openlibrary", ExternalID: "OL82563W", Title: "J. K. Rowling — Harry Potter and the Philosopher's Stone"},
	}
	got := dedupResults(results)
	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %d: %+v", len(got), got)
	}
	if got[0].ExternalID != "OL82563W" {
		t.Fatalf("expected OL82563W, got %q", got[0].ExternalID)
	}
}

func TestDedupResults_EmptyInput(t *testing.T) {
	got := dedupResults(nil)
	if len(got) != 0 {
		t.Fatalf("expected empty, got %d", len(got))
	}
}
