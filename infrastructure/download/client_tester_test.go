package download

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

type fakeTestable struct {
	err error
}

func (f *fakeTestable) TestConnection(ctx context.Context) error {
	return f.err
}

func TestClientTester_Test_Success(t *testing.T) {
	testable := &fakeTestable{}
	builder := func(c domain.DownloadClient) (domain.TestableClient, error) {
		return testable, nil
	}
	tester := NewClientTester(builder)

	c := domain.DownloadClient{
		Type:     domain.DownloadClientQBittorrent,
		Name:     "qb",
		Settings: map[string]string{"host": "http://localhost:8080"},
	}
	err := tester.Test(context.Background(), c)
	assert.NoError(t, err)
}

func TestClientTester_Test_BuilderError(t *testing.T) {
	builder := func(c domain.DownloadClient) (domain.TestableClient, error) {
		return nil, errors.New("build failed")
	}
	tester := NewClientTester(builder)

	c := domain.DownloadClient{Type: domain.DownloadClientQBittorrent}
	err := tester.Test(context.Background(), c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "build failed")
}

func TestClientTester_Test_ConnectionError(t *testing.T) {
	testable := &fakeTestable{err: errors.New("connection refused")}
	builder := func(c domain.DownloadClient) (domain.TestableClient, error) {
		return testable, nil
	}
	tester := NewClientTester(builder)

	c := domain.DownloadClient{Type: domain.DownloadClientQBittorrent}
	err := tester.Test(context.Background(), c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}
