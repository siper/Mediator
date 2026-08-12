package authortoday

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

const searchHTML = `
<html><body>
<div class="book-row">
  <img src="/static/covers/123.jpg"/>
  <div class="book-title"><a href="/work/12345">My Book</a></div>
  <div class="annotation"><p>An epic <em class='searched-item'>tale</em> of adventure and mystery.</p></div>
  <a href="/work/series/77">Cycle of Books</a>
  <div class="book-author"><a href="/u/jdoe/works?format=ebook">Jane Doe</a></div>
</div>
<div class="book-row">
  <img src="/static/covers/999.jpg"/>
  <div class="book-title"><a href="/work/999">Other Book</a></div>
  <div class="book-author"><a href="/u/smith/works?format=ebook">John Smith</a></div>
  <i class="icon-pencil book-status-icon"></i>
</div>
</body></html>
`

func TestParseBookRow(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(searchHTML))
	if err != nil {
		t.Fatal(err)
	}
	var results []scrapeResult
	doc.Find("div.book-row").Each(func(_ int, sel *goquery.Selection) {
		r, ok := parseBookRow(sel)
		if ok {
			results = append(results, r)
		}
	})

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	r := results[0]
	if r.WorkID != 12345 {
		t.Errorf("WorkID: got %d want 12345", r.WorkID)
	}
	if r.Title != "My Book" {
		t.Errorf("Title: got %q want %q", r.Title, "My Book")
	}
	if r.AuthorID != "jdoe" || r.AuthorName != "Jane Doe" {
		t.Errorf("Author: got id=%q name=%q", r.AuthorID, r.AuthorName)
	}
	if r.CoverURL != "https://author.today/static/covers/123.jpg" {
		t.Errorf("CoverURL: got %q", r.CoverURL)
	}
	if r.SeriesID != 77 || r.Series != "Cycle of Books" {
		t.Errorf("Series: got id=%d title=%q", r.SeriesID, r.Series)
	}
	if r.Annotation != "An epic tale of adventure and mystery." {
		t.Errorf("Annotation: got %q", r.Annotation)
	}
	if !r.Completed {
		t.Error("expected completed=true (no pencil icon)")
	}

	if results[1].Completed {
		t.Error("expected completed=false for second (pencil icon present)")
	}
}

func TestParseAuthorLink(t *testing.T) {
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(`<a href="/u/jdoe/works?format=ebook">Jane Doe</a>`))
	id, name := parseAuthorLink(doc.Find("a").First())
	if id != "jdoe" || name != "Jane Doe" {
		t.Errorf("got id=%q name=%q", id, name)
	}
}

func TestLastSegment(t *testing.T) {
	cases := map[string]string{
		"/work/123":  "123",
		"/work/123/": "123",
		"/u/x":       "x",
		"abc":        "abc",
	}
	for in, want := range cases {
		if got := lastSegment(in); got != want {
			t.Errorf("lastSegment(%q)=%q want %q", in, got, want)
		}
	}
}

func TestAbsCoverURL(t *testing.T) {
	if got := absCoverURL("/static/x.jpg"); got != "https://author.today/static/x.jpg" {
		t.Error(got)
	}
	if got := absCoverURL("https://cdn.example/x.jpg"); got != "https://cdn.example/x.jpg" {
		t.Error(got)
	}
	if got := absCoverURL(""); got != "" {
		t.Error(got)
	}
}
