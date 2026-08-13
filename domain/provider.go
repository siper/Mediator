package domain

import (
	"context"
	"strconv"
	"strings"
)

type SearchResult struct {
	ProviderName string
	ExternalID   string
	Title        string
	Overview     string
	CoverURL     string
	MediaType    MediaType
	Year         *int
}

func YearFromDate(s string) *int {
	s = strings.TrimSpace(s)
	if len(s) < 4 {
		return nil
	}
	y, err := strconv.Atoi(s[:4])
	if err != nil {
		return nil
	}
	return YearPtr(y)
}

func YearPtr(y int) *int {
	if y < 1000 || y > 3000 {
		return nil
	}
	return &y
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
