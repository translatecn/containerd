package sbserver

import (
	"context"
	"time"

	runtime "demo/pkg/api/cri/v1"
)

// ImageFsInfo returns information of the filesystem that is used to store images.
// TODO(windows): Usage for windows is always 0 right now. Support this for windows.
func (c *CriService) ImageFsInfo(ctx context.Context, r *runtime.ImageFsInfoRequest) (*runtime.ImageFsInfoResponse, error) {
	snapshots := c.snapshotStore.List()
	timestamp := time.Now().UnixNano()
	var usedBytes, inodesUsed uint64
	for _, sn := range snapshots {
		// Use the oldest timestamp as the timestamp of imagefs info.
		if sn.Timestamp < timestamp {
			timestamp = sn.Timestamp
		}
		usedBytes += sn.Size
		inodesUsed += sn.Inodes
	}
	// TODO(random-liu): Handle content store
	return &runtime.ImageFsInfoResponse{
		ImageFilesystems: []*runtime.FilesystemUsage{
			{
				Timestamp:  timestamp,
				FsId:       &runtime.FilesystemIdentifier{Mountpoint: c.imageFSPath},
				UsedBytes:  &runtime.UInt64Value{Value: usedBytes},
				InodesUsed: &runtime.UInt64Value{Value: inodesUsed},
			},
		},
	}, nil
}
