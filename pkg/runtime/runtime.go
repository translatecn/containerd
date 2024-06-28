package runtime

import (
	"context"
	"demo/pkg/typeurl/v2"
	"time"

	"demo/pkg/mount"
)

// IO holds process IO information
type IO struct {
	Stdin    string
	Stdout   string
	Stderr   string
	Terminal bool
}

// CreateOpts contains task creation data
type CreateOpts struct {
	// Spec is the OCI runtime spec
	Spec typeurl.Any
	// Rootfs mounts to perform to gain access to the container's filesystem
	Rootfs []mount.Mount
	// IO for the container's main process
	IO IO
	// Checkpoint digest to restore container state
	Checkpoint string
	// RuntimeOptions for the runtime
	RuntimeOptions typeurl.Any
	// TaskOptions received for the task
	TaskOptions typeurl.Any
	// Runtime name to use (e.g. `io.containerd.NAME.VERSION`).
	// As an alternative full abs path to binary may be specified instead.
	Runtime string
	// SandboxID is an optional ID of sandbox this container belongs to
	SandboxID string
}

// Exit information for a process
type Exit struct {
	Pid       uint32
	Status    uint32
	Timestamp time.Time
}

// PlatformRuntime is responsible for the creation and management of
// tasks and processes for a platform.
type PlatformRuntime interface {
	// ID of the runtime
	ID() string
	Create(ctx context.Context, taskID string, opts CreateOpts) (Task, error)
	// Get returns a task.
	Get(ctx context.Context, taskID string) (Task, error)
	// Tasks returns all the current tasks for the runtime.
	// Any container runs at most one task at a time.
	Tasks(ctx context.Context, all bool) ([]Task, error)
	// Delete remove a task.
	Delete(ctx context.Context, taskID string) (*Exit, error)
}
