package oci

import (
	"context"
	"demo/pkg/namespaces"
	"encoding/json"
	"github.com/opencontainers/runtime-spec/specs-go"
	"os"
	"path/filepath"

	"demo/pkg/containers"
	"demo/pkg/platforms"
)

const (
	rwm               = "rwm"
	defaultRootfsPath = "rootfs"
)

var (
	defaultUnixEnv = []string{
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
	}
)

// Spec is a type alias to the OCI runtime spec to allow third part SpecOpts
// to be created without the "issues" with go vendoring and package imports
type Spec = specs.Spec

const ConfigFilename = "config.json"

// ReadSpec deserializes JSON into an OCI runtime Spec from a given path.
func ReadSpec(path string) (*Spec, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var s Spec
	if err := json.NewDecoder(f).Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

func GenerateSpec(ctx context.Context, client Client, c *containers.Container, opts ...SpecOpts) (*Spec, error) {
	return GenerateSpecWithPlatform(ctx, client, platforms.DefaultString(), c, opts...)
}

// GenerateSpecWithPlatform will generate a default spec from the provided image
// for use as a containerd container in the platform requested.
func GenerateSpecWithPlatform(ctx context.Context, client Client, platform string, c *containers.Container, opts ...SpecOpts) (*Spec, error) {
	var s Spec
	if err := generateDefaultSpecWithPlatform(ctx, platform, c.ID, &s); err != nil {
		return nil, err
	}

	return &s, ApplyOpts(ctx, client, c, &s, opts...)
}

func generateDefaultSpecWithPlatform(ctx context.Context, platform, id string, s *Spec) error {
	return populateDefaultUnixSpec(ctx, s, id)
}

// ApplyOpts applies the options to the given spec, injecting data from the
// context, client and container instance.
func ApplyOpts(ctx context.Context, client Client, c *containers.Container, s *Spec, opts ...SpecOpts) error {
	for _, o := range opts {
		if err := o(ctx, client, c, s); err != nil {
			return err
		}
	}

	return nil
}

func defaultUnixCaps() []string {
	return []string{
		"CAP_CHOWN",
		"CAP_DAC_OVERRIDE",
		"CAP_FSETID",
		"CAP_FOWNER",
		"CAP_MKNOD",
		"CAP_NET_RAW",
		"CAP_SETGID",
		"CAP_SETUID",
		"CAP_SETFCAP",
		"CAP_SETPCAP",
		"CAP_NET_BIND_SERVICE",
		"CAP_SYS_CHROOT",
		"CAP_KILL",
		"CAP_AUDIT_WRITE",
	}
}

func defaultUnixNamespaces() []specs.LinuxNamespace {
	return []specs.LinuxNamespace{
		{
			Type: specs.PIDNamespace,
		},
		{
			Type: specs.IPCNamespace,
		},
		{
			Type: specs.UTSNamespace,
		},
		{
			Type: specs.MountNamespace,
		},
		{
			Type: specs.NetworkNamespace,
		},
	}
}

func populateDefaultUnixSpec(ctx context.Context, s *Spec, id string) error {
	ns, err := namespaces.NamespaceRequired(ctx)
	if err != nil {
		return err
	}

	*s = Spec{
		Version: specs.Version,
		Root: &specs.Root{
			Path: defaultRootfsPath,
		},
		Process: &specs.Process{
			Cwd:             "/",
			NoNewPrivileges: true,
			User: specs.User{
				UID: 0,
				GID: 0,
			},
			Capabilities: &specs.LinuxCapabilities{
				Bounding:  defaultUnixCaps(), // 集合
				Permitted: defaultUnixCaps(), // 线程
				Effective: defaultUnixCaps(), // 进程
			},
			Rlimits: []specs.POSIXRlimit{
				{
					Type: "RLIMIT_NOFILE",
					Hard: uint64(1024),
					Soft: uint64(1024),
				},
			},
		},
		Linux: &specs.Linux{
			MaskedPaths: []string{
				"/proc/acpi",
				"/proc/asound",
				"/proc/kcore",
				"/proc/keys",
				"/proc/latency_stats",
				"/proc/timer_list",
				"/proc/timer_stats",
				"/proc/sched_debug",
				"/sys/firmware",
				"/sys/devices/virtual/powercap",
				"/proc/scsi",
			},
			ReadonlyPaths: []string{
				"/proc/bus",
				"/proc/fs",
				"/proc/irq",
				"/proc/sys",
				"/proc/sysrq-trigger",
			},
			CgroupsPath: filepath.Join("/", ns, id),
			Resources: &specs.LinuxResources{
				Devices: []specs.LinuxDeviceCgroup{
					{
						Allow:  false,
						Access: rwm,
					},
				},
			},
			Namespaces: defaultUnixNamespaces(),
		},
	}
	s.Mounts = defaultMounts()
	return nil
}
