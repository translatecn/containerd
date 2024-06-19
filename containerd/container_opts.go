package containerd

import (
	"context"
	"demo/over/namespaces"
	"demo/over/protobuf"
	"demo/over/snapshots"
	"demo/over/typeurl/v2"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/opencontainers/image-spec/identity"

	"demo/over/containers"
	"demo/over/content"
	"demo/over/errdefs"
	"demo/over/images"
	"demo/pkg/oci"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// DeleteOpts allows the caller to set options for the deletion of a container
type DeleteOpts func(ctx context.Context, client *Client, c containers.Container) error

// NewContainerOpts allows the caller to set additional options when creating a container
type NewContainerOpts func(ctx context.Context, client *Client, c *containers.Container) error

// UpdateContainerOpts allows the caller to set additional options when updating a container
type UpdateContainerOpts func(ctx context.Context, client *Client, c *containers.Container) error

// InfoOpts controls how container metadata is fetched and returned
type InfoOpts func(*InfoConfig)

// InfoConfig specifies how container metadata is fetched
type InfoConfig struct {
	// Refresh will to a fetch of the latest container metadata
	Refresh bool
}

// WithSandbox joins the container to a container group (aka sandbox) from the given ID
// Note: shim runtime must support sandboxes environments.
func WithSandbox(sandboxID string) NewContainerOpts {
	return func(ctx context.Context, client *Client, c *containers.Container) error {
		c.SandboxID = sandboxID
		return nil
	}
}

// WithImageConfigLabels sets the image config labels on the container.
// The existing labels are cleared as this is expected to be the first
// operation in setting up a container's labels. Use WithAdditionalContainerLabels
// to add/overwrite the existing image config labels.
func WithImageConfigLabels(image Image) NewContainerOpts {
	return func(ctx context.Context, _ *Client, c *containers.Container) error {
		ic, err := image.Config(ctx)
		if err != nil {
			return err
		}
		var (
			ociimage v1.Image
			config   v1.ImageConfig
		)
		switch ic.MediaType {
		case v1.MediaTypeImageConfig, images.MediaTypeDockerSchema2Config:
			p, err := content.ReadBlob(ctx, image.ContentStore(), ic)
			if err != nil {
				return err
			}

			if err := json.Unmarshal(p, &ociimage); err != nil {
				return err
			}
			config = ociimage.Config
		default:
			return fmt.Errorf("unknown image config media type %s", ic.MediaType)
		}
		c.Labels = config.Labels
		return nil
	}
}

// WithAdditionalContainerLabels adds the provided labels to the container
// The existing labels are preserved as long as they do not conflict with the added labels.
func WithAdditionalContainerLabels(labels map[string]string) NewContainerOpts {
	return func(_ context.Context, _ *Client, c *containers.Container) error {
		if c.Labels == nil {
			c.Labels = labels
			return nil
		}
		for k, v := range labels {
			c.Labels[k] = v
		}
		return nil
	}
}

// WithImageStopSignal sets a well-known containerd label (StopSignalLabel)
// on the container for storing the stop signal specified in the OCI image
// config
func WithImageStopSignal(image Image, defaultSignal string) NewContainerOpts {
	return func(ctx context.Context, _ *Client, c *containers.Container) error {
		if c.Labels == nil {
			c.Labels = make(map[string]string)
		}
		stopSignal, err := GetOCIStopSignal(ctx, image, defaultSignal)
		if err != nil {
			return err
		}
		c.Labels[StopSignalLabel] = stopSignal
		return nil
	}
}

// WithSnapshotter sets the provided snapshotter for use by the container
//
// This option must appear before other snapshotter options to have an effect.
func WithSnapshotter(name string) NewContainerOpts {
	return func(ctx context.Context, client *Client, c *containers.Container) error {
		c.Snapshotter = name
		return nil
	}
}

// WithSnapshotCleanup deletes the rootfs snapshot allocated for the container
func WithSnapshotCleanup(ctx context.Context, client *Client, c containers.Container) error {
	if c.SnapshotKey != "" {
		if c.Snapshotter == "" {
			return fmt.Errorf("container.Snapshotter must be set to cleanup rootfs snapshot: %w", errdefs.ErrInvalidArgument)
		}
		s, err := client.getSnapshotter(ctx, c.Snapshotter)
		if err != nil {
			return err
		}
		if err := s.Remove(ctx, c.SnapshotKey); err != nil && !errdefs.IsNotFound(err) {
			return err
		}
	}
	return nil
}

// WithNewSnapshotView allocates a new snapshot to be used by the container as the
// root filesystem in read-only mode

// WithContainerExtension appends extension data to the container object.
// Use this to decorate the container object with additional data for the client
// integration.
//
// Make sure to register the type of `extension` in the typeurl package via
// `typeurl.Register` or container creation may fail.
func WithContainerExtension(name string, extension interface{}) NewContainerOpts {
	return func(ctx context.Context, client *Client, c *containers.Container) error {
		if name == "" {
			return fmt.Errorf("extension key must not be zero-length: %w", errdefs.ErrInvalidArgument)
		}

		any, err := typeurl.MarshalAny(extension)
		if err != nil {
			if errors.Is(err, typeurl.ErrNotFound) {
				return fmt.Errorf("extension %q is not registered with the typeurl package, see `typeurl.Register`: %w", name, err)
			}
			return fmt.Errorf("error marshalling extension: %w", err)
		}

		if c.Extensions == nil {
			c.Extensions = make(map[string]typeurl.Any)
		}
		c.Extensions[name] = any
		return nil
	}
}

func WithSpec(s *oci.Spec, opts ...oci.SpecOpts) NewContainerOpts {
	return func(ctx context.Context, client *Client, c *containers.Container) error {
		if err := oci.ApplyOpts(ctx, client, c, s, opts...); err != nil {
			return err
		}

		var err error
		c.Spec, err = protobuf.MarshalAnyToProto(s)
		return err
	}
}

// WithoutRefreshedMetadata will use the current metadata attached to the container object
func WithoutRefreshedMetadata(i *InfoConfig) {
	i.Refresh = false
}

// WithNewSpec generates a new spec for a new container
func WithNewSpec(opts ...oci.SpecOpts) NewContainerOpts {
	return func(ctx context.Context, client *Client, c *containers.Container) error {
		if _, ok := namespaces.Namespace(ctx); !ok {
			ctx = namespaces.WithNamespace(ctx, client.DefaultNamespace())
		}
		s, err := oci.GenerateSpec(ctx, client, c, opts...)
		if err != nil {
			return err
		}
		c.Spec, err = typeurl.MarshalAny(s)
		return err
	}
}

// WithRuntime allows a user to specify the runtime name and additional options that should
// be used to create tasks for the container
func WithRuntime(name string, options interface{}) NewContainerOpts {
	return func(ctx context.Context, client *Client, c *containers.Container) error {
		var (
			any typeurl.Any
			err error
		)
		if options != nil {
			any, err = typeurl.MarshalAny(options)
			if err != nil {
				return err
			}
		}
		c.Runtime = containers.RuntimeInfo{
			Name:    name,
			Options: any,
		}
		return nil
	}
}

// WithImage sets the provided image as the base for the container
func WithImage(i Image) NewContainerOpts {
	return func(ctx context.Context, client *Client, c *containers.Container) error {
		c.Image = i.Name()
		return nil
	}
}

// WithContainerLabels sets the provided labels to the container.
// The existing labels are cleared.
// Use WithAdditionalContainerLabels to preserve the existing labels.
func WithContainerLabels(labels map[string]string) NewContainerOpts {
	return func(_ context.Context, _ *Client, c *containers.Container) error {
		c.Labels = labels
		return nil
	}
}

// WithNewSnapshot allocates a new snapshot to be used by the container as the
// root filesystem in read-write mode
func WithNewSnapshot(id string, i Image, opts ...snapshots.Opt) NewContainerOpts {
	return func(ctx context.Context, client *Client, c *containers.Container) error {
		diffIDs, err := i.RootFS(ctx)
		if err != nil {
			return err
		}

		parent := identity.ChainID(diffIDs).String()
		c.Snapshotter, err = client.resolveSnapshotterName(ctx, c.Snapshotter)
		if err != nil {
			return err
		}
		s, err := client.getSnapshotter(ctx, c.Snapshotter)
		if err != nil {
			return err
		}

		parent, err = resolveSnapshotOptions(ctx, client, c.Snapshotter, s, parent, opts...)
		if err != nil {
			return err
		}
		if _, err := s.Prepare(ctx, id, parent, opts...); err != nil {
			return err
		}
		c.SnapshotKey = id
		c.Image = i.Name()
		return nil
	}
}
