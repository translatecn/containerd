package encryption

import (
	"fmt"
	"io"

	"demo/pkg/images"
	"github.com/containers/ocicrypt"
	encconfig "github.com/containers/ocicrypt/config"
	encocispec "github.com/containers/ocicrypt/spec"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// LayerFilter allows to select Layers by certain criteria
type LayerFilter func(desc ocispec.Descriptor) bool

// DecryptLayer decrypts the layer using the DecryptConfig and creates a new OCI Descriptor.
// The caller is expected to store the returned plain data and OCI Descriptor
func DecryptLayer(dc *encconfig.DecryptConfig, dataReader io.Reader, desc ocispec.Descriptor, unwrapOnly bool) (ocispec.Descriptor, io.Reader, digest.Digest, error) {
	resultReader, layerDigest, err := ocicrypt.DecryptLayer(dc, dataReader, desc, unwrapOnly)
	if err != nil || unwrapOnly {
		return ocispec.Descriptor{}, nil, "", err
	}

	newDesc := ocispec.Descriptor{
		Size:     0,
		Platform: desc.Platform,
	}

	switch desc.MediaType {
	case encocispec.MediaTypeLayerGzipEnc:
		newDesc.MediaType = images.MediaTypeDockerSchema2LayerGzip
	case encocispec.MediaTypeLayerZstdEnc:
		newDesc.MediaType = ocispec.MediaTypeImageLayerZstd
	case encocispec.MediaTypeLayerEnc:
		newDesc.MediaType = images.MediaTypeDockerSchema2Layer
	default:
		return ocispec.Descriptor{}, nil, "", fmt.Errorf("unsupporter layer MediaType: %s", desc.MediaType)
	}
	return newDesc, resultReader, layerDigest, nil
}
