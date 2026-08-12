package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"

	"stersh.ru/mediator/domain"
	"stersh.ru/mediator/infrastructure/httpc"
)

const Name = "tmdb"

var (
	baseURL   = "https://api.themoviedb.org/3"
	httpTimeout = 30 * time.Second
)

type TMDBProvider struct {
	apiKey string
	lang   string
	http   *http.Client
}

func NewTMDBProvider(apiKey, lang string, proxy *domain.Proxy) *TMDBProvider {
	return &TMDBProvider{
		apiKey: apiKey,
		lang:   lang,
		http:   httpc.NewClient(httpTimeout, proxy),
	}
}

func (p *TMDBProvider) Name() string {
	return Name
}

func (p *TMDBProvider) Search(query string, mediaType *domain.MediaType, page, limit int) ([]domain.SearchResult, bool, error) {
	if page < 1 {
		page = 1
	}
	q := url.Values{}
	q.Set("query", query)
	q.Set("language", p.lang)
	q.Set("page", strconv.Itoa(page))
	u := baseURL + "/search/multi?" + q.Encode()
	body, err := p.get(u)
	if err != nil {
		return nil, false, err
	}

	var searchRes tmdbSearchResponse
	if err := json.Unmarshal(body, &searchRes); err != nil {
		return nil, false, err
	}

	var results []domain.SearchResult
	for _, r := range searchRes.Results {
		if r.MediaType != "movie" && r.MediaType != "tv" {
			continue
		}
		mt := tmdbMediaTypeToDomain(r.MediaType)
		if mediaType != nil && mt != *mediaType {
			continue
		}
		title := r.Title
		if title == "" {
			title = r.Name
		}
		result := domain.SearchResult{
			ProviderName: Name,
			ExternalID:   strconv.Itoa(r.ID),
			Title:        title,
			Overview:     r.Overview,
			CoverURL:     p.buildCoverURL(r.PosterPath),
			MediaType:    mt,
		}
		results = append(results, result)
	}
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	hasMore := searchRes.Page < searchRes.TotalPages
	return results, hasMore, nil
}

func (p *TMDBProvider) GetMedia(externalID string, mediaType domain.MediaType) (*domain.ProviderMedia, error) {
	switch mediaType {
	case domain.MediaTypeMovie:
		return p.getMovie(externalID)
	case domain.MediaTypeSeries:
		return p.getTVSeries(externalID)
	default:
		return nil, fmt.Errorf("tmdb: unsupported media type %v", mediaType)
	}
}

func (p *TMDBProvider) getMovie(externalID string) (*domain.ProviderMedia, error) {
	q := url.Values{}
	q.Set("language", p.lang)
	u := baseURL + "/movie/" + externalID + "?" + q.Encode()
	body, err := p.get(u)
	if err != nil {
		return nil, err
	}

	var movie tmdbMovie
	if err := json.Unmarshal(body, &movie); err != nil {
		return nil, err
	}

	return &domain.ProviderMedia{
		ExternalID:   externalID,
		Title:        movie.Title,
		OriginalName: movie.OriginalTitle,
		Overview:     movie.Overview,
		CoverURL:     p.buildCoverURL(movie.PosterPath),
		MediaType:    domain.MediaTypeMovie,
	}, nil
}

func (p *TMDBProvider) getTVSeries(externalID string) (*domain.ProviderMedia, error) {
	q := url.Values{}
	q.Set("language", p.lang)
	u := baseURL + "/tv/" + externalID + "?" + q.Encode()
	body, err := p.get(u)
	if err != nil {
		return nil, err
	}

	var tv tmdbTV
	if err := json.Unmarshal(body, &tv); err != nil {
		return nil, err
	}

	pm := &domain.ProviderMedia{
		ExternalID:   externalID,
		Title:        tv.Name,
		OriginalName: tv.OriginalName,
		Overview:     tv.Overview,
		CoverURL:     p.buildCoverURL(tv.PosterPath),
		MediaType:    domain.MediaTypeSeries,
	}

	for _, s := range tv.Seasons {
		if s.SeasonNumber == 0 {
			continue
		}
		pm.Groups = append(pm.Groups, domain.ProviderGroup{
			Name:  s.Name,
			Order: s.SeasonNumber,
		})
	}

	type seasonEpisodes struct {
		season   tmdbTVSeason
		episodes []tmdbEpisode
	}

	var seasonResults []seasonEpisodes
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, s := range tv.Seasons {
		if s.SeasonNumber == 0 {
			continue
		}
		wg.Add(1)
		go func(season tmdbTVSeason) {
			defer wg.Done()
			episodes, err := p.getSeasonEpisodes(externalID, season.SeasonNumber)
			if err != nil {
				return
			}
			mu.Lock()
			seasonResults = append(seasonResults, seasonEpisodes{season: season, episodes: episodes})
			mu.Unlock()
		}(s)
	}
	wg.Wait()

	sort.Slice(seasonResults, func(i, j int) bool {
		return seasonResults[i].season.SeasonNumber < seasonResults[j].season.SeasonNumber
	})

	for _, sr := range seasonResults {
		groupName := sr.season.Name
		for _, ep := range sr.episodes {
			name := ep.Name
			order := ep.EpisodeNumber
			pm.Parts = append(pm.Parts, domain.ProviderPart{
				Name:       &name,
				GroupName:  &groupName,
				GroupOrder: &order,
			})
		}
	}

	return pm, nil
}

func (p *TMDBProvider) getSeasonEpisodes(tvID string, seasonNumber int) ([]tmdbEpisode, error) {
	q := url.Values{}
	q.Set("language", p.lang)
	u := baseURL + "/tv/" + tvID + "/season/" + strconv.Itoa(seasonNumber) + "?" + q.Encode()
	body, err := p.get(u)
	if err != nil {
		return nil, err
	}

	var season tmdbSeasonDetail
	if err := json.Unmarshal(body, &season); err != nil {
		return nil, err
	}
	return season.Episodes, nil
}

func (p *TMDBProvider) buildCoverURL(path string) string {
	if path == "" {
		return ""
	}
	return "https://image.tmdb.org/t/p/w500" + path
}

func (p *TMDBProvider) TestConnection(ctx context.Context) error {
	u := baseURL + "/configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	resp, err := p.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("tmdb: invalid API key (401)")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("tmdb: HTTP %d", resp.StatusCode)
	}
	return nil
}

func (p *TMDBProvider) get(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	resp, err := p.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tmdb: %s: %s", resp.Status, string(body))
	}
	return body, nil
}

func tmdbMediaTypeToDomain(t string) domain.MediaType {
	switch t {
	case "movie":
		return domain.MediaTypeMovie
	case "tv":
		return domain.MediaTypeSeries
	default:
		return domain.MediaTypeMovie
	}
}

var _ domain.TestableProvider = (*TMDBProvider)(nil)

type tmdbSearchResponse struct {
	Page         int `json:"page"`
	TotalPages   int `json:"total_pages"`
	TotalResults int `json:"total_results"`
	Results      []struct {
		ID          int    `json:"id"`
		MediaType   string `json:"media_type"`
		Title       string `json:"title"`
		Name        string `json:"name"`
		Overview    string `json:"overview"`
		PosterPath  string `json:"poster_path"`
		ReleaseDate string `json:"release_date"`
	} `json:"results"`
}

type tmdbMovie struct {
	Title         string `json:"title"`
	OriginalTitle string `json:"original_title"`
	Overview      string `json:"overview"`
	PosterPath    string `json:"poster_path"`
}

type tmdbTV struct {
	Name         string         `json:"name"`
	OriginalName string         `json:"original_name"`
	Overview     string         `json:"overview"`
	PosterPath   string         `json:"poster_path"`
	Seasons      []tmdbTVSeason `json:"seasons"`
}

type tmdbTVSeason struct {
	SeasonNumber int    `json:"season_number"`
	Name         string `json:"name"`
}

type tmdbSeasonDetail struct {
	Name     string        `json:"name"`
	Episodes []tmdbEpisode `json:"episodes"`
}

type tmdbEpisode struct {
	Name          string `json:"name"`
	EpisodeNumber int    `json:"episode_number"`
}
