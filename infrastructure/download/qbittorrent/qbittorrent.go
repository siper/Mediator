package qbittorrent

import (
	"context"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
	"unicode"

	"stersh.ru/mediator/domain"
)

type Client struct {
	name     string
	host     string
	username string
	password string
	http     *http.Client
}

func New(name, host, username, password string) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
		return &Client{
		name:     name,
		host:     strings.TrimRight(host, "/"),
		username: username,
		password: password,
		http:     &http.Client{Jar: jar, Timeout: 30 * time.Second},
	}, nil
}

func (c *Client) Name() string { return c.name }

func (c *Client) TestConnection(ctx context.Context) error {
	return c.login(ctx)
}

func (c *Client) login(ctx context.Context) error {
	form := url.Values{}
	form.Set("username", c.username)
	form.Set("password", c.password)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+"/api/v2/auth/login", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("qbittorrent login failed: unauthorized (bad credentials)")
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("qbittorrent login failed: HTTP %d", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusOK && !strings.HasPrefix(string(body), "Ok") && strings.TrimSpace(string(body)) != "" {
		return fmt.Errorf("qbittorrent login failed: %s", strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *Client) Add(ctx context.Context, r domain.Release, category string) (*domain.DownloadTask, error) {
	if err := c.login(ctx); err != nil {
		return nil, err
	}
	start := time.Now().Unix()
	form := url.Values{}
	src := r.MagnetURI
	if src == "" {
		src = r.DownloadURL
	}
	form.Set("urls", src)
	if category != "" {
		form.Set("category", category)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+"/api/v2/torrents/add", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("qbittorrent add failed: HTTP %d", resp.StatusCode)
	}
	id := infohashFromMagnet(r.MagnetURI)
	if id == "" {
		if hash, err := c.resolveInfoHash(ctx, r.Title, category, start); err == nil {
			id = hash
		} else {
			id = r.DownloadURL
		}
	}
	return &domain.DownloadTask{ID: id, Name: r.Title, Status: domain.DownloadQueued}, nil
}

func (c *Client) resolveInfoHash(ctx context.Context, name, category string, afterUnix int64) (string, error) {
	for i := 0; i < 5; i++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(time.Second):
		}
		torrents, err := c.listTorrents(ctx, category)
		if err != nil {
			return "", err
		}
		var bestHash string
		bestScore := 0
		nameTokens := tokenize(name)
		for _, t := range torrents {
			if t.AddedOn < afterUnix {
				continue
			}
			score := tokenOverlap(nameTokens, tokenize(t.Name))
			if score > bestScore {
				bestScore = score
				bestHash = t.Hash
			}
		}
		if bestHash != "" {
			return bestHash, nil
		}
	}
	return "", domain.ErrPartNotFound
}

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

func matchReleaseName(a, b string) bool {
	ta := tokenize(a)
	tb := tokenize(b)
	if len(ta) == 0 || len(tb) == 0 {
		return false
	}
	common := 0
	for w := range ta {
		if _, ok := tb[w]; ok {
			common++
		}
	}
	union := len(ta) + len(tb) - common
	if union == 0 {
		return false
	}
	return float64(common)/float64(union) >= 0.4 && common >= 2
}

func tokenOverlap(a, b map[string]struct{}) int {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	common := 0
	for w := range a {
		if _, ok := b[w]; ok {
			common++
		}
	}
	return common
}

func (c *Client) Get(ctx context.Context, id string) (*domain.DownloadTask, error) {
	if err := c.login(ctx); err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("hashes", id)
	u := c.host + "/api/v2/torrents/info?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qbittorrent info failed: HTTP %d", resp.StatusCode)
	}
	var items []qbTorrent
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, domain.ErrPartNotFound
	}
	return mapTorrent(items[0]), nil
}

func (c *Client) List(ctx context.Context) ([]domain.DownloadTask, error) {
	items, err := c.listTorrents(ctx, "")
	if err != nil {
		return nil, err
	}
	tasks := make([]domain.DownloadTask, 0, len(items))
	for _, it := range items {
		tasks = append(tasks, *mapTorrent(it))
	}
	return tasks, nil
}

func (c *Client) listTorrents(ctx context.Context, category string) ([]qbTorrent, error) {
	if err := c.login(ctx); err != nil {
		return nil, err
	}
	u := c.host + "/api/v2/torrents/info"
	if category != "" {
		u += "?category=" + url.QueryEscape(category)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qbittorrent info failed: HTTP %d", resp.StatusCode)
	}
	var items []qbTorrent
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c *Client) Remove(ctx context.Context, id string, deleteData bool) error {
	if err := c.login(ctx); err != nil {
		return err
	}
	form := url.Values{}
	form.Set("hashes", id)
	form.Set("files", boolStr(deleteData))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+"/api/v2/torrents/delete", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("qbittorrent delete failed: HTTP %d", resp.StatusCode)
	}
	return nil
}

type qbTorrent struct {
	Hash        string  `json:"hash"`
	Name        string  `json:"name"`
	State       string  `json:"state"`
	Progress    float64 `json:"progress"`
	ContentPath string  `json:"content_path"`
	AddedOn     int64   `json:"added_on"`
}

func mapTorrent(t qbTorrent) *domain.DownloadTask {
	status := domain.DownloadQueued
	switch {
	case t.State == "error":
		status = domain.DownloadFailed
	case t.Progress >= 1.0 || isSeedingState(t.State):
		status = domain.DownloadCompleted
	case isDownloadingState(t.State):
		status = domain.Downloading
	}
	return &domain.DownloadTask{
		ID:         t.Hash,
		Name:       t.Name,
		Status:     status,
		Progress:   t.Progress,
		OutputPath: t.ContentPath,
	}
}

func isSeedingState(s string) bool {
	switch s {
	case "uploading", "queuedUP", "stalledUP", "pausedUP", "stoppedUP", "forcedUP", "checkingUP":
		return true
	}
	return false
}

func isDownloadingState(s string) bool {
	switch s {
	case "downloading", "metaDL", "forcedDL", "stalledDL", "pausedDL", "stoppedDL":
		return true
	}
	return false
}

func infohashFromMagnet(magnet string) string {
	u, err := url.Parse(magnet)
	if err != nil {
		return ""
	}
	xt := u.Query().Get("xt")
	if i := strings.LastIndex(xt, ":"); i != -1 {
		hash := strings.ToLower(xt[i+1:])
		if len(hash) == 32 {
			if decoded, err := base32.StdEncoding.DecodeString(strings.ToUpper(hash)); err == nil {
				return hex.EncodeToString(decoded)
			}
		}
		return hash
	}
	return ""
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

var _ domain.DownloadRunner = (*Client)(nil)
var _ domain.TestableClient = (*Client)(nil)
