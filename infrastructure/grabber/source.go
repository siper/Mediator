package grabber

import "stersh.ru/mediator/domain"

type RebuildFunc func() []domain.Grabber

type ClientReloader struct {
	rebuild RebuildFunc
	sinks   []domain.GrabberSink
}

func NewClientReloader(rebuild RebuildFunc, sinks ...domain.GrabberSink) *ClientReloader {
	return &ClientReloader{rebuild: rebuild, sinks: sinks}
}

func (r *ClientReloader) Reload() error {
	next := r.rebuild()
	for _, s := range r.sinks {
		s.SetGrabbers(next)
	}
	return nil
}

var _ domain.ClientReloader = (*ClientReloader)(nil)
