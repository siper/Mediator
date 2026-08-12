package domain

import "context"

type SearchResult struct {
	ProviderName string
	ExternalID   string
	Title        string
	Overview     string
	CoverURL     string
	MediaType    MediaType
}

type SearchPage struct {
	Results []SearchResult
	Page    int
	Limit   int
	HasMore bool
}

type ProviderMedia struct {
	ExternalID   string
	Title        string
	OriginalName string
	Overview     string
	CoverURL     string
	MediaType    MediaType
	Status       MediaStatus
	LastModified string
	Groups       []ProviderGroup
	Parts        []ProviderPart
}

type ProviderGroup struct {
	Name  string
	Order int
}

type ProviderPart struct {
	Name       *string
	GroupName  *string
	GroupOrder *int
}

type MediaProvider interface {
	Name() string
	Search(query string, mediaType *MediaType, page, limit int) (results []SearchResult, hasMore bool, err error)
	GetMedia(externalID string, mediaType MediaType) (*ProviderMedia, error)
}

type TestableProvider interface {
	TestConnection(ctx context.Context) error
}

type VersionedReader interface {
	FetchLastModified(ctx context.Context, externalID string, mediaType MediaType) (string, error)
}
