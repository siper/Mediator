package openlibrary

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"stersh.ru/mediator/domain"
	"stersh.ru/mediator/infrastructure/httpc"
)

const Name = "openlibrary"

const userAgent = "media/0.1 (https://github.com/siper/Mediator; contact: mediator@example.com)"

var (
	baseURL     = "https://openlibrary.org"
	coverBase   = "https://covers.openlibrary.org/b/id"
	httpTimeout = 30 * time.Second
	minQueryLen = 3
)

type OpenLibraryProvider struct {
	http       *http.Client
	baseURL    string
	authorBase string
}

func NewProvider(proxy *domain.Proxy) *OpenLibraryProvider {
	return &OpenLibraryProvider{
		http:       httpc.NewClient(httpTimeout, proxy),
		baseURL:    baseURL,
		authorBase: baseURL,
	}
}

func (p *OpenLibraryProvider) Name() string {
	return Name
}

func (p *OpenLibraryProvider) Search(query string, mediaType *domain.MediaType, page, limit int) ([]domain.SearchResult, bool, error) {
	if mediaType != nil && *mediaType != domain.MediaTypeBook {
		return nil, false, nil
	}
	if len(query) < minQueryLen {
		return nil, false, nil
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	q := url.Values{}
	q.Set("q", query)
	q.Set("limit", strconv.Itoa(limit))
	q.Set("page", strconv.Itoa(page))
	q.Set("fields", "key,title,author_name,cover_i,first_publish_year")
	u := p.baseURL + "/search.json?" + q.Encode()
	body, err := p.get(u)
	if err != nil {
		return nil, false, err
	}

	var res olSearchResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, false, err
	}

	results := make([]domain.SearchResult, 0, len(res.Docs))
	for _, d := range res.Docs {
		externalID := workKeyToOLID(d.Key)
		if externalID == "" {
			continue
		}
		title := buildBookTitle(d.AuthorName, d.Title)
		results = append(results, domain.SearchResult{
			ProviderName: Name,
			ExternalID:   externalID,
			Title:        title,
			CoverURL:     makeCoverURL(d.CoverID),
			MediaType:    domain.MediaTypeBook,
			Year:         domain.YearPtr(d.FirstPublishYear),
		})
	}
	hasMore := page*limit < res.NumFound
	return results, hasMore, nil
}

func (p *OpenLibraryProvider) GetMedia(externalID string, mediaType domain.MediaType) (*domain.ProviderMedia, error) {
	if mediaType != domain.MediaTypeBook {
		return nil, fmt.Errorf("openlibrary: unsupported media type %v", mediaType)
	}

	u := p.baseURL + "/works/" + externalID + ".json"
	body, err := p.get(u)
	if err != nil {
		return nil, err
	}

	var w olWork
	if err := json.Unmarshal(body, &w); err != nil {
		return nil, err
	}

	title := w.Title
	if authorName, err := p.fetchAuthorName(w); err == nil && authorName != "" {
		title = authorName + " — " + w.Title
	}

	return &domain.ProviderMedia{
		ExternalID:   externalID,
		Title:        title,
		Overview:     w.Description.Value,
		CoverURL:     makeCoverURLFromArray(w.Covers),
		MediaType:    domain.MediaTypeBook,
		Status:       domain.MediaStatusCompleted,
		LastModified: w.LastModified.Value,
	}, nil
}

func (p *OpenLibraryProvider) TestConnection(ctx context.Context) error {
	u := p.baseURL + "/search.json?q=test&limit=1"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := p.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("openlibrary: HTTP %d", resp.StatusCode)
	}
	return nil
}

func (p *OpenLibraryProvider) FetchLastModified(ctx context.Context, externalID string, mediaType domain.MediaType) (string, error) {
	if mediaType != domain.MediaTypeBook {
		return "", nil
	}

	u := p.baseURL + "/works/" + externalID + ".json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := p.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openlibrary: HTTP %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var w olWork
	if err := json.Unmarshal(body, &w); err != nil {
		return "", err
	}
	return w.LastModified.Value, nil
}

func (p *OpenLibraryProvider) fetchAuthorName(w olWork) (string, error) {
	if len(w.Authors) == 0 || w.Authors[0].Author.Key == "" {
		return "", nil
	}
	u := p.authorBase + w.Authors[0].Author.Key + ".json"
	body, err := p.get(u)
	if err != nil {
		return "", err
	}
	var a olAuthor
	if err := json.Unmarshal(body, &a); err != nil {
		return "", err
	}
	return a.Name, nil
}

func (p *OpenLibraryProvider) get(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := p.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openlibrary: %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func workKeyToOLID(key string) string {
	const prefix = "/works/"
	if !strings.HasPrefix(key, prefix) {
		return ""
	}
	return strings.TrimSuffix(strings.TrimPrefix(key, prefix), ".json")
}

func buildBookTitle(authors []string, title string) string {
	if len(authors) == 0 || authors[0] == "" {
		return title
	}
	return authors[0] + " — " + title
}

func makeCoverURL(coverID int) string {
	if coverID <= 0 {
		return ""
	}
	return coverBase + "/" + strconv.Itoa(coverID) + "-M.jpg"
}

func makeCoverURLFromArray(covers []int) string {
	if len(covers) == 0 {
		return ""
	}
	return makeCoverURL(covers[0])
}

type olSearchResponse struct {
	NumFound int           `json:"numFound"`
	Docs     []olSearchDoc `json:"docs"`
}

type olSearchDoc struct {
	Key              string   `json:"key"`
	Title            string   `json:"title"`
	AuthorName       []string `json:"author_name"`
	CoverID          int      `json:"cover_i"`
	FirstPublishYear int      `json:"first_publish_year"`
}

type olWork struct {
	Title        string         `json:"title"`
	Authors      []olWorkAuthor `json:"authors"`
	Covers       []int          `json:"covers"`
	Description  olTypedText    `json:"description"`
	LastModified olTypedText    `json:"last_modified"`
}

type olWorkAuthor struct {
	Author olRef `json:"author"`
}

type olRef struct {
	Key string `json:"key"`
}

type olTypedText struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

func (t *olTypedText) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		t.Value = s
		t.Type = ""
		return nil
	}
	type alias olTypedText
	var obj alias
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	*t = olTypedText(obj)
	return nil
}

type olAuthor struct {
	Name string `json:"name"`
}

var (
	_ domain.MediaProvider    = (*OpenLibraryProvider)(nil)
	_ domain.TestableProvider = (*OpenLibraryProvider)(nil)
	_ domain.VersionedReader  = (*OpenLibraryProvider)(nil)
)
