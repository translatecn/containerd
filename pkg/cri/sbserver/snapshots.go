package sbserver

import (
	"context"
	snapshotstore "demo/pkg/cri/store/snapshot"
	ctrdutil "demo/pkg/cri/util"
	snapshot "demo/pkg/snapshots"
	"fmt"
	"time"

	"demo/pkg/errdefs"
	"github.com/sirupsen/logrus"
)

// snapshotsSyncer syncs snapshot stats periodically. imagefs info and container stats
// should both use cached result here.
// TODO(random-liu): Benchmark with high workload. We may need a statsSyncer instead if
// benchmark result shows that container cpu/memory stats also need to be cached.
type snapshotsSyncer struct {
	store       *snapshotstore.Store
	snapshotter snapshot.Snapshotter
	syncPeriod  time.Duration
}

// newSnapshotsSyncer creates a snapshot syncer.
func newSnapshotsSyncer(store *snapshotstore.Store, snapshotter snapshot.Snapshotter,
	period time.Duration) *snapshotsSyncer {
	return &snapshotsSyncer{
		store:       store,
		snapshotter: snapshotter,
		syncPeriod:  period,
	}
}

// start starts the snapshots syncer. No stop function is needed because
// the syncer doesn't update any persistent states, it's fine to let it
// exit with the process.
func (s *snapshotsSyncer) start() {
	tick := time.NewTicker(s.syncPeriod)
	go func() {
		defer tick.Stop()
		// TODO(random-liu): This is expensive. We should do benchmark to
		// check the resource usage and optimize this.
		for {
			if err := s.sync(); err != nil {
				logrus.WithError(err).Error("Failed to sync snapshot stats")
			}
			<-tick.C
		}
	}()
}

// sync updates all snapshots stats.
func (s *snapshotsSyncer) sync() error {
	ctx := ctrdutil.NamespacedContext()
	start := time.Now().UnixNano()
	var snapshots []snapshot.Info
	// Do not call `Usage` directly in collect function, because
	// `Usage` takes time, we don't want `Walk` to hold read lock
	// of snapshot metadata store for too long time.
	// TODO(random-liu): Set timeout for the following 2 contexts.
	if err := s.snapshotter.Walk(ctx, func(ctx context.Context, info snapshot.Info) error {
		snapshots = append(snapshots, info)
		return nil
	}); err != nil {
		return fmt.Errorf("walk all snapshots failed: %w", err)
	}
	for _, info := range snapshots {
		sn, err := s.store.Get(info.Name)
		if err == nil {
			// Only update timestamp for non-active snapshot.
			if sn.Kind == info.Kind && sn.Kind != snapshot.KindActive {
				sn.Timestamp = time.Now().UnixNano()
				s.store.Add(sn)
				continue
			}
		}
		// Get newest stats if the snapshot is new or active.
		sn = snapshotstore.Snapshot{
			Key:       info.Name,
			Kind:      info.Kind,
			Timestamp: time.Now().UnixNano(),
		}
		usage, err := s.snapshotter.Usage(ctx, info.Name)
		if err != nil {
			if !errdefs.IsNotFound(err) {
				logrus.WithError(err).Errorf("Failed to get usage for snapshot %q", info.Name)
			}
			continue
		}
		sn.Size = uint64(usage.Size)
		sn.Inodes = uint64(usage.Inodes)
		s.store.Add(sn)
	}
	for _, sn := range s.store.List() {
		if sn.Timestamp >= start {
			continue
		}
		// Delete the snapshot stats if it's not updated this time.
		s.store.Delete(sn.Key)
	}
	return nil
}
