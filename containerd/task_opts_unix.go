package containerd

import (
	"context"
	"demo/config/runc"
	"demo/over/api/runctypes"
	"errors"
)

// WithNoPivotRoot instructs the runtime not to you pivot_root
func WithNoPivotRoot(_ context.Context, _ *Client, ti *TaskInfo) error {
	if CheckRuntime(ti.Runtime(), "io.containerd.runc") {
		if ti.Options == nil {
			ti.Options = &runc.Options{}
		}
		opts, ok := ti.Options.(*runc.Options)
		if !ok {
			return errors.New("invalid v2 shim create options format")
		}
		opts.NoPivotRoot = true
	} else {
		if ti.Options == nil {
			ti.Options = &runctypes.CreateOptions{
				NoPivotRoot: true,
			}
			return nil
		}
		opts, ok := ti.Options.(*runctypes.CreateOptions)
		if !ok {
			return errors.New("invalid options type, expected runctypes.CreateOptions")
		}
		opts.NoPivotRoot = true
	}
	return nil
}

// WithUIDOwner allows console I/O to work with the remapped UID in user namespace
func WithUIDOwner(uid uint32) NewTaskOpts {
	return func(ctx context.Context, c *Client, ti *TaskInfo) error {
		if CheckRuntime(ti.Runtime(), "io.containerd.runc") {
			if ti.Options == nil {
				ti.Options = &runc.Options{}
			}
			opts, ok := ti.Options.(*runc.Options)
			if !ok {
				return errors.New("invalid v2 shim create options format")
			}
			opts.IoUid = uid
		} else {
			if ti.Options == nil {
				ti.Options = &runctypes.CreateOptions{}
			}
			opts, ok := ti.Options.(*runctypes.CreateOptions)
			if !ok {
				return errors.New("could not cast TaskInfo Options to CreateOptions")
			}
			opts.IoUid = uid
		}
		return nil
	}
}

// WithGIDOwner allows console I/O to work with the remapped GID in user namespace
func WithGIDOwner(gid uint32) NewTaskOpts {
	return func(ctx context.Context, c *Client, ti *TaskInfo) error {
		if CheckRuntime(ti.Runtime(), "io.containerd.runc") {
			if ti.Options == nil {
				ti.Options = &runc.Options{}
			}
			opts, ok := ti.Options.(*runc.Options)
			if !ok {
				return errors.New("invalid v2 shim create options format")
			}
			opts.IoGid = gid
		} else {
			if ti.Options == nil {
				ti.Options = &runctypes.CreateOptions{}
			}
			opts, ok := ti.Options.(*runctypes.CreateOptions)
			if !ok {
				return errors.New("could not cast TaskInfo Options to CreateOptions")
			}
			opts.IoGid = gid
		}
		return nil
	}
}
