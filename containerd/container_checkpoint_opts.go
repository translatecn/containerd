package containerd

import (
	"bytes"
	"context"
	"demo/config/runc"
	"demo/over/protobuf"
	"demo/over/protobuf/proto"
	"demo/over/rootfs"
	"errors"
	"fmt"
	"runtime"

	tasks "demo/over/api/services/tasks/v1"
	"demo/over/containers"
	"demo/over/diff"
	"demo/over/images"
	"demo/over/platforms"
	"github.com/opencontainers/go-digest"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
)

var (
	// ErrCheckpointRWUnsupported is returned if the container runtime does not support checkpoint
	ErrCheckpointRWUnsupported = errors.New("rw checkpoint is only supported on v2 runtimes")
	// ErrMediaTypeNotFound returns an error when a media type in the manifest is unknown
	ErrMediaTypeNotFound = errors.New("media type not found")
)

// CheckpointOpts are options to manage the checkpoint operation
type CheckpointOpts func(context.Context, *Client, *containers.Container, *imagespec.Index, *runc.CheckpointOptions) error

// WithCheckpointImage includes the container image in the checkpoint
func WithCheckpointImage(ctx context.Context, client *Client, c *containers.Container, index *imagespec.Index, copts *runc.CheckpointOptions) error {
	ir, err := client.ImageService().Get(ctx, c.Image)
	if err != nil {
		return err
	}
	index.Manifests = append(index.Manifests, ir.Target)
	return nil
}

// WithCheckpointTask includes the running task
func WithCheckpointTask(ctx context.Context, client *Client, c *containers.Container, index *imagespec.Index, copts *runc.CheckpointOptions) error {
	any, err := protobuf.MarshalAnyToProto(copts)
	if err != nil {
		return nil
	}
	task, err := client.TaskService().Checkpoint(ctx, &tasks.CheckpointTaskRequest{
		ContainerID: c.ID,
		Options:     any,
	})
	if err != nil {
		return err
	}
	for _, d := range task.Descriptors {
		platformSpec := platforms.DefaultSpec()
		index.Manifests = append(index.Manifests, imagespec.Descriptor{
			MediaType:   d.MediaType,
			Size:        d.Size,
			Digest:      digest.Digest(d.Digest),
			Platform:    &platformSpec,
			Annotations: d.Annotations,
		})
	}
	// save copts
	data, err := proto.Marshal(any)
	if err != nil {
		return err
	}
	r := bytes.NewReader(data)
	desc, err := writeContent(ctx, client.ContentStore(), images.MediaTypeContainerd1CheckpointOptions, c.ID+"-checkpoint-options", r)
	if err != nil {
		return err
	}
	desc.Platform = &imagespec.Platform{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
	}
	index.Manifests = append(index.Manifests, desc)
	return nil
}

// WithCheckpointRuntime includes the container runtime info
func WithCheckpointRuntime(ctx context.Context, client *Client, c *containers.Container, index *imagespec.Index, copts *runc.CheckpointOptions) error {
	if c.Runtime.Options != nil && c.Runtime.Options.GetValue() != nil {
		any := protobuf.FromAny(c.Runtime.Options)
		data, err := proto.Marshal(any)
		if err != nil {
			return err
		}
		r := bytes.NewReader(data)
		desc, err := writeContent(ctx, client.ContentStore(), images.MediaTypeContainerd1CheckpointRuntimeOptions, c.ID+"-runtime-options", r)
		if err != nil {
			return err
		}
		desc.Platform = &imagespec.Platform{
			OS:           runtime.GOOS,
			Architecture: runtime.GOARCH,
		}
		index.Manifests = append(index.Manifests, desc)
	}
	return nil
}

// WithCheckpointRW includes the rw in the checkpoint
func WithCheckpointRW(ctx context.Context, client *Client, c *containers.Container, index *imagespec.Index, copts *runc.CheckpointOptions) error {
	diffOpts := []diff.Opt{
		diff.WithReference(fmt.Sprintf("checkpoint-rw-%s", c.SnapshotKey)),
	}
	rw, err := rootfs.CreateDiff(ctx,
		c.SnapshotKey,
		client.SnapshotService(c.Snapshotter),
		client.DiffService(),
		diffOpts...,
	)
	if err != nil {
		return err

	}
	rw.Platform = &imagespec.Platform{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
	}
	index.Manifests = append(index.Manifests, rw)
	return nil
}

// WithCheckpointTaskExit causes the task to exit after checkpoint

// GetIndexByMediaType returns the index in a manifest for the specified media type
func GetIndexByMediaType(index *imagespec.Index, mt string) (*imagespec.Descriptor, error) {
	for _, d := range index.Manifests {
		if d.MediaType == mt {
			return &d, nil
		}
	}
	return nil, ErrMediaTypeNotFound
}
