package content

import (
	"context"
	srvconfig "demo/config/server"
	"demo/others/cgroups/v3"
	"demo/pkg/log"
	"demo/pkg/sys"
	"os"

	cgroup1 "demo/others/cgroups/v3/cgroup1"
	cgroupsv2 "demo/others/cgroups/v3/cgroup2"
	"demo/pkg/ttrpc"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// apply sets config settings on the server process
func apply(ctx context.Context, config *srvconfig.Config) error {
	if config.OOMScore != 0 {
		log.G(ctx).Debugf("changing OOM score to %d", config.OOMScore)
		if err := sys.SetOOMScore(os.Getpid(), config.OOMScore); err != nil {
			log.G(ctx).WithError(err).Errorf("failed to change OOM score to %d", config.OOMScore)
		}
	}
	if config.Cgroup.Path != "" {
		if cgroups.Mode() == cgroups.Unified {
			cg, err := cgroupsv2.Load(config.Cgroup.Path)
			if err != nil {
				return err
			}
			if err := cg.AddProc(uint64(os.Getpid())); err != nil {
				return err
			}
		} else {
			cg, err := cgroup1.Load(cgroup1.StaticPath(config.Cgroup.Path))
			if err != nil {
				if err != cgroup1.ErrCgroupDeleted {
					return err
				}
				if cg, err = cgroup1.New(cgroup1.StaticPath(config.Cgroup.Path), &specs.LinuxResources{}); err != nil {
					return err
				}
			}
			if err := cg.AddProc(uint64(os.Getpid())); err != nil {
				return err
			}
		}
	}
	return nil
}

func newTTRPCServer() (*ttrpc.Server, error) {
	return ttrpc.NewServer(ttrpc.WithServerHandshaker(ttrpc.UnixSocketRequireSameUser()))
}
