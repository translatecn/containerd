package nvidia

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"demo/over/containers"
	"demo/over/oci"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// NvidiaCLI is the path to the Nvidia helper binary
const NvidiaCLI = "nvidia-container-cli"

// Capability specifies capabilities for the gpu inside the container
// Detailed explanation of options can be found:
// https://github.com/nvidia/nvidia-container-runtime#supported-driver-capabilities
type Capability string

const (
	// Compute capability
	Compute Capability = "compute"
	// Compat32 capability
	Compat32 Capability = "compat32"
	// Graphics capability
	Graphics Capability = "graphics"
	// Utility capability
	Utility Capability = "utility"
	// Video capability
	Video Capability = "video"
	// Display capability
	Display Capability = "display"
)

// AllCaps returns the complete list of supported Nvidia capabilities.
func AllCaps() []Capability {
	return []Capability{
		Compute,
		Compat32,
		Graphics,
		Utility,
		Video,
		Display,
	}
}

// WithGPUs adds NVIDIA gpu support to a container
func WithGPUs(opts ...Opts) oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		c := &config{}
		for _, o := range opts {
			if err := o(c); err != nil {
				return err
			}
		}
		if c.OCIHookPath == "" {
			path, err := exec.LookPath("containerd")
			if err != nil {
				return err
			}
			c.OCIHookPath = path
		}
		nvidiaPath, err := exec.LookPath(NvidiaCLI)
		if err != nil {
			return err
		}
		if s.Hooks == nil {
			s.Hooks = &specs.Hooks{}
		}
		s.Hooks.Prestart = append(s.Hooks.Prestart, specs.Hook{
			Path: c.OCIHookPath,
			Args: append([]string{
				"containerd",
				"oci-hook",
				"--",
				nvidiaPath,
				// ensures the required kernel modules are properly loaded
				"--load-kmods",
			}, c.args()...),
			Env: os.Environ(),
		})
		return nil
	}
}

type config struct {
	Devices      []string
	Capabilities []Capability
	LoadKmods    bool
	LDCache      string
	LDConfig     string
	Requirements []string
	OCIHookPath  string
	NoCgroups    bool
}

func (c *config) args() []string {
	var args []string

	if c.LoadKmods {
		args = append(args, "--load-kmods")
	}
	if c.LDCache != "" {
		args = append(args, fmt.Sprintf("--ldcache=%s", c.LDCache))
	}
	args = append(args,
		"configure",
	)
	if len(c.Devices) > 0 {
		args = append(args, fmt.Sprintf("--device=%s", strings.Join(c.Devices, ",")))
	}
	for _, c := range c.Capabilities {
		args = append(args, fmt.Sprintf("--%s", c))
	}
	if c.LDConfig != "" {
		args = append(args, fmt.Sprintf("--ldconfig=%s", c.LDConfig))
	}
	for _, r := range c.Requirements {
		args = append(args, fmt.Sprintf("--require=%s", r))
	}
	if c.NoCgroups {
		args = append(args, "--no-cgroups")
	}
	args = append(args, "--pid={{pid}}", "{{rootfs}}")
	return args
}

// Opts are options for configuring gpu support
type Opts func(*config) error

// WithDevices adds the provided device indexes to the container
func WithDevices(ids ...int) Opts {
	return func(c *config) error {
		for _, i := range ids {
			c.Devices = append(c.Devices, strconv.Itoa(i))
		}
		return nil
	}
}

// WithAllCapabilities adds all capabilities to the container for the gpus
func WithAllCapabilities(c *config) error {
	c.Capabilities = AllCaps()
	return nil
}
