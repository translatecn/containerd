package sandbox

import (
	"context"
	"demo/pkg/typeurl/v2"
	"fmt"
	"time"

	"demo/pkg/errdefs"
)

// Sandbox is an object stored in metadata database
type Sandbox struct {
	// ID uniquely identifies the sandbox in a namespace
	ID string
	// Labels provide metadata extension for a sandbox
	Labels map[string]string
	// Runtime shim to use for this sandbox
	Runtime RuntimeOpts
	// Spec carries the runtime specification used to implement the sandbox
	Spec typeurl.Any
	// CreatedAt is the time at which the sandbox was created
	CreatedAt time.Time
	// UpdatedAt is the time at which the sandbox was updated
	UpdatedAt time.Time
	// Extensions stores client-specified metadata
	Extensions map[string]typeurl.Any
}

// RuntimeOpts holds runtime specific information
type RuntimeOpts struct {
	Name    string
	Options typeurl.Any
}

// Store is a storage interface for sandbox metadata objects
type Store interface {
	Create(ctx context.Context, sandbox Sandbox) (Sandbox, error)

	// Update the sandbox with the provided sandbox object and fields
	Update(ctx context.Context, sandbox Sandbox, fieldpaths ...string) (Sandbox, error)

	// Get sandbox metadata using the id
	Get(ctx context.Context, id string) (Sandbox, error)

	// List returns sandboxes that match one or more of the provided filters
	List(ctx context.Context, filters ...string) ([]Sandbox, error)

	// Delete a sandbox from metadata store using the id
	Delete(ctx context.Context, id string) error
}

// AddExtension is a helper function to add sandbox metadata extension.
func (s *Sandbox) AddExtension(name string, obj interface{}) error {
	if s.Extensions == nil {
		s.Extensions = map[string]typeurl.Any{}
	}

	out, err := typeurl.MarshalAny(obj)
	if err != nil {
		return fmt.Errorf("failed to marshal sandbox extension %q: %w", name, err)
	}

	s.Extensions[name] = out
	return nil
}

// AddLabel adds a label to sandbox's labels.
func (s *Sandbox) AddLabel(name string, value string) {
	if s.Labels == nil {
		s.Labels = map[string]string{}
	}

	s.Labels[name] = value
}

// GetExtension retrieves a sandbox extension by name.
func (s *Sandbox) GetExtension(name string, obj interface{}) error {
	out, ok := s.Extensions[name]
	if !ok {
		return errdefs.ErrNotFound
	}

	if err := typeurl.UnmarshalTo(out, obj); err != nil {
		return fmt.Errorf("failed to unmarshal sandbox extension %q: %w", name, err)
	}

	return nil
}

// GetLabel retrieves a sandbox label by name.
func (s *Sandbox) GetLabel(name string) (string, error) {
	out, ok := s.Labels[name]
	if !ok {
		return "", fmt.Errorf("unable to find label %q in sandbox metadata: %w", name, errdefs.ErrNotFound)
	}

	return out, nil
}
