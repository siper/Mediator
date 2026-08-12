package indexer

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"stersh.ru/mediator/domain"
	"stersh.ru/mediator/infrastructure/indexer/prowlarr"
	"stersh.ru/mediator/infrastructure/indexer/torznab"
)

const jackettTorznabPath = "/api/v2.0/indexers/all/results/torznab"

var (
	discoverClient = &http.Client{Timeout: 15 * time.Second}
	discoverFn     = prowlarr.Discover
)

func Build(ix domain.Indexer) ([]domain.ReleaseIndexer, error) {
	host := hostOnly(ix.Get("endpoint"))
	if host == "" {
		return nil, fmt.Errorf("indexer endpoint is empty")
	}
	apiKey := ix.Get("api_key")

	switch ix.Type {
	case domain.IndexerJackett:
		return []domain.ReleaseIndexer{torznab.New(ix.Name, host+jackettTorznabPath, apiKey)}, nil

	case domain.IndexerProwlarr:
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		defs, err := discoverFn(ctx, host, apiKey, discoverClient)
		if err != nil {
			return nil, fmt.Errorf("prowlarr discover %q: %w", host, err)
		}
		if len(defs) == 0 {
			return nil, fmt.Errorf("prowlarr %q: no enabled indexers found", host)
		}

		out := make([]domain.ReleaseIndexer, 0, len(defs))
		for _, d := range defs {
			name := fmt.Sprintf("%s: %s", ix.Name, d.Name)
			endpoint := host + "/" + strconv.Itoa(d.ID) + "/api"
			out = append(out, torznab.New(name, endpoint, apiKey))
		}
		return out, nil

	default:
		return nil, fmt.Errorf("unknown indexer type: %s", ix.Type)
	}
}

func hostOnly(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return ""
	}
	if !strings.Contains(endpoint, "://") {
		endpoint = "http://" + endpoint
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}
