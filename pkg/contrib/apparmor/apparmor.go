package apparmor

import (
	"context"
	"fmt"
	"os"

	"demo/over/containers"
	"demo/pkg/oci"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// WithProfile sets the provided apparmor profile to the spec
func WithProfile(profile string) oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		s.Process.ApparmorProfile = profile
		return nil
	}
}

// WithDefaultProfile will generate a default apparmor profile under the provided name
// for the container.  It is only generated if a profile under that name does not exist.
//
// FIXME: pkg/cri/[sb]server/container_create_linux_test.go depends on go:noinline
// since Go 1.21.
//
//go:noinline
func WithDefaultProfile(name string) oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		if err := LoadDefaultProfile(name); err != nil {
			return err
		}
		s.Process.ApparmorProfile = name
		return nil
	}
}

// LoadDefaultProfile ensures the default profile to be loaded with the given name.
// Returns nil error if the profile is already loaded.
func LoadDefaultProfile(name string) error {
	yes, err := isLoaded(name)
	if err != nil {
		return err
	}
	if yes {
		return nil
	}
	p, err := loadData(name)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(os.Getenv("XDG_RUNTIME_DIR"), p.Name)
	if err != nil {
		return err
	}
	defer f.Close()
	path := f.Name()
	defer os.Remove(path)

	if err := generate(p, f); err != nil {
		return err
	}
	if err := load(path); err != nil {
		return fmt.Errorf("load apparmor profile %s: %w", path, err)
	}
	return nil
}
