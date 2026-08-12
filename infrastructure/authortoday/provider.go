package authortoday

import (
	"context"
	"fmt"
	"strconv"

	"stersh.ru/mediator/domain"
)

const Name = "author_today"

type Provider struct {
	scraper *Scraper
	client  *APIClient
}

func NewProvider(scraper *Scraper, client *APIClient) *Provider {
	return &Provider{scraper: scraper, client: client}
}

func (p *Provider) Name() string { return Name }

func (p *Provider) TestConnection(ctx context.Context) error {
	if p.scraper == nil {
		return fmt.Errorf("author.today: no scraper configured")
	}
	return p.scraper.TestConnection(ctx)
}

func (p *Provider) Search(query string, mediaType *domain.MediaType, page, limit int) ([]domain.SearchResult, bool, error) {
	if mediaType != nil && *mediaType != domain.MediaTypeBook {
		return nil, false, nil
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 25
	}
	results, err := p.scraper.SearchWorks(context.Background(), query)
	if err != nil {
		return nil, false, err
	}
	out := make([]domain.SearchResult, 0, len(results))
	for _, r := range results {
		out = append(out, domain.SearchResult{
			ProviderName: Name,
			ExternalID:   strconv.FormatInt(r.WorkID, 10),
			Title:        r.Title,
			Overview:     r.Annotation,
			CoverURL:     r.CoverURL,
			MediaType:    domain.MediaTypeBook,
		})
	}
	start := (page - 1) * limit
	if start >= len(out) {
		return nil, false, nil
	}
	end := start + limit
	hasMore := end < len(out)
	if end > len(out) {
		end = len(out)
	}
	return out[start:end], hasMore, nil
}

func (p *Provider) GetMedia(externalID string, mediaType domain.MediaType) (*domain.ProviderMedia, error) {
	workID, err := strconv.ParseInt(externalID, 10, 64)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()

	if p.client != nil && p.client.HasToken() {
		w, err := p.client.WorkDetails(ctx, workID)
		if err == nil {
			return providerMediaFromDetails(w), nil
		}
	}

	r, err := p.scraper.GetWork(ctx, workID)
	if err != nil {
		return nil, err
	}
	return providerMediaFromScrape(r), nil
}

func (p *Provider) FetchLastModified(ctx context.Context, externalID string, _ domain.MediaType) (string, error) {
	if p.client == nil || !p.client.HasToken() {
		return "", domain.ErrNoToken
	}
	workID, err := strconv.ParseInt(externalID, 10, 64)
	if err != nil {
		return "", err
	}
	w, err := p.client.WorkDetails(ctx, workID)
	if err != nil {
		return "", err
	}
	return w.LastModificationTime, nil
}

func providerMediaFromDetails(w *workDetails) *domain.ProviderMedia {
	authors := []string{w.AuthorFIO}
	if w.CoAuthorFIO != "" {
		authors = append(authors, w.CoAuthorFIO)
	}
	overview := w.Annotation
	if len(authors) > 0 {
		overview = authors[0] + "\n\n" + overview
	}
	status := domain.MediaStatusContinuing
	if w.IsFinished {
		status = domain.MediaStatusCompleted
	}
	return &domain.ProviderMedia{
		ExternalID:   strconv.FormatInt(w.ID, 10),
		Title:        w.Title,
		Overview:     overview,
		CoverURL:     absCoverURL(w.CoverURL),
		MediaType:    domain.MediaTypeBook,
		Status:       status,
		LastModified: w.LastModificationTime,
	}
}

func providerMediaFromScrape(r *scrapeResult) *domain.ProviderMedia {
	overview := r.AuthorName
	if r.Series != "" {
		overview = r.Series + " — " + overview
	}
	status := domain.MediaStatusContinuing
	if r.Completed {
		status = domain.MediaStatusCompleted
	}
	return &domain.ProviderMedia{
		ExternalID: strconv.FormatInt(r.WorkID, 10),
		Title:      r.Title,
		Overview:   overview,
		CoverURL:   r.CoverURL,
		MediaType:  domain.MediaTypeBook,
		Status:     status,
	}
}

var _ domain.MediaProvider = (*Provider)(nil)
var _ domain.TestableProvider = (*Provider)(nil)
