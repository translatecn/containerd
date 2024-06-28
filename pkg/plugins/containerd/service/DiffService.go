package service

import (
	"context"
	"demo/pkg/plugin"
	"demo/pkg/plugins"
	"demo/pkg/typeurl/v2"
	"fmt"

	diffapi "demo/pkg/api/services/diff/v1"
	"demo/pkg/api/types"
	"demo/pkg/diff"
	"demo/pkg/errdefs"
	"demo/pkg/mount"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"google.golang.org/grpc"
)

type config struct {
	// Order is the order of preference in which to try diff algorithms, the
	// first differ which is supported is used.
	// Note when multiple differs may be supported, this order will be
	// respected for which is chosen. Each differ should return the same
	// correct output, allowing any ordering to be used to prefer
	// more optimimal implementations.
	Order []string `toml:"default"`
}

type differ interface {
	diff.Comparer
	diff.Applier
}

func init() {
	plugin.Register(&plugin.Registration{
		Type: plugin.ServicePlugin,
		ID:   plugins.DiffService,
		Requires: []plugin.Type{
			plugin.DiffPlugin,
		},
		Config: defaultDifferConfig,
		InitFn: func(ic *plugin.InitContext) (interface{}, error) {
			differs, err := ic.GetByType(plugin.DiffPlugin)
			if err != nil {
				return nil, err
			}

			orderedNames := ic.Config.(*config).Order
			ordered := make([]differ, len(orderedNames))
			for i, n := range orderedNames {
				differp, ok := differs[n]
				if !ok {
					return nil, fmt.Errorf("needed differ not loaded: %s", n)
				}
				d, err := differp.Instance()
				if err != nil {
					return nil, fmt.Errorf("could not load required differ due plugin init error: %s: %w", n, err)
				}

				ordered[i], ok = d.(differ)
				if !ok {
					return nil, fmt.Errorf("differ does not implement Comparer and Applier interface: %s", n)
				}
			}

			return &localDiff{
				differs: ordered,
			}, nil
		},
	})
}

type localDiff struct {
	differs []differ
}

var _ diffapi.DiffClient = &localDiff{}

func (l *localDiff) Apply(ctx context.Context, er *diffapi.ApplyRequest, _ ...grpc.CallOption) (*diffapi.ApplyResponse, error) {
	var (
		ocidesc ocispec.Descriptor
		err     error
		desc    = toDescriptor(er.Diff)
		mounts  = toMounts(er.Mounts)
	)

	var opts []diff.ApplyOpt
	if er.Payloads != nil {
		payloads := make(map[string]typeurl.Any)
		for k, v := range er.Payloads {
			payloads[k] = v
		}
		opts = append(opts, diff.WithPayloads(payloads))
	}
	opts = append(opts, diff.WithSyncFs(er.SyncFs))

	for _, differ := range l.differs {
		ocidesc, err = differ.Apply(ctx, desc, mounts, opts...)
		if !errdefs.IsNotImplemented(err) {
			break
		}
	}

	if err != nil {
		return nil, errdefs.ToGRPC(err)
	}

	return &diffapi.ApplyResponse{
		Applied: fromDescriptor(ocidesc),
	}, nil

}

func (l *localDiff) Diff(ctx context.Context, dr *diffapi.DiffRequest, _ ...grpc.CallOption) (*diffapi.DiffResponse, error) {
	var (
		ocidesc ocispec.Descriptor
		err     error
		aMounts = toMounts(dr.Left)
		bMounts = toMounts(dr.Right)
	)

	var opts []diff.Opt
	if dr.MediaType != "" {
		opts = append(opts, diff.WithMediaType(dr.MediaType))
	}
	if dr.Ref != "" {
		opts = append(opts, diff.WithReference(dr.Ref))
	}
	if dr.Labels != nil {
		opts = append(opts, diff.WithLabels(dr.Labels))
	}
	if dr.SourceDateEpoch != nil {
		tm := dr.SourceDateEpoch.AsTime()
		opts = append(opts, diff.WithSourceDateEpoch(&tm))
	}

	for _, d := range l.differs {
		ocidesc, err = d.Compare(ctx, aMounts, bMounts, opts...)
		if !errdefs.IsNotImplemented(err) {
			break
		}
	}
	if err != nil {
		return nil, errdefs.ToGRPC(err)
	}

	return &diffapi.DiffResponse{
		Diff: fromDescriptor(ocidesc),
	}, nil
}

func toMounts(apim []*types.Mount) []mount.Mount {
	mounts := make([]mount.Mount, len(apim))
	for i, m := range apim {
		mounts[i] = mount.Mount{
			Type:    m.Type,
			Source:  m.Source,
			Target:  m.Target,
			Options: m.Options,
		}
	}
	return mounts
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
