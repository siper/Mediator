package openlibrary

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"stersh.ru/mediator/domain"
)

func newTestProvider(t *testing.T, baseURL string) *OpenLibraryProvider {
	t.Helper()
	return &OpenLibraryProvider{
		http:       http.DefaultClient,
		baseURL:    baseURL,
		authorBase: baseURL,
	}
}

func TestSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"numFound": 1,
			"docs": [{
				"key": "/works/OL27448W",
				"title": "The Lord of the Rings",
				"author_name": ["J.R.R. Tolkien"],
				"cover_i": 14625765,
				"first_publish_year": 1954
			}]
		}`))
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	results, _, err := p.Search("the lord of the rings", nil, 1, 25)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.ExternalID != "OL27448W" {
		t.Errorf("ExternalID = %q, want OL27448W", r.ExternalID)
	}
	if r.Title != "J.R.R. Tolkien — The Lord of the Rings" {
		t.Errorf("Title = %q", r.Title)
	}
	if r.MediaType != domain.MediaTypeBook {
		t.Errorf("MediaType = %d, want %d", r.MediaType, domain.MediaTypeBook)
	}
	if r.ProviderName != Name {
		t.Errorf("ProviderName = %q, want %q", r.ProviderName, Name)
	}
	if r.CoverURL != "https://covers.openlibrary.org/b/id/14625765-M.jpg" {
		t.Errorf("CoverURL = %q", r.CoverURL)
	}
	if r.Year == nil || *r.Year != 1954 {
		t.Errorf("Year = %v, want 1954", r.Year)
	}
}

func TestSearch_EmptyResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"numFound":0,"docs":[]}`))
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	res, _, err := p.Search("nope", nil, 1, 25)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(res) != 0 {
		t.Fatalf("expected 0, got %d", len(res))
	}
}

func TestSearch_NoTitleAuthor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"docs":[{"key":"/works/OL1W","title":"Untitled"}]}`))
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	res, _, err := p.Search("untitled", nil, 1, 25)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1, got %d", len(res))
	}
	if res[0].Title != "Untitled" {
		t.Errorf("Title = %q, want bare title without author", res[0].Title)
	}
	if res[0].CoverURL != "" {
		t.Errorf("CoverURL = %q, want empty", res[0].CoverURL)
	}
}

func TestSearch_ShortQueryReturnsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be called for short query")
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	for _, q := range []string{"", "a", "ha"} {
		res, _, err := p.Search(q, nil, 1, 25)
		if err != nil {
			t.Fatalf("Search(%q) err: %v", q, err)
		}
		if len(res) != 0 {
			t.Fatalf("Search(%q) = %d results, want 0", q, len(res))
		}
	}
}

func TestSearch_RejectsNonBookType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be called for non-book media type")
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	mt := domain.MediaTypeMovie
	res, _, err := p.Search("anything", &mt, 1, 25)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(res) != 0 {
		t.Fatalf("expected 0, got %d", len(res))
	}
}

func TestGetMedia(t *testing.T) {
	workSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/works/OL27448W.json":
			_, _ = w.Write([]byte(`{
				"title": "The Lord of the Rings",
				"authors": [{"author": {"key": "/authors/OL26320A"}}],
				"covers": [14625765, 11658206],
				"description": {"type":"/type/text","value":"Epic fantasy."},
				"last_modified": {"type":"/type/datetime","value":"2026-04-09T06:22:12.405370"}
			}`))
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer workSrv.Close()

	authorSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name": "J.R.R. Tolkien"}`))
	}))
	defer authorSrv.Close()

	p := newTestProvider(t, workSrv.URL)
	p.authorBase = authorSrv.URL

	pm, err := p.GetMedia("OL27448W", domain.MediaTypeBook)
	if err != nil {
		t.Fatalf("GetMedia failed: %v", err)
	}
	if pm.ExternalID != "OL27448W" {
		t.Errorf("ExternalID = %q", pm.ExternalID)
	}
	if pm.Title != "J.R.R. Tolkien — The Lord of the Rings" {
		t.Errorf("Title = %q", pm.Title)
	}
	if pm.MediaType != domain.MediaTypeBook {
		t.Errorf("MediaType mismatch")
	}
	if pm.Status != domain.MediaStatusCompleted {
		t.Errorf("Status = %q", pm.Status)
	}
	if pm.LastModified != "2026-04-09T06:22:12.405370" {
		t.Errorf("LastModified = %q", pm.LastModified)
	}
	if pm.CoverURL != "https://covers.openlibrary.org/b/id/14625765-M.jpg" {
		t.Errorf("CoverURL = %q", pm.CoverURL)
	}
	if pm.Overview != "Epic fantasy." {
		t.Errorf("Overview = %q", pm.Overview)
	}
	if len(pm.Groups) != 0 || len(pm.Parts) != 0 {
		t.Errorf("expected no groups/parts for book")
	}
}

func TestGetMedia_NoCoversNoDescription(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"title": "Apocalipsis",
			"authors": [{"author": {"key": "/authors/OL42359A"}}],
			"last_modified": {"type":"/type/datetime","value":"2026-08-02T05:31:26"}
		}`))
	}))
	defer srv.Close()

	authorSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name": "Pedro Páramo"}`))
	}))
	defer authorSrv.Close()

	p := newTestProvider(t, srv.URL)
	p.authorBase = authorSrv.URL

	pm, err := p.GetMedia("OL579495W", domain.MediaTypeBook)
	if err != nil {
		t.Fatalf("GetMedia failed: %v", err)
	}
	if pm.CoverURL != "" {
		t.Errorf("CoverURL = %q, want empty", pm.CoverURL)
	}
	if pm.Overview != "" {
		t.Errorf("Overview = %q, want empty", pm.Overview)
	}
}

func TestGetMedia_StringDescription(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"title": "Harry Potter and the Cursed Child",
			"authors": [{"author": {"key": "/authors/OL1A"}}],
			"covers": [8763851],
			"description": "The Eighth Story. Nineteen Years Later.",
			"last_modified": {"type":"/type/datetime","value":"2025-02-19T00:18:41.722591"}
		}`))
	}))
	defer srv.Close()

	authorSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name": "J.K. Rowling"}`))
	}))
	defer authorSrv.Close()

	p := newTestProvider(t, srv.URL)
	p.authorBase = authorSrv.URL

	pm, err := p.GetMedia("OL17360811W", domain.MediaTypeBook)
	if err != nil {
		t.Fatalf("GetMedia failed on string-typed description: %v", err)
	}
	if pm.Title != "J.K. Rowling — Harry Potter and the Cursed Child" {
		t.Errorf("Title = %q", pm.Title)
	}
	if pm.Overview != "The Eighth Story. Nineteen Years Later." {
		t.Errorf("Overview = %q", pm.Overview)
	}
	if pm.CoverURL != "https://covers.openlibrary.org/b/id/8763851-M.jpg" {
		t.Errorf("CoverURL = %q", pm.CoverURL)
	}
}

func TestOlTypedText_Unmarshal(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantVal string
		wantTyp string
	}{
		{"string", `"plain text"`, "plain text", ""},
		{"object", `{"type":"/type/text","value":"obj text"}`, "obj text", "/type/text"},
		{"empty_object", `{}`, "", ""},
		{"null", `null`, "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var v olTypedText
			if err := json.Unmarshal([]byte(c.input), &v); err != nil {
				t.Fatalf("unmarshal err: %v", err)
			}
			if v.Value != c.wantVal {
				t.Errorf("Value = %q, want %q", v.Value, c.wantVal)
			}
			if v.Type != c.wantTyp {
				t.Errorf("Type = %q, want %q", v.Type, c.wantTyp)
			}
		})
	}
}

func TestGetMedia_RejectsNonBookType(t *testing.T) {
	p := newTestProvider(t, "http://unused")
	_, err := p.GetMedia("OL1W", domain.MediaTypeMovie)
	if err == nil {
		t.Fatal("expected error for non-book type")
	}
}

func TestFetchLastModified(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"title": "X",
			"authors": [{"author": {"key": "/authors/OL1A"}}],
			"last_modified": {"type":"/type/datetime","value":"2024-01-02T03:04:05Z"}
		}`))
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	got, err := p.FetchLastModified(context.Background(), "OL1W", domain.MediaTypeBook)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != "2024-01-02T03:04:05Z" {
		t.Errorf("got %q", got)
	}
}

func TestFetchLastModified_NonBookReturnsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("should not call server for non-book type")
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	got, err := p.FetchLastModified(context.Background(), "OL1W", domain.MediaTypeMovie)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestWorkKeyToOLID(t *testing.T) {
	cases := []struct {
		key  string
		want string
	}{
		{"/works/OL27448W", "OL27448W"},
		{"/authors/OL26320A", ""},
		{"", ""},
		{"OL27448W", ""},
	}
	for _, c := range cases {
		got := workKeyToOLID(c.key)
		if got != c.want {
			t.Errorf("workKeyToOLID(%q) = %q, want %q", c.key, got, c.want)
		}
	}
}

func TestMakeCoverURL(t *testing.T) {
	if u := makeCoverURL(14625765); u != "https://covers.openlibrary.org/b/id/14625765-M.jpg" {
		t.Errorf("got %q", u)
	}
	if u := makeCoverURL(0); u != "" {
		t.Errorf("got %q, want empty", u)
	}
	if u := makeCoverURL(-1); u != "" {
		t.Errorf("got %q, want empty", u)
	}
}

func TestBuildBookTitle(t *testing.T) {
	cases := []struct {
		authors []string
		title   string
		want    string
	}{
		{[]string{"Tolkien"}, "Rings", "Tolkien — Rings"},
		{nil, "Rings", "Rings"},
		{[]string{}, "Rings", "Rings"},
		{[]string{""}, "Rings", "Rings"},
	}
	for _, c := range cases {
		got := buildBookTitle(c.authors, c.title)
		if got != c.want {
			t.Errorf("buildBookTitle(%v, %q) = %q, want %q", c.authors, c.title, got, c.want)
		}
	}
}
