package podsandbox

import (
	"sync"

	"demo/pkg/containerd"
)

type Status struct {
	Waiter <-chan containerd.ExitStatus
}

type Store struct {
	sync.Map
}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) Save(id string, exitCh <-chan containerd.ExitStatus) {
	s.Store(id, &Status{Waiter: exitCh})
}

func (s *Store) Get(id string) *Status {
	i, ok := s.LoadAndDelete(id)
	if !ok {
		// not exist
		return nil
	}
	// Only save *Status
	return i.(*Status)
}
