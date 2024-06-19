package proxy

import (
	"context"
	"demo/over/epoch"
	"demo/over/protobuf"
	ptypes "demo/over/protobuf/types"

	diffapi "demo/over/api/services/diff/v1"
	"demo/over/api/types"
	"demo/over/diff"
	"demo/over/errdefs"
	"demo/over/mount"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// NewDiffApplier returns a new comparer and applier which communicates
// over a GRPC connection.
func NewDiffApplier(client diffapi.DiffClient) interface{} {
	return &diffRemote{
		client: client,
	}
}

type diffRemote struct {
	client diffapi.DiffClient
}

func (r *diffRemote) Apply(ctx context.Context, desc ocispec.Descriptor, mounts []mount.Mount, opts ...diff.ApplyOpt) (ocispec.Descriptor, error) {
	var config diff.ApplyConfig
	for _, opt := range opts {
		if err := opt(ctx, desc, &config); err != nil {
			return ocispec.Descriptor{}, err
		}
	}

	payloads := make(map[string]*ptypes.Any)
	for k, v := range config.ProcessorPayloads {
		payloads[k] = protobuf.FromAny(v)
	}

	req := &diffapi.ApplyRequest{
		Diff:     fromDescriptor(desc),
		Mounts:   fromMounts(mounts),
		Payloads: payloads,
		SyncFs:   config.SyncFs,
	}
	resp, err := r.client.Apply(ctx, req)
	if err != nil {
		return ocispec.Descriptor{}, errdefs.FromGRPC(err)
	}
	return toDescriptor(resp.Applied), nil
}

func (r *diffRemote) Compare(ctx context.Context, a, b []mount.Mount, opts ...diff.Opt) (ocispec.Descriptor, error) {
	var config diff.Config
	for _, opt := range opts {
		if err := opt(&config); err != nil {
			return ocispec.Descriptor{}, err
		}
	}
	if tm := epoch.FromContext(ctx); tm != nil && config.SourceDateEpoch == nil {
		config.SourceDateEpoch = tm
	}
	var sourceDateEpoch *timestamppb.Timestamp
	if config.SourceDateEpoch != nil {
		sourceDateEpoch = timestamppb.New(*config.SourceDateEpoch)
	}
	req := &diffapi.DiffRequest{
		Left:            fromMounts(a),
		Right:           fromMounts(b),
		MediaType:       config.MediaType,
		Ref:             config.Reference,
		Labels:          config.Labels,
		SourceDateEpoch: sourceDateEpoch,
	}
	resp, err := r.client.Diff(ctx, req)
	if err != nil {
		return ocispec.Descriptor{}, errdefs.FromGRPC(err)
	}
	return toDescriptor(resp.Diff), nil
}

func toDescriptor(d *types.Descriptor) ocispec.Descriptor {
	return ocispec.Descriptor{
		MediaType:   d.MediaType,
		Digest:      digest.Digest(d.Digest),
		Size:        d.Size,
		Annotations: d.Annotations,
	}
}

func fromDescriptor(d ocispec.Descriptor) *types.Descriptor {
	return &types.Descriptor{
		MediaType:   d.MediaType,
		Digest:      d.Digest.String(),
		Size:        d.Size,
		Annotations: d.Annotations,
	}
}

func fromMounts(mounts []mount.Mount) []*types.Mount {
	apiMounts := make([]*types.Mount, len(mounts))
	for i, m := range mounts {
		apiMounts[i] = &types.Mount{
			Type:    m.Type,
			Source:  m.Source,
			Target:  m.Target,
			Options: m.Options,
		}
	}
	return apiMounts
}
