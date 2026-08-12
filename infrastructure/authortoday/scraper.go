package authortoday

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"stersh.ru/mediator/domain"
	"stersh.ru/mediator/infrastructure/httpc"
)

type Scraper struct {
	http *http.Client
}

func NewScraper(proxy *domain.Proxy) *Scraper {
	return &Scraper{http: httpc.NewClient(30 * time.Second, proxy)}
}

func (s *Scraper) TestConnection(ctx context.Context) error {
	u := siteBase + "/search?q=test&category=works"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; media)")
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("author.today: HTTP %d", resp.StatusCode)
	}
	return nil
}

func (s *Scraper) SearchWorks(ctx context.Context, query string) ([]scrapeResult, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("category", "works")
	doc, err := s.fetchDoc(ctx, siteBase+"/search?"+params.Encode())
	if err != nil {
		return nil, err
	}
	var out []scrapeResult
	doc.Find("div.book-row").Each(func(_ int, sel *goquery.Selection) {
		r, ok := parseBookRow(sel)
		if ok {
			out = append(out, r)
		}
	})
	return out, nil
}

func (s *Scraper) GetWork(ctx context.Context, workID int64) (*scrapeResult, error) {
	doc, err := s.fetchDoc(ctx, siteBase+"/work/"+strconv.FormatInt(workID, 10)+"/")
	if err != nil {
		return nil, err
	}
	panel := doc.Find("div.panel-body").First()
	if panel.Length() == 0 {
		return nil, fmt.Errorf("author.today: work %d not found", workID)
	}

	cover, _ := panel.Find("img.cover-image").First().Attr("src")
	title := strings.TrimSpace(doc.Find("h1.book-title").First().Text())
	completed := doc.Find("span[class^='label label-success']").Length() > 0

	var seriesID int64
	var seriesTitle string
	seriesSel := doc.Find("a[href^='/work/series/']").First()
	if seriesSel.Length() > 0 {
		href, _ := seriesSel.Attr("href")
		if id, err := strconv.ParseInt(strings.TrimPrefix(strings.TrimPrefix(href, "/work/series/"), "/"), 10, 64); err == nil {
			seriesID = id
		}
		seriesTitle = strings.TrimSpace(seriesSel.Text())
	}

	authorSel := doc.Find("a[href^='/u/']").First()
	authorID, authorName := parseAuthorLink(authorSel)

	return &scrapeResult{
		WorkID:     workID,
		Title:      title,
		CoverURL:   absCoverURL(cover),
		AuthorID:   authorID,
		AuthorName: authorName,
		Completed:  completed,
		SeriesID:   seriesID,
		Series:     seriesTitle,
	}, nil
}

func (s *Scraper) fetchDoc(ctx context.Context, u string) (*goquery.Document, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; media)")
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("author.today scrape: HTTP %d for %s", resp.StatusCode, u)
	}
	return goquery.NewDocumentFromReader(strings.NewReader(string(body)))
}

func parseBookRow(sel *goquery.Selection) (scrapeResult, bool) {
	titleSel := sel.Find("div.book-title a").First()
	if titleSel.Length() == 0 {
		return scrapeResult{}, false
	}
	href, _ := titleSel.Attr("href")
	workID, err := strconv.ParseInt(lastSegment(href), 10, 64)
	if err != nil {
		return scrapeResult{}, false
	}
	title := strings.TrimSpace(titleSel.Text())

	cover, _ := sel.Find("img").First().Attr("src")
	annotation := strings.TrimSpace(sel.Find("div.annotation").First().Text())

	var seriesID int64
	var seriesTitle string
	seriesSel := sel.Find("a[href^='/work/series']").First()
	if seriesSel.Length() > 0 {
		seriesTitle = strings.TrimSpace(seriesSel.Text())
		if id, err := strconv.ParseInt(lastSegment(strings.TrimPrefix(seriesSel.AttrOr("href", ""), "/work/series")), 10, 64); err == nil {
			seriesID = id
		}
	}

	authorID, authorName := parseAuthorLink(sel.Find("div.book-author a[href]").First())
	if authorName == "" {
		return scrapeResult{}, false
	}

	completed := sel.Find("i[class^='icon-pencil book-status-icon']").Length() == 0

	return scrapeResult{
		WorkID:     workID,
		Title:      title,
		Annotation: annotation,
		CoverURL:   absCoverURL(cover),
		AuthorID:   authorID,
		AuthorName: authorName,
		Completed:  completed,
		SeriesID:   seriesID,
		Series:     seriesTitle,
	}, true
}

func parseAuthorLink(sel *goquery.Selection) (string, string) {
	if sel.Length() == 0 {
		return "", ""
	}
	href, _ := sel.Attr("href")
	id := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(href, "/u/", ""), "/works?format=ebook", ""))
	name := strings.TrimSpace(sel.Text())
	return id, name
}

func lastSegment(path string) string {
	path = strings.TrimRight(path, "/")
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}
