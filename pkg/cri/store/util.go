package store

import "sync"

// StopCh is used to propagate the stop information of a container.
type StopCh struct {
	ch   chan struct{}
	once sync.Once
}

// NewStopCh creates a stop channel. The channel is open by default.
func NewStopCh() *StopCh {
	return &StopCh{ch: make(chan struct{})}
}

// Stop close stopCh of the container.
func (s *StopCh) Stop() {
	s.once.Do(func() {
		close(s.ch)
	})
}

// Stopped return the stopCh of the container as a readonly channel.
func (s *StopCh) Stopped() <-chan struct{} {
	return s.ch
}
