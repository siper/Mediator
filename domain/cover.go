package domain

import "context"

type CoverStore interface {
	Store(ctx context.Context, sourceURL string) (servedPath string, err error)
}
