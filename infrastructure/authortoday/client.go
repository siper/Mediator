package authortoday

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"stersh.ru/mediator/domain"
)

var siteBase = "https://author.today"

const (
	minRequestInterval = time.Second
)

var apiBase = "https://api.author.today/"

type TokenResolver func() string

type APIClient struct {
	http  *http.Client
	token TokenResolver

	mu      sync.Mutex
	lastReq time.Time
}

func NewAPIClient(token TokenResolver) *APIClient {
	return &APIClient{
		http:  &http.Client{Timeout: 30 * time.Second},
		token: token,
	}
}

func (c *APIClient) HasToken() bool {
	return c.token != nil && c.token() != ""
}

func (c *APIClient) TestConnection(ctx context.Context) error {
	if !c.HasToken() {
		return fmt.Errorf("author.today: %w", domain.ErrNoToken)
	}
	c.throttle()

	u := apiBase + "v1/work/1/details"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token())
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("author.today: invalid token (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("author.today: HTTP %d", resp.StatusCode)
	}
	return nil
}

func (c *APIClient) WorkDetails(ctx context.Context, workID int64) (*workDetails, error) {
	var w workDetails
	if err := c.getJSON(ctx, "v1/work/"+strconv.FormatInt(workID, 10)+"/details", nil, &w); err != nil {
		return nil, err
	}
	return &w, nil
}

type chapterResponse struct {
	IsSuccessful bool `json:"isSuccessful"`
	Data         struct {
		Text string `json:"text"`
	} `json:"data"`
}

func (c *APIClient) ChapterText(ctx context.Context, workID, chapterID int64) (text, readerSecret string, err error) {
	if !c.HasToken() {
		return "", "", fmt.Errorf("author.today: %w", domain.ErrNoToken)
	}
	c.throttle()

	q := url.Values{}
	q.Set("id", strconv.FormatInt(chapterID, 10))
	u := siteBase + "/reader/" + strconv.FormatInt(workID, 10) + "/chapter?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.token())
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("author.today reader chapter %d: HTTP %d: %s", chapterID, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var cr chapterResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return "", "", fmt.Errorf("author.today reader chapter %d: decode: %w", chapterID, err)
	}
	if !cr.IsSuccessful {
		return "", "", fmt.Errorf("author.today reader chapter %d: not successful", chapterID)
	}
	return cr.Data.Text, resp.Header.Get("Reader-Secret"), nil
}

func (c *APIClient) getJSON(ctx context.Context, path string, params url.Values, dst any) error {
	if !c.HasToken() {
		slog.Warn("author.today api: no token", "path", path)
		return fmt.Errorf("author.today: %w", domain.ErrNoToken)
	}
	c.throttle()

	u := apiBase + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	slog.Debug("author.today api: request", "path", path, "url", u)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		slog.Warn("author.today api: request creation failed", "path", path, "err", err)
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token())
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		slog.Warn("author.today api: request failed", "path", path, "err", err)
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if strings.Contains(path, "many-texts") {
		slog.Debug("author.today api: many-texts raw body", "path", path, "body_len", len(body), "body_preview", func() string {
			if len(body) > 500 {
				return string(body[:500])
			}
			return string(body)
		}())
	}
	if resp.StatusCode != http.StatusOK {
		slog.Warn("author.today api: non-200 response", "path", path, "status", resp.StatusCode, "body", strings.TrimSpace(string(body)))
		return fmt.Errorf("author.today api %s: HTTP %d: %s", path, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.Unmarshal(body, dst); err != nil {
		slog.Warn("author.today api: decode failed", "path", path, "err", err, "body_len", len(body))
		return fmt.Errorf("author.today api %s: decode: %w", path, err)
	}
	slog.Debug("author.today api: response ok", "path", path)
	return nil
}

func (c *APIClient) throttle() {
	c.mu.Lock()
	defer c.mu.Unlock()
	wait := minRequestInterval - time.Since(c.lastReq)
	if wait > 0 && wait < minRequestInterval {
		time.Sleep(wait)
	}
	c.lastReq = time.Now()
}

func absCoverURL(raw string) string {
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	if strings.HasPrefix(raw, "//") {
		return "https:" + raw
	}
	if strings.HasPrefix(raw, "/") {
		return siteBase + raw
	}
	return siteBase + "/" + raw
}

var _ domain.TestableClient = (*APIClient)(nil)
