package authortoday

import (
	"log/slog"
	"strings"
)

type Book struct {
	Title      string
	Annotation string
	Authors    []string
	Series     *Series
	Chapters   []Chapter
}

type Series struct {
	Title  string
	Number int
}

type Chapter struct {
	Title string
	HTML  string
}

func BuildFB2(b Book) string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	sb.WriteString(`<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:xlink="http://www.w3.org/1999/xlink">` + "\n")

	sb.WriteString("<description>\n<title-info>\n")
	for _, author := range b.Authors {
		writeAuthor(&sb, author)
	}
	if b.Annotation != "" {
		sb.WriteString("<annotation>")
		sb.WriteString(xmlText(clearText(b.Annotation)))
		sb.WriteString("</annotation>\n")
	}
	sb.WriteString("<book-title><p>")
	sb.WriteString(xmlText(b.Title))
	sb.WriteString("</p></book-title>\n")
	if b.Series != nil && b.Series.Title != "" {
		sb.WriteString(`<sequence name="`)
		sb.WriteString(xmlAttr(b.Series.Title))
		sb.WriteString(`" number="`)
		sb.WriteString(itoa(b.Series.Number))
		sb.WriteString("\"/>\n")
	}
	sb.WriteString("</title-info>\n</description>\n")

	sb.WriteString("<body>\n<title><p>")
	sb.WriteString(xmlText(b.Title))
	sb.WriteString("</p></title>\n")
	for i, ch := range b.Chapters {
		slog.Debug("fb2 build: chapter", "index", i, "title", ch.Title, "html_len", len(ch.HTML))
		sb.WriteString("<section><title><p>")
		sb.WriteString(xmlText(ch.Title))
		sb.WriteString("</p></title>")
		cleared := clearText(ch.HTML)
		slog.Debug("fb2 build: chapter cleared", "index", i, "title", ch.Title, "cleared_len", len(cleared))
		sb.WriteString(cleared)
		sb.WriteString("</section>\n")
	}
	sb.WriteString("</body>\n")

	sb.WriteString("</FictionBook>\n")
	return sb.String()
}

func writeAuthor(sb *strings.Builder, name string) {
	parts := strings.Fields(name)
	sb.WriteString("<author>")
	switch len(parts) {
	case 1:
		sb.WriteString("<first-name>" + xmlText(parts[0]) + "</first-name>")
	case 2:
		sb.WriteString("<first-name>" + xmlText(parts[0]) + "</first-name>")
		sb.WriteString("<last-name>" + xmlText(parts[1]) + "</last-name>")
	default:
		if len(parts) >= 3 {
			sb.WriteString("<first-name>" + xmlText(parts[0]) + "</first-name>")
			sb.WriteString("<middle-name>" + xmlText(parts[1]) + "</middle-name>")
			sb.WriteString("<last-name>" + xmlText(strings.Join(parts[2:], " ")) + "</last-name>")
		} else {
			sb.WriteString("<nickname>" + xmlText(name) + "</nickname>")
		}
	}
	sb.WriteString("</author>\n")
}

var tagReplacer = strings.NewReplacer(
	"<b>", "<strong>", "</b>", "</strong>",
	"<i>", "<emphasis>", "</i>", "</emphasis>",
	"<em>", "<emphasis>", "</em>", "</emphasis>",
	"<del>", "<strikethrough>", "</del>", "</strikethrough>",
	"<strikethrough>", "<strikethrough>", "</strikethrough>", "</strikethrough>",
	"<blockquote>", "<cite>", "</blockquote>", "</cite>",
	"<br>", "<empty-line/>", "<br/>", "<empty-line/>", "<br />", "<empty-line/>",
)

func clearText(text string) string {
	return tagReplacer.Replace(text)
}

func xmlText(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch r {
		case '<':
			sb.WriteString("&lt;")
		case '>':
			sb.WriteString("&gt;")
		case '&':
			sb.WriteString("&amp;")
		case '"':
			sb.WriteString("&quot;")
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func xmlAttr(s string) string {
	return xmlText(s)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var digits [20]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		digits[i] = '-'
	}
	return string(digits[i:])
}
