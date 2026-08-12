package authortoday

import (
	"strings"
	"testing"
)

func TestBuildFB2_ContainsTitleAuthorAndChapters(t *testing.T) {
	b := Book{
		Title:      "Test Book",
		Annotation: "An annotation",
		Authors:    []string{"Ivan Petrov"},
		Series:     &Series{Title: "Cycle", Number: 2},
		Chapters: []Chapter{
			{Title: "Chapter One", HTML: "<p>Hello <b>world</b></p><br><p>line2</p>"},
		},
	}
	out := BuildFB2(b)

	checks := []string{
		`<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0"`,
		"<book-title><p>Test Book</p></book-title>",
		"<first-name>Ivan</first-name>",
		"<last-name>Petrov</last-name>",
		"<annotation>An annotation</annotation>",
		`<sequence name="Cycle" number="2"/>`,
		"<section><title><p>Chapter One</p></title>",
		"<strong>world</strong>",
		"<empty-line/>",
		"</FictionBook>",
	}
	for _, c := range checks {
		if !strings.Contains(out, c) {
			t.Errorf("missing %q in output:\n%s", c, out)
		}
	}
}

func TestBuildFB2_EscapesXMLSpecialChars(t *testing.T) {
	b := Book{Title: "A & B <C>", Chapters: nil}
	out := BuildFB2(b)
	if !strings.Contains(out, "A &amp; B &lt;C&gt;") {
		t.Errorf("xml escaping failed:\n%s", out)
	}
}
