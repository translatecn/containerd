package imgcrypt

import (
	"demo/pkg/typeurl"

	encconfig "github.com/containers/ocicrypt/config"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	PayloadURI = "io.containerd.ocicrypt.v1.Payload"
)

var PayloadToolIDs = []string{
	"io.containerd.ocicrypt.decoder.v1.tar",
	"io.containerd.ocicrypt.decoder.v1.tar.gzip",
}

func init() {
	typeurl.Register(&Payload{}, PayloadURI)
}

// Payload holds data that the external layer decryption tool
// needs for decrypting a layer
type Payload struct {
	DecryptConfig encconfig.DecryptConfig
	Descriptor    ocispec.Descriptor
}
