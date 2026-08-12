package torznab

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"stersh.ru/mediator/domain"
)

const httpTimeout = 30 * time.Second

type Client struct {
	name     string
	endpoint string
	apiKey   string
	http     *http.Client
}

func New(name, endpoint, apiKey string) *Client {
	return &Client{
		name:     name,
		endpoint: endpoint,
		apiKey:   apiKey,
		http:     &http.Client{Timeout: httpTimeout},
	}
}

func (c *Client) Name() string { return c.name }

func (c *Client) Search(ctx context.Context, query string, cats []int) ([]domain.Release, error) {
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("t", "search")
	q.Set("q", query)
	if len(cats) > 0 {
		q.Set("cat", joinInts(cats))
	}
	if c.apiKey != "" {
		q.Set("apikey", c.apiKey)
	}
	u.RawQuery = q.Encode()

	return c.fetch(ctx, u.String(), c.name)
}

func (c *Client) RSS(ctx context.Context, cats []int, since time.Time) ([]domain.Release, error) {
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("t", "rss")
	if len(cats) > 0 {
		q.Set("cat", joinInts(cats))
	}
	if !since.IsZero() {
		q.Set("since", strconv.FormatInt(since.Unix(), 10))
	}
	if c.apiKey != "" {
		q.Set("apikey", c.apiKey)
	}
	u.RawQuery = q.Encode()

	return c.fetch(ctx, u.String(), c.name)
}

func (c *Client) TestConnection(ctx context.Context) error {
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return err
	}
	q := u.Query()
	q.Set("t", "caps")
	if c.apiKey != "" {
		q.Set("apikey", c.apiKey)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("torznab %s: HTTP %d", c.name, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if desc := parseCapsError(body); desc != "" {
		return fmt.Errorf("torznab %s: %s", c.name, desc)
	}
	return nil
}

func parseCapsError(data []byte) string {
	var root struct {
		XMLName     xml.Name `xml:"error"`
		Description string   `xml:"description,attr"`
	}
	if err := xml.Unmarshal(data, &root); err == nil && root.Description != "" {
		return root.Description
	}
	var nested struct {
		Error *struct {
			Description string `xml:"description,attr"`
		} `xml:"error"`
	}
	_ = xml.Unmarshal(data, &nested)
	if nested.Error != nil && nested.Error.Description != "" {
		return nested.Error.Description
	}
	return ""
}

func (c *Client) fetch(ctx context.Context, target, indexerName string) ([]domain.Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("torznab %s: HTTP %d", c.name, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parseResults(body, indexerName)
}

type rss struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Items []item `xml:"item"`
	} `xml:"channel"`
}

type item struct {
	Title     string  `xml:"title"`
	Link      string  `xml:"link"`
	GUID      string  `xml:"guid"`
	PubDate   string  `xml:"pubDate"`
	Enclosure enclose `xml:"enclosure"`
	Attrs     []attr  `xml:"attr"`
}

type enclose struct {
	URL    string `xml:"url,attr"`
	Length string `xml:"length,attr"`
}

type attr struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

func parseResults(data []byte, indexerName string) ([]domain.Release, error) {
	var feed rss
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, err
	}

	releases := make([]domain.Release, 0, len(feed.Channel.Items))
	for _, it := range feed.Channel.Items {
		r := domain.Release{
			Title:       it.Title,
			DownloadURL: it.Enclosure.URL,
			Indexer:     indexerName,
		}
		if r.DownloadURL == "" {
			r.DownloadURL = it.Link
		}
		for _, a := range it.Attrs {
			switch a.Name {
			case "magneturl":
				r.MagnetURI = a.Value
			case "infohash":
				r.InfoHash = a.Value
			case "seeders":
				r.Seeders, _ = strconv.Atoi(a.Value)
			case "size":
				if v, err := strconv.ParseInt(a.Value, 10, 64); err == nil {
					r.Size = v
				}
			}
		}
		if r.MagnetURI == "" && r.InfoHash != "" {
			r.MagnetURI = "magnet:?xt=urn:btih:" + r.InfoHash
		}
		if r.Size == 0 && it.Enclosure.Length != "" {
			if v, err := strconv.ParseInt(it.Enclosure.Length, 10, 64); err == nil {
				r.Size = v
			}
		}
		if it.PubDate != "" {
			if t, err := time.Parse(time.RFC1123Z, it.PubDate); err == nil {
				r.PublishDate = t
			}
		}
		releases = append(releases, r)
	}
	return releases, nil
}

func joinInts(xs []int) string {
	parts := make([]string, len(xs))
	for i, v := range xs {
		parts[i] = strconv.Itoa(v)
	}
	return strings.Join(parts, ",")
}

var _ domain.TestableIndexer = (*Client)(nil)
