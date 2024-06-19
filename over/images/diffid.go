package images

import (
	"context"
	"demo/over/labels"
	"io"

	"demo/over/archive/compression"
	"demo/over/content"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
)

// GetDiffID gets the diff ID of the layer blob descriptor.
func GetDiffID(ctx context.Context, cs content.Store, desc ocispec.Descriptor) (digest.Digest, error) {
	switch desc.MediaType {
	case
		// If the layer is already uncompressed, we can just return its digest
		MediaTypeDockerSchema2Layer,
		ocispec.MediaTypeImageLayer,
		MediaTypeDockerSchema2LayerForeign,
		ocispec.MediaTypeImageLayerNonDistributable: //nolint:staticcheck // deprecated
		return desc.Digest, nil
	}
	info, err := cs.Info(ctx, desc.Digest)
	if err != nil {
		return "", err
	}
	v, ok := info.Labels[labels.LabelUncompressed]
	if ok {
		// Fast path: if the image is already unpacked, we can use the label value
		return digest.Parse(v)
	}
	// if the image is not unpacked, we may not have the label
	ra, err := cs.ReaderAt(ctx, desc)
	if err != nil {
		return "", err
	}
	defer ra.Close()
	r := content.NewReader(ra)
	uR, err := compression.DecompressStream(r)
	if err != nil {
		return "", err
	}
	defer uR.Close()
	digester := digest.Canonical.Digester()
	hashW := digester.Hash()
	if _, err := io.Copy(hashW, uR); err != nil {
		return "", err
	}
	if err := ra.Close(); err != nil {
		return "", err
	}
	digest := digester.Digest()
	// memorize the computed value
	if info.Labels == nil {
		info.Labels = make(map[string]string)
	}
	info.Labels[labels.LabelUncompressed] = digest.String()
	if _, err := cs.Update(ctx, info, "labels"); err != nil {
		logrus.WithError(err).Warnf("failed to set %s label for %s", labels.LabelUncompressed, desc.Digest)
	}
	return digest, nil
}
