package encryption

import (
	"context"
	"demo/over/typeurl"
	"fmt"

	"demo/containerd"
	"demo/others/imgcrypt"
	"demo/over/diff"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// WithDecryptedUnpack allows to pass parameters the 'layertool' needs to the applier
func WithDecryptedUnpack(data *imgcrypt.Payload) diff.ApplyOpt {
	return func(_ context.Context, desc ocispec.Descriptor, c *diff.ApplyConfig) error {
		data.Descriptor = desc
		anything, err := typeurl.MarshalAny(data)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}

		for _, id := range imgcrypt.PayloadToolIDs {
			setProcessorPayload(c, id, anything)
		}
		return nil
	}
}

// WithUnpackConfigApplyOpts allows to pass an ApplyOpt
func WithUnpackConfigApplyOpts(opt diff.ApplyOpt) containerd.UnpackOpt {
	return func(_ context.Context, uc *containerd.UnpackConfig) error {
		uc.ApplyOpts = append(uc.ApplyOpts, opt)
		return nil
	}
}

// WithUnpackOpts is used to add unpack options to the unpacker.

// WithAuthorizationCheck checks the authorization of keys used for encrypted containers
// be checked upon creation of a container
