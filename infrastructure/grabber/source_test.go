package grabber

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

type fakeSink struct {
	received []domain.Grabber
}

func (f *fakeSink) SetGrabbers(g []domain.Grabber) {
	f.received = g
}

func TestClientReloader_PushesRebuildToAllSinks(t *testing.T) {
	var calls int
	a := &stubGrabber{name: "a"}
	b := &stubGrabber{name: "b"}
	rebuild := func() []domain.Grabber {
		calls++
		return []domain.Grabber{a}
	}
	s1 := &fakeSink{}
	s2 := &fakeSink{}

	r := NewClientReloader(rebuild, s1, s2)
	require.NoError(t, r.Reload())

	assert.Equal(t, 1, calls)
	require.Len(t, s1.received, 1)
	require.Len(t, s2.received, 1)
	assert.Same(t, a, s1.received[0])
	assert.Same(t, a, s2.received[0])
	assert.NotSame(t, b, s1.received[0])

	s1.received = nil
	require.NoError(t, r.Reload())
	assert.Len(t, s1.received, 1)
	assert.Same(t, a, s1.received[0])
}

type stubGrabber struct{ name string }

func (s *stubGrabber) Name() string                    { return s.name }
func (s *stubGrabber) Supports(domain.GrabTarget) bool { return true }
func (s *stubGrabber) SupportsProvider(string) bool  { return false }
func (s *stubGrabber) Submit(context.Context, domain.GrabTarget) (domain.GrabHandle, error) {
	return domain.GrabHandle{}, nil
}
func (s *stubGrabber) Status(context.Context, domain.GrabHandle) (domain.GrabStatus, error) {
	return domain.GrabStatus{}, nil
}
func (s *stubGrabber) Cancel(context.Context, domain.GrabHandle) error { return nil }
