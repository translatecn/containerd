package containerd

import (
	"demo/pkg/snapshots"
	"time"

	"demo/pkg/images"
	"demo/pkg/platforms"
	"demo/pkg/remotes"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"google.golang.org/grpc"
)

type clientOpts struct {
	defaultns       string
	defaultRuntime  string
	defaultPlatform platforms.MatchComparer
	services        *services
	dialOptions     []grpc.DialOption
	callOptions     []grpc.CallOption
	timeout         time.Duration
}

// ClientOpt allows callers to set options on the containerd client
type ClientOpt func(c *clientOpts) error

// WithDefaultNamespace sets the default namespace on the client
//
// Any operation that does not have a namespace set on the context will
// be provided the default namespace
func WithDefaultNamespace(ns string) ClientOpt {
	return func(c *clientOpts) error {
		c.defaultns = ns
		return nil
	}
}

// WithDefaultRuntime sets the default runtime on the client
func WithDefaultRuntime(rt string) ClientOpt {
	return func(c *clientOpts) error {
		c.defaultRuntime = rt
		return nil
	}
}

// WithDefaultPlatform sets the default platform matcher on the client
func WithDefaultPlatform(platform platforms.MatchComparer) ClientOpt {
	return func(c *clientOpts) error {
		c.defaultPlatform = platform
		return nil
	}
}

// WithTimeout sets the connection timeout for the client
func WithTimeout(d time.Duration) ClientOpt {
	return func(c *clientOpts) error {
		c.timeout = d
		return nil
	}
}

// RemoteOpt allows the caller to set distribution options for a remote
type RemoteOpt func(*Client, *RemoteContext) error

// WithPlatform allows the caller to specify a platform to retrieve
// content for
func WithPlatform(platform string) RemoteOpt {
	if platform == "" {
		platform = platforms.DefaultString()
	}
	return func(_ *Client, c *RemoteContext) error {
		for _, p := range c.Platforms {
			if p == platform {
				return nil
			}
		}

		c.Platforms = append(c.Platforms, platform)
		return nil
	}
}

// WithPlatformMatcher specifies the matcher to use for
// determining which platforms to pull content for.
// This value supersedes anything set with `WithPlatform`.
func WithPlatformMatcher(m platforms.MatchComparer) RemoteOpt {
	return func(_ *Client, c *RemoteContext) error {
		c.PlatformMatcher = m
		return nil
	}
}

// WithPullUnpack is used to unpack an image after pull. This
// uses the snapshotter, content store, and diff service
// configured for the client.
func WithPullUnpack(_ *Client, c *RemoteContext) error {
	c.Unpack = true
	return nil
}

// WithUnpackOpts is used to add unpack options to the unpacker.
func WithUnpackOpts(opts []UnpackOpt) RemoteOpt {
	return func(_ *Client, c *RemoteContext) error {
		c.UnpackOpts = append(c.UnpackOpts, opts...)
		return nil
	}
}

// WithPullSnapshotter specifies snapshotter name used for unpacking.
func WithPullSnapshotter(snapshotterName string, opts ...snapshots.Opt) RemoteOpt {
	return func(_ *Client, c *RemoteContext) error {
		c.Snapshotter = snapshotterName
		c.SnapshotterOpts = opts
		return nil
	}
}

// WithPullLabel sets a label to be associated with a pulled reference

// WithPullLabels associates a set of labels to a pulled reference
func WithPullLabels(labels map[string]string) RemoteOpt {
	return func(_ *Client, rc *RemoteContext) error {
		if rc.Labels == nil {
			rc.Labels = make(map[string]string)
		}

		for k, v := range labels {
			rc.Labels[k] = v
		}
		return nil
	}
}

// WithChildLabelMap sets the map function used to define the labels set
// on referenced child content in the content store. This can be used
// to overwrite the default GC labels or filter which labels get set
// for content.
// The default is `images.ChildGCLabels`.
func WithChildLabelMap(fn func(ocispec.Descriptor) []string) RemoteOpt {
	return func(_ *Client, c *RemoteContext) error {
		c.ChildLabelMap = fn
		return nil
	}
}

// WithSchema1Conversion is used to convert Docker registry schema 1
// manifests to oci manifests on pull. Without this option schema 1
// manifests will return a not supported error.
//
// Deprecated: use Schema 2 or OCI images.
func WithSchema1Conversion(client *Client, c *RemoteContext) error {
	c.ConvertSchema1 = true
	return nil
}

// WithResolver specifies the resolver to use.
func WithResolver(resolver remotes.Resolver) RemoteOpt {
	return func(client *Client, c *RemoteContext) error {
		c.Resolver = resolver
		return nil
	}
}

// WithImageHandler adds a base handler to be called on dispatch.
func WithImageHandler(h images.Handler) RemoteOpt {
	return func(client *Client, c *RemoteContext) error {
		c.BaseHandlers = append(c.BaseHandlers, h)
		return nil
	}
}

// WithImageHandlerWrapper wraps the handlers to be called on dispatch.
func WithImageHandlerWrapper(w func(images.Handler) images.Handler) RemoteOpt {
	return func(client *Client, c *RemoteContext) error {
		c.HandlerWrapper = w
		return nil
	}
}

// WithMaxConcurrentDownloads sets max concurrent download limit.
func WithMaxConcurrentDownloads(max int) RemoteOpt {
	return func(client *Client, c *RemoteContext) error {
		c.MaxConcurrentDownloads = max
		return nil
	}
}

// WithMaxConcurrentUploadedLayers sets max concurrent uploaded layer limit.
func WithMaxConcurrentUploadedLayers(max int) RemoteOpt {
	return func(client *Client, c *RemoteContext) error {
		c.MaxConcurrentUploadedLayers = max
		return nil
	}
}

// WithAllMetadata downloads all manifests and known-configuration files
func WithAllMetadata() RemoteOpt {
	return func(_ *Client, c *RemoteContext) error {
		c.AllMetadata = true
		return nil
	}
}
