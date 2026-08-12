package source

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

type fakeTesterProvider struct {
	err error
}

func (f *fakeTesterProvider) Name() string { return "fake" }
func (f *fakeTesterProvider) TestConnection(ctx context.Context) error {
	return f.err
}

func (f *fakeTesterProvider) Search(string, *domain.MediaType, int, int) ([]domain.SearchResult, bool, error) {
	return nil, false, nil
}
func (f *fakeTesterProvider) GetMedia(string, domain.MediaType) (*domain.ProviderMedia, error) {
	return nil, nil
}

type fakeRepo struct{}

func (fakeRepo) Add(*domain.Source) error                  { return nil }
func (fakeRepo) GetByID(domain.ID) (*domain.Source, error) { return nil, nil }
func (fakeRepo) List() ([]domain.Source, error)            { return nil, nil }
func (fakeRepo) Update(*domain.Source) error               { return nil }
func (fakeRepo) Remove(domain.ID) error                    { return nil }

func TestSource_Test_DelegatesToProvider(t *testing.T) {
	p := &fakeTesterProvider{}
	s := New(fakeRepo{}, func(domain.Source) (domain.MediaProvider, error) { return p, nil })

	err := s.Test(context.Background(), domain.Source{Type: "fake", Name: "x"})
	assert.NoError(t, err)
}

func TestSource_Test_UnknownType(t *testing.T) {
	s := New(fakeRepo{}, func(src domain.Source) (domain.MediaProvider, error) {
		return nil, errors.New("unknown source type: " + src.Type)
	})

	err := s.Test(context.Background(), domain.Source{Type: "bogus"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus")
}

func TestSource_Test_NotTestable(t *testing.T) {
	nt := &nonTestableProvider{}
	s := New(fakeRepo{}, func(domain.Source) (domain.MediaProvider, error) { return nt, nil })

	err := s.Test(context.Background(), domain.Source{Type: "notest"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support connection testing")
}

type nonTestableProvider struct{}

func (nonTestableProvider) Name() string                                                      { return "n/a" }
func (nonTestableProvider) Search(string, *domain.MediaType, int, int) ([]domain.SearchResult, bool, error) {
	return nil, false, nil
}
func (nonTestableProvider) GetMedia(string, domain.MediaType) (*domain.ProviderMedia, error)  { return nil, nil }
