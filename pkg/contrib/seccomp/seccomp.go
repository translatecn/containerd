package seccomp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"demo/pkg/containers"
	"demo/pkg/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
)

// WithProfile receives the name of a file stored on disk comprising a json
// formatted seccomp profile, as specified by the opencontainers/runtime-spec.
// The profile is read from the file, unmarshaled, and set to the spec.
//
// FIXME: pkg/cri/[sb]server/container_create_linux_test.go depends on go:noinline
// since Go 1.21.
//
//go:noinline
func WithProfile(profile string) oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		s.Linux.Seccomp = &specs.LinuxSeccomp{}
		f, err := os.ReadFile(profile)
		if err != nil {
			return fmt.Errorf("cannot load seccomp profile %q: %v", profile, err)
		}
		if err := json.Unmarshal(f, s.Linux.Seccomp); err != nil {
			return fmt.Errorf("decoding seccomp profile failed %q: %v", profile, err)
		}
		return nil
	}
}

// WithDefaultProfile sets the default seccomp profile to the spec.
// Note: must follow the setting of process capabilities
//
// FIXME: pkg/cri/[sb]server/container_create_linux_test.go depends on go:noinline
// since Go 1.21.
//
//go:noinline
func WithDefaultProfile() oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		s.Linux.Seccomp = DefaultProfile(s)
		return nil
	}
}
