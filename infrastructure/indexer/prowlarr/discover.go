package prowlarr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Definition struct {
	ID   int
	Name string
}

type indexerEntry struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Enable *bool  `json:"enable"`
}

func Discover(ctx context.Context, host, apiKey string, client *http.Client) ([]Definition, error) {
	base := strings.TrimRight(strings.TrimSpace(host), "/")
	if base == "" {
		return nil, fmt.Errorf("empty host")
	}
	target := base + "/api/v1/indexer"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	if apiKey != "" {
		q := req.URL.Query()
		q.Set("apikey", apiKey)
		req.URL.RawQuery = q.Encode()
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prowlarr discover: HTTP %d", resp.StatusCode)
	}

	var entries []indexerEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, err
	}

	defs := make([]Definition, 0, len(entries))
	for _, e := range entries {
		if e.ID == 0 || strings.TrimSpace(e.Name) == "" {
			continue
		}
		if e.Enable != nil && !*e.Enable {
			continue
		}
		defs = append(defs, Definition{ID: e.ID, Name: e.Name})
	}
	return defs, nil
}
