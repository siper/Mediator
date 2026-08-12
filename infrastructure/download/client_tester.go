package download

import (
	"context"

	"stersh.ru/mediator/domain"
)

type ClientBuilder func(domain.DownloadClient) (domain.TestableClient, error)

type ClientTester struct {
	builder ClientBuilder
}

func NewClientTester(builder ClientBuilder) *ClientTester {
	return &ClientTester{builder: builder}
}

func (t *ClientTester) Test(ctx context.Context, c domain.DownloadClient) error {
	client, err := t.builder(c)
	if err != nil {
		return err
	}
	return client.TestConnection(ctx)
}

var _ domain.DownloadClientTester = (*ClientTester)(nil)
