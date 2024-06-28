package rootfs

import (
	"context"
	"crypto/rand"
	"demo/pkg/log"
	"demo/pkg/snapshots"
	"encoding/base64"
	"fmt"
	"time"

	"demo/pkg/diff"
	"demo/pkg/errdefs"
	"demo/pkg/mount"
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/identity"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Layer represents the descriptors for a layer diff. These descriptions
// include the descriptor for the uncompressed tar diff as well as a blob
// used to transport that tar. The blob descriptor may or may not describe
// a compressed object.
type Layer struct {
	Diff ocispec.Descriptor
	Blob ocispec.Descriptor
}

// ApplyLayerWithOpts applies a single layer on top of the given provided layer chain,
// using the provided snapshotter, applier, and apply opts. If the layer was unpacked true
// is returned, if the layer already exists false is returned.
func ApplyLayerWithOpts(ctx context.Context, layer Layer, chain []digest.Digest, sn snapshots.Snapshotter, a diff.Applier, opts []snapshots.Opt, applyOpts []diff.ApplyOpt) (bool, error) {
	var (
		chainID = identity.ChainID(append(chain, layer.Diff.Digest)).String()
		applied bool
	)
	if _, err := sn.Stat(ctx, chainID); err != nil {
		if !errdefs.IsNotFound(err) {
			return false, fmt.Errorf("failed to stat snapshot %s: %w", chainID, err)
		}

		if err := applyLayers(ctx, []Layer{layer}, append(chain, layer.Diff.Digest), sn, a, opts, applyOpts); err != nil {
			if !errdefs.IsAlreadyExists(err) {
				return false, err
			}
		} else {
			applied = true
		}
	}
	return applied, nil

}

func applyLayers(ctx context.Context, layers []Layer, chain []digest.Digest, sn snapshots.Snapshotter, a diff.Applier, opts []snapshots.Opt, applyOpts []diff.ApplyOpt) error {
	var (
		parent  = identity.ChainID(chain[:len(chain)-1])
		chainID = identity.ChainID(chain)
		layer   = layers[len(layers)-1]
		diff    ocispec.Descriptor
		key     string
		mounts  []mount.Mount
		err     error
	)

	for {
		key = fmt.Sprintf(snapshots.UnpackKeyFormat, uniquePart(), chainID)

		// Prepare snapshot with from parent, label as root
		mounts, err = sn.Prepare(ctx, key, parent.String(), opts...)
		if err != nil {
			if errdefs.IsNotFound(err) && len(layers) > 1 {
				if err := applyLayers(ctx, layers[:len(layers)-1], chain[:len(chain)-1], sn, a, opts, applyOpts); err != nil {
					if !errdefs.IsAlreadyExists(err) {
						return err
					}
				}
				// Do no try applying layers again
				layers = nil
				continue
			} else if errdefs.IsAlreadyExists(err) {
				// Try a different key
				continue
			}

			// Already exists should have the caller retry
			return fmt.Errorf("failed to prepare extraction snapshot %q: %w", key, err)

		}
		break
	}
	defer func() {
		if err != nil {
			if !errdefs.IsAlreadyExists(err) {
				log.G(ctx).WithError(err).WithField("key", key).Infof("apply failure, attempting cleanup")
			}

			if rerr := sn.Remove(ctx, key); rerr != nil {
				log.G(ctx).WithError(rerr).WithField("key", key).Warnf("extraction snapshot removal failed")
			}
		}
	}()

	diff, err = a.Apply(ctx, layer.Blob, mounts, applyOpts...)
	if err != nil {
		err = fmt.Errorf("failed to extract layer %s: %w", layer.Diff.Digest, err)
		return err
	}
	if diff.Digest != layer.Diff.Digest {
		err = fmt.Errorf("wrong diff id calculated on extraction %q", diff.Digest)
		return err
	}

	if err = sn.Commit(ctx, chainID.String(), key, opts...); err != nil {
		err = fmt.Errorf("failed to commit snapshot %s: %w", key, err)
		return err
	}

	return nil
}

func uniquePart() string {
	t := time.Now()
	var b [3]byte
	// Ignore read failures, just decreases uniqueness
	rand.Read(b[:])
	return fmt.Sprintf("%d-%s", t.Nanosecond(), base64.URLEncoding.EncodeToString(b[:]))
}
