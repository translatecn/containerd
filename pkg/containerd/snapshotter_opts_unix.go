package containerd

import (
	"context"
	"demo/pkg/snapshots"
	"fmt"
)

const (
	capabRemapIDs = "remap-ids"
)

// WithRemapperLabels creates the labels used by any supporting snapshotter
// to shift the filesystem ownership (user namespace mapping) automatically; currently
// supported by the fuse-overlayfs snapshotter
func WithRemapperLabels(ctrUID, hostUID, ctrGID, hostGID, length uint32) snapshots.Opt {
	return snapshots.WithLabels(map[string]string{
		snapshots.LabelSnapshotUIDMapping: fmt.Sprintf("%d:%d:%d", ctrUID, hostUID, length),
		snapshots.LabelSnapshotGIDMapping: fmt.Sprintf("%d:%d:%d", ctrGID, hostGID, length)})
}

func resolveSnapshotOptions(ctx context.Context, client *Client, snapshotterName string, snapshotter snapshots.Snapshotter, parent string, opts ...snapshots.Opt) (string, error) {
	capabs, err := client.GetSnapshotterCapabilities(ctx, snapshotterName)
	if err != nil {
		return "", err
	}

	for _, capab := range capabs {
		if capab == capabRemapIDs {
			// Snapshotter supports ID remapping, we don't need to do anything.
			return parent, nil
		}
	}

	var local snapshots.Info
	for _, opt := range opts {
		opt(&local)
	}

	needsRemap := false
	var uidMap, gidMap string

	if value, ok := local.Labels[snapshots.LabelSnapshotUIDMapping]; ok {
		needsRemap = true
		uidMap = value
	}
	if value, ok := local.Labels[snapshots.LabelSnapshotGIDMapping]; ok {
		needsRemap = true
		gidMap = value
	}

	if !needsRemap {
		return parent, nil
	}

	var ctrUID, hostUID, length uint32
	_, err = fmt.Sscanf(uidMap, "%d:%d:%d", &ctrUID, &hostUID, &length)
	if err != nil {
		return "", fmt.Errorf("uidMap unparsable: %w", err)
	}

	var ctrGID, hostGID, lengthGID uint32
	_, err = fmt.Sscanf(gidMap, "%d:%d:%d", &ctrGID, &hostGID, &lengthGID)
	if err != nil {
		return "", fmt.Errorf("gidMap unparsable: %w", err)
	}

	if ctrUID != 0 || ctrGID != 0 {
		return "", fmt.Errorf("Container UID/GID of 0 only supported currently (%d/%d)", ctrUID, ctrGID)
	}

	// TODO(dgl): length isn't taken into account for the intermediate snapshot randomId.
	usernsID := fmt.Sprintf("%s-%d-%d", parent, hostUID, hostGID)
	if _, err := snapshotter.Stat(ctx, usernsID); err == nil {
		return usernsID, nil
	}
	mounts, err := snapshotter.Prepare(ctx, usernsID+"-remap", parent)
	if err != nil {
		return "", err
	}
	// TODO(dgl): length isn't taken into account here yet either.
	if err := remapRootFS(ctx, mounts, hostUID, hostGID); err != nil {
		snapshotter.Remove(ctx, usernsID+"-remap")
		return "", err
	}
	if err := snapshotter.Commit(ctx, usernsID, usernsID+"-remap"); err != nil {
		return "", err
	}

	return usernsID, nil
}
