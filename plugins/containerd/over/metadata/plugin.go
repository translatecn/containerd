package metadata

import (
	"demo/over/log"
	"demo/over/metadata"
	"demo/over/my_mk"
	"demo/over/plugin"
	"demo/over/snapshots"
	"demo/over/timeout"
	"fmt"
	"path/filepath"
	"time"

	"demo/over/content"
	"demo/over/errdefs"
	bolt "go.etcd.io/bbolt"
)

const (
	boltOpenTimeout = "io.containerd.timeout.bolt.open"
)

func init() {
	timeout.Set(boltOpenTimeout, 0) // set to 0 means to wait indefinitely for bolt.Open
}

// BoltConfig defines the configuration values for the bolt plugin, which is
// loaded here, rather than back registered in the metadata package.
type BoltConfig struct {
	// ContentSharingPolicy 命名空间之间的内容共享策略
	ContentSharingPolicy string `toml:"content_sharing_policy"`
}

const (
	// SharingPolicyShared represents the "shared" sharing policy
	SharingPolicyShared = "shared"
	// SharingPolicyIsolated represents the "isolated" sharing policy
	SharingPolicyIsolated = "isolated"
)

// Validate validates if BoltConfig is valid
func (bc *BoltConfig) Validate() error {
	switch bc.ContentSharingPolicy {
	case SharingPolicyShared, SharingPolicyIsolated:
		return nil
	default:
		return fmt.Errorf("unknown policy: %s: %w", bc.ContentSharingPolicy, errdefs.ErrInvalidArgument)
	}
}

func init() {
	plugin.Register(&plugin.Registration{
		Type: plugin.MetadataPlugin,
		ID:   "bolt",
		Requires: []plugin.Type{
			plugin.ContentPlugin,
			plugin.SnapshotPlugin,
		},
		Config: &BoltConfig{
			ContentSharingPolicy: SharingPolicyShared,
		},
		InitFn: func(ic *plugin.InitContext) (interface{}, error) {
			if err := my_mk.MkdirAll(ic.Root, 0711); err != nil {
				return nil, err // /var/lib/containerd/io.containerd.metadata.v1.bolt
			}
			cs, err := ic.Get(plugin.ContentPlugin)
			if err != nil {
				return nil, err
			}

			snapshottersRaw, err := ic.GetByType(plugin.SnapshotPlugin)
			if err != nil {
				return nil, err
			}

			snapshotters := make(map[string]snapshots.Snapshotter)
			for name, snapshotPlugin := range snapshottersRaw {
				sn, err := snapshotPlugin.Instance()
				if err != nil {
					if !plugin.IsSkipPlugin(err) {
						log.G(ic.Context).WithError(err).Warnf("could not use snapshotter %v in metadata plugin", name)
					}
					continue
				}
				snapshotters[name] = sn.(snapshots.Snapshotter)
			}

			shared := true
			ic.Meta.Exports["policy"] = SharingPolicyShared
			if cfg, ok := ic.Config.(*BoltConfig); ok {
				if cfg.ContentSharingPolicy != "" {
					if err := cfg.Validate(); err != nil {
						return nil, err
					}
					if cfg.ContentSharingPolicy == SharingPolicyIsolated {
						ic.Meta.Exports["policy"] = SharingPolicyIsolated
						shared = false
					}

					log.G(ic.Context).WithField("policy", cfg.ContentSharingPolicy).Info("metadata content store policy set")
				}
			}
			// /var/lib/containerd/io.containerd.metadata.v1.bolt/meta.db
			path := filepath.Join(ic.Root, "meta.db")
			ic.Meta.Exports["path"] = path

			options := *bolt.DefaultOptions
			// Reading bbolt's freelist sometimes fails when the file has a data corruption.
			// Disabling freelist sync reduces the chance of the breakage.
			// https://github.com/etcd-io/bbolt/pull/1
			// https://github.com/etcd-io/bbolt/pull/6
			options.NoFreelistSync = true
			// Without the timeout, bbolt.Open would block indefinitely due to flock(2).
			options.Timeout = timeout.Get(boltOpenTimeout)

			doneCh := make(chan struct{})
			go func() {
				t := time.NewTimer(10 * time.Second)
				defer t.Stop()
				select {
				case <-t.C:
					log.G(ic.Context).WithField("plugin", "bolt").Warn("waiting for response from boltdb open")
				case <-doneCh:
					return
				}
			}()
			db, err := bolt.Open(path, 0644, &options)
			close(doneCh)
			if err != nil {
				return nil, err
			}

			dbopts := []metadata.DBOpt{
				metadata.WithEventsPublisher(ic.Events),
			}

			if !shared {
				dbopts = append(dbopts, metadata.WithPolicyIsolated)
			}

			mdb := metadata.NewDB(db, cs.(content.Store), snapshotters, dbopts...)
			if err := mdb.Init(ic.Context); err != nil {
				return nil, err
			}
			return mdb, nil
		},
	})
}
