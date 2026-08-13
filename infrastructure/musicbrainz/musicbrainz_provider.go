package musicbrainz

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"stersh.ru/mediator/domain"
	"stersh.ru/mediator/infrastructure/httpc"
)

const Name = "musicbrainz"

const userAgent = "media/0.1 (https://github.com/siper/Mediator; contact: mediator@example.com)"

var (
	baseURL     = "https://musicbrainz.org/ws/2"
	coverBase   = "https://coverartarchive.org/release"
	httpTimeout = 30 * time.Second
)

type MusicBrainzProvider struct {
	http *http.Client
}

func NewProvider(proxy *domain.Proxy) *MusicBrainzProvider {
	return &MusicBrainzProvider{
		http: httpc.NewClient(httpTimeout, proxy),
	}
}

func (p *MusicBrainzProvider) Name() string {
	return Name
}

func (p *MusicBrainzProvider) Search(query string, mediaType *domain.MediaType, page, limit int) ([]domain.SearchResult, bool, error) {
	if mediaType != nil && *mediaType != domain.MediaTypeMusicAlbum {
		return nil, false, nil
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}

	q := url.Values{}
	q.Set("query", query)
	q.Set("fmt", "json")
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa((page-1)*limit))
	u := baseURL + "/release?" + q.Encode()
	body, err := p.get(u)
	if err != nil {
		return nil, false, err
	}

	var res mbSearchResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, false, err
	}

	hasMore := res.Offset+len(res.Releases) < res.Count
	return dedupByReleaseGroup(res.Releases), hasMore, nil
}

func (p *MusicBrainzProvider) GetMedia(externalID string, mediaType domain.MediaType) (*domain.ProviderMedia, error) {
	if mediaType != domain.MediaTypeMusicAlbum {
		return nil, fmt.Errorf("musicbrainz: unsupported media type %v", mediaType)
	}

	q := url.Values{}
	q.Set("inc", "artist-credits recordings")
	q.Set("fmt", "json")
	u := baseURL + "/release/" + externalID + "?" + q.Encode()
	body, err := p.get(u)
	if err != nil {
		return nil, err
	}

	var rel mbRelease
	if err := json.Unmarshal(body, &rel); err != nil {
		return nil, err
	}

	title := firstArtistName(rel.ArtistCredit) + " - " + rel.Title

	var cover string
	if rel.CoverArt != nil && (rel.CoverArt.Available || rel.CoverArt.Front) {
		cover = makeCoverURL(rel.ID)
	}

	var groups []domain.ProviderGroup
	var parts []domain.ProviderPart

	for _, medium := range rel.MediumList {
		discName := "Disc " + strconv.Itoa(medium.Position)
		if medium.Position == 1 && len(rel.MediumList) == 1 {
			discName = "Album"
		}
		groupOrder := medium.Position

		groups = append(groups, domain.ProviderGroup{
			Name:  discName,
			Order: groupOrder,
		})

		for _, track := range medium.TrackList {
			trackNum := track.Number
			name := track.Title
			parts = append(parts, domain.ProviderPart{
				Name:       &name,
				GroupOrder: parseIntPtr(trackNum),
				GroupName:  &discName,
			})
		}

	}

	status := domain.MediaStatusCompleted
	if len(parts) == 0 {
		groups = nil
	}

	return &domain.ProviderMedia{
		ExternalID:   rel.ID,
		Title:        title,
		Overview:     "",
		CoverURL:     cover,
		MediaType:    domain.MediaTypeMusicAlbum,
		Status:       status,
		LastModified: rel.LastUpdated,
		Groups:       groups,
		Parts:        parts,
	}, nil
}

func (p *MusicBrainzProvider) FetchLastModified(ctx context.Context, externalID string, mediaType domain.MediaType) (string, error) {
	if mediaType != domain.MediaTypeMusicAlbum {
		return "", nil
	}

	q := url.Values{}
	q.Set("fmt", "json")
	u := baseURL + "/release/" + externalID + "?" + q.Encode()
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
		return "", fmt.Errorf("musicbrainz: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var rel mbRelease
	if err := json.Unmarshal(body, &rel); err != nil {
		return "", err
	}
	return rel.LastUpdated, nil
}

func (p *MusicBrainzProvider) TestConnection(ctx context.Context) error {
	u := baseURL + "/release?query=" + url.QueryEscape("a") + "&fmt=json&limit=1"
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
		return fmt.Errorf("musicbrainz: HTTP %d", resp.StatusCode)
	}
	return nil
}

func (p *MusicBrainzProvider) get(url string) ([]byte, error) {
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
		return nil, fmt.Errorf("musicbrainz: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func makeCoverURL(mbid string) string {
	return coverBase + "/" + mbid + "/front"
}

func dedupByReleaseGroup(releases []mbSearchRelease) []domain.SearchResult {
	type entry struct {
		result   domain.SearchResult
		hasCover bool
	}
	seen := make(map[string]int, len(releases))
	out := make([]entry, 0, len(releases))
	for _, r := range releases {
		key := r.ID
		if r.ReleaseGroup != nil && r.ReleaseGroup.ID != "" {
			key = r.ReleaseGroup.ID
		}
		artistName := firstArtistName(r.ArtistCredit)
		cover := hasCoverArt(r.CoverArt)
		sr := domain.SearchResult{
			ProviderName: Name,
			ExternalID:   r.ID,
			Title:        artistName + " - " + r.Title,
			Overview:     "",
			CoverURL:     makeCoverURL(r.ID),
			MediaType:    domain.MediaTypeMusicAlbum,
			Year:         domain.YearFromDate(r.Date),
		}
		if idx, ok := seen[key]; ok {
			if cover && !out[idx].hasCover {
				out[idx] = entry{result: sr, hasCover: cover}
			}
			continue
		}
		seen[key] = len(out)
		out = append(out, entry{result: sr, hasCover: cover})
	}
	results := make([]domain.SearchResult, len(out))
	for i, e := range out {
		results[i] = e.result
	}
	return results
}

func hasCoverArt(c *mbCoverArt) bool {
	return c != nil && (c.Available || c.Front || c.Artwork)
}

func firstArtistName(credit []mbArtistCreditEntry) string {
	for _, c := range credit {
		if c.Artist != nil && c.Artist.Name != "" {
			return c.Artist.Name
		}
		if c.Name != "" {
			return c.Name
		}
	}
	return ""
}

func parseIntPtr(s string) *int {
	if s == "" {
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &n
}

type mbSearchResponse struct {
	Count    int               `json:"count"`
	Offset   int               `json:"offset"`
	Releases []mbSearchRelease `json:"releases"`
}

type mbSearchRelease struct {
	ID           string                `json:"id"`
	Title        string                `json:"title"`
	Date         string                `json:"date"`
	ArtistCredit []mbArtistCreditEntry `json:"artist-credit"`
	CoverArt     *mbCoverArt           `json:"cover-art-archive"`
	ReleaseGroup *mbReleaseGroup       `json:"release-group"`
}

type mbReleaseGroup struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	PrimaryType string `json:"primary-type"`
}

type mbRelease struct {
	ID           string                `json:"id"`
	Title        string                `json:"title"`
	ArtistCredit []mbArtistCreditEntry `json:"artist-credit"`
	Date         string                `json:"date"`
	LastUpdated  string                `json:"lastupdated"`
	MediumList   []mbMedium            `json:"media"`
	CoverArt     *mbCoverArt           `json:"cover-art-archive"`
}

type mbMedium struct {
	Position  int       `json:"position"`
	TrackList []mbTrack `json:"tracks"`
}

type mbTrack struct {
	Title  string `json:"title"`
	Number string `json:"number"`
}

type mbArtistCreditEntry struct {
	Name   string       `json:"name"`
	Artist *mbArtistRef `json:"artist"`
}

type mbArtistRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type mbCoverArt struct {
	Artwork   bool `json:"artwork"`
	Available bool `json:"available"`
	Front     bool `json:"front"`
	Count     int  `json:"count"`
}

var _ domain.MediaProvider = (*MusicBrainzProvider)(nil)
var _ domain.VersionedReader = (*MusicBrainzProvider)(nil)
var _ domain.TestableProvider = (*MusicBrainzProvider)(nil)
