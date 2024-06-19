package label

import (
	"sync"

	"github.com/opencontainers/selinux/go-selinux"
)

// Store is used to store SELinux process labels
type Store struct {
	sync.Mutex
	levels   map[string]int
	Releaser func(string)
	Reserver func(string)
}

// NewStore creates a new SELinux process label store
func NewStore() *Store {
	return &Store{
		levels:   map[string]int{},
		Releaser: selinux.ReleaseLabel,
		Reserver: selinux.ReserveLabel,
	}
}

// Reserve reserves the MLS/MCS level component of the specified label
// and prevents multiple reserves for the same level
func (s *Store) Reserve(label string) error {
	s.Lock()
	defer s.Unlock()

	context, err := selinux.NewContext(label)
	if err != nil {
		return err
	}

	level := context["level"]
	// no reason to count empty
	if level == "" {
		return nil
	}

	if _, ok := s.levels[level]; !ok {
		s.Reserver(label)
	}

	s.levels[level]++
	return nil
}

// Release un-reserves the MLS/MCS level component of the specified label,
// allowing it to be used by another process once labels with the same
// level have been released.
func (s *Store) Release(label string) {
	s.Lock()
	defer s.Unlock()

	context, err := selinux.NewContext(label)
	if err != nil {
		return
	}

	level := context["level"]
	if level == "" {
		return
	}

	count, ok := s.levels[level]
	if !ok {
		return
	}
	switch {
	case count == 1:
		s.Releaser(label)
		delete(s.levels, level)
	case count < 1:
		delete(s.levels, level)
	case count > 1:
		s.levels[level] = count - 1
	}
}
