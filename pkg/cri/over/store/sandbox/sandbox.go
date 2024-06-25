package sandbox

import (
	"demo/over/netns"
	"demo/pkg/cri/over/store"
	"demo/pkg/cri/over/store/label"
	"demo/pkg/cri/over/store/stats"
	"sync"

	"demo/containerd"
	"demo/over/errdefs"
	"demo/over/truncindex"
)

// Sandbox contains all resources associated with the sandbox. All methods to
// mutate the internal state are thread safe.
type Sandbox struct {
	// Metadata is the metadata of the sandbox, it is immutable after created.
	Metadata
	// Status stores the status of the sandbox.
	Status StatusStorage
	// Container is the containerd sandbox container client.
	Container containerd.Container // pause container
	// CNI network namespace client.
	// For hostnetwork pod, this is always nil;
	// For non hostnetwork pod, this should never be nil.
	NetNS *netns.NetNS
	// StopCh is used to propagate the stop information of the sandbox.
	*store.StopCh
	// Stats contains (mutable) stats for the (pause) sandbox container
	Stats *stats.ContainerStats
}

// NewSandbox creates an internally used sandbox type. This functions reminds
// the caller that a sandbox must have a status.
func NewSandbox(metadata Metadata, status Status) Sandbox {
	s := Sandbox{
		Metadata: metadata,
		Status:   StoreStatus(status),
		StopCh:   store.NewStopCh(),
	}
	if status.State == StateNotReady {
		s.Stop()
	}
	return s
}

// Store stores all sandboxes.
type Store struct {
	lock      sync.RWMutex
	sandboxes map[string]Sandbox
	idIndex   *truncindex.TruncIndex
	labels    *label.Store
}

// NewStore creates a sandbox store.
func NewStore(labels *label.Store) *Store {
	return &Store{
		sandboxes: make(map[string]Sandbox),
		idIndex:   truncindex.NewTruncIndex([]string{}),
		labels:    labels,
	}
}

// List lists all sandboxes.
func (s *Store) List() []Sandbox {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var sandboxes []Sandbox
	for _, sb := range s.sandboxes {
		sandboxes = append(sandboxes, sb)
	}
	return sandboxes
}

// UpdateContainerStats updates the sandbox specified by ID with the
// stats present in 'newContainerStats'. Returns errdefs.ErrNotFound
// if the sandbox does not exist in the store.
func (s *Store) UpdateContainerStats(id string, newContainerStats *stats.ContainerStats) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	id, err := s.idIndex.Get(id)
	if err != nil {
		if err == truncindex.ErrNotExist {
			err = errdefs.ErrNotFound
		}
		return err
	}

	if _, ok := s.sandboxes[id]; !ok {
		return errdefs.ErrNotFound
	}

	c := s.sandboxes[id]
	c.Stats = newContainerStats
	s.sandboxes[id] = c
	return nil
}

// Delete deletes the sandbox with specified id.
func (s *Store) Delete(id string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	id, err := s.idIndex.Get(id)
	if err != nil {
		// Note: The idIndex.Delete and delete doesn't handle truncated index.
		// So we need to return if there are error.
		return
	}
	s.labels.Release(s.sandboxes[id].ProcessSelinuxLabel)
	s.idIndex.Delete(id)
	delete(s.sandboxes, id)
}

// Add a sandbox into the store. Returns errdefs.ErrAlreadyExists if the sandbox is
// already stored.
func (s *Store) Add(sb Sandbox) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, ok := s.sandboxes[sb.ID]; ok {
		return errdefs.ErrAlreadyExists
	}
	if err := s.labels.Reserve(sb.ProcessSelinuxLabel); err != nil {
		return err
	}
	if err := s.idIndex.Add(sb.ID); err != nil {
		return err
	}
	s.sandboxes[sb.ID] = sb
	return nil
}

// Get returns the sandbox with specified id.
// Returns errdefs.ErrNotFound if the sandbox doesn't exist.
func (s *Store) Get(id string) (Sandbox, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	id, err := s.idIndex.Get(id)
	if err != nil {
		if err == truncindex.ErrNotExist {
			err = errdefs.ErrNotFound
		}
		return Sandbox{}, err
	}
	if sb, ok := s.sandboxes[id]; ok {
		return sb, nil
	}
	return Sandbox{}, errdefs.ErrNotFound
}
