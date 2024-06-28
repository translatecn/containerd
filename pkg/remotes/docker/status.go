package docker

import (
	"fmt"
	"sync"

	"demo/pkg/content"
	"demo/pkg/errdefs"
	"github.com/moby/locker"
)

// Status of a content operation
type Status struct {
	content.Status

	Committed bool

	// ErrClosed contains error encountered on close.
	ErrClosed error

	// UploadUUID is used by the Docker registry to reference blob uploads
	UploadUUID string

	// PushStatus contains status related to push.
	PushStatus
}

type PushStatus struct {
	// MountedFrom is the source content was cross-repo mounted from (empty if no cross-repo mount was performed).
	MountedFrom string

	// Exists indicates whether content already exists in the repository and wasn't uploaded.
	Exists bool
}

// StatusTracker to track status of operations
type StatusTracker interface {
	GetStatus(string) (Status, error)
	SetStatus(string, Status)
}

// StatusTrackLocker to track status of operations with lock
type StatusTrackLocker interface {
	StatusTracker
	Lock(string)
	Unlock(string)
}

type memoryStatusTracker struct {
	statuses map[string]Status
	m        sync.Mutex
	locker   *locker.Locker
}

// NewInMemoryTracker returns a StatusTracker that tracks content status in-memory
func NewInMemoryTracker() StatusTrackLocker {
	return &memoryStatusTracker{
		statuses: map[string]Status{},
		locker:   locker.New(),
	}
}

func (t *memoryStatusTracker) GetStatus(ref string) (Status, error) {
	t.m.Lock()
	defer t.m.Unlock()
	status, ok := t.statuses[ref]
	if !ok {
		return Status{}, fmt.Errorf("status for ref %v: %w", ref, errdefs.ErrNotFound)
	}
	return status, nil
}

func (t *memoryStatusTracker) SetStatus(ref string, status Status) {
	t.m.Lock()
	t.statuses[ref] = status
	t.m.Unlock()
}

func (t *memoryStatusTracker) Lock(ref string) {
	t.locker.Lock(ref)
}

func (t *memoryStatusTracker) Unlock(ref string) {
	t.locker.Unlock(ref)
}
