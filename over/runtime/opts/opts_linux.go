package opts

import (
	"context"
	"demo/others/cgroups/v3"
	cgroup1 "demo/others/cgroups/v3/cgroup1"
	cgroup2 "demo/others/cgroups/v3/cgroup2"
	"demo/over/namespaces"
)

// WithNamespaceCgroupDeletion removes the cgroup directory that was created for the namespace
func WithNamespaceCgroupDeletion(ctx context.Context, i *namespaces.DeleteInfo) error {
	if cgroups.Mode() == cgroups.Unified { // v2
		cg, err := cgroup2.Load(i.Name)
		if err != nil {
			return err
		}
		return cg.Delete()
	}
	cg, err := cgroup1.Load(cgroup1.StaticPath(i.Name))
	if err != nil {
		if err == cgroup1.ErrCgroupDeleted {
			return nil
		}
		return err
	}
	return cg.Delete()
}
