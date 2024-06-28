package overlay

import (
	"demo/pkg/plugin"
	"demo/pkg/snapshots/overlay"
	"errors"

	"demo/pkg/platforms"
)

// Config represents configuration for the overlay plugin.
type Config struct {
	// Root directory for the plugin
	RootPath      string `toml:"root_path"`
	UpperdirLabel bool   `toml:"upperdir_label"`
	SyncRemove    bool   `toml:"sync_remove"`

	// MountOptions are options used for the overlay mount (not used on bind mounts)
	MountOptions []string `toml:"mount_options"`
}

func init() {
	plugin.Register(&plugin.Registration{
		Type:   plugin.SnapshotPlugin,
		ID:     "overlayfs",
		Config: &Config{},
		InitFn: func(ic *plugin.InitContext) (interface{}, error) {
			ic.Meta.Platforms = append(ic.Meta.Platforms, platforms.DefaultSpec())

			config, ok := ic.Config.(*Config)
			if !ok {
				return nil, errors.New("invalid overlay configuration")
			}

			root := ic.Root // /var/lib/containerd/io.containerd.snapshotter.v1.overlayfs
			if config.RootPath != "" {
				root = config.RootPath
			}

			var oOpts []overlay.Opt
			if config.UpperdirLabel {
				oOpts = append(oOpts, overlay.WithUpperdirLabel)
			}
			if !config.SyncRemove {
				oOpts = append(oOpts, overlay.AsynchronousRemove)
			}

			if len(config.MountOptions) > 0 {
				oOpts = append(oOpts, overlay.WithMountOptions(config.MountOptions))
			}
			// /var/lib/containerd/io.containerd.snapshotter.v1.overlayfs
			ic.Meta.Exports[plugin.SnapshotterRootDir] = root
			return overlay.NewSnapshotter(root, oOpts...)
		},
	})
}
