package containerd

import (
	"context"
	"encoding/json"
	"fmt"
	"syscall"

	"demo/pkg/content"
	"demo/pkg/images"
	"github.com/moby/sys/signal"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// StopSignalLabel is a well-known containerd label for storing the stop
// signal specified in the OCI image config
const StopSignalLabel = "io.containerd.image.config.stop-signal"

// GetStopSignal retrieves the container stop signal, specified by the
// well-known containerd label (StopSignalLabel)
func GetStopSignal(ctx context.Context, container Container, defaultSignal syscall.Signal) (syscall.Signal, error) {
	labels, err := container.Labels(ctx)
	if err != nil {
		return -1, err
	}

	if stopSignal, ok := labels[StopSignalLabel]; ok {
		return signal.ParseSignal(stopSignal)
	}

	return defaultSignal, nil
}

// GetOCIStopSignal retrieves the stop signal specified in the OCI image config
func GetOCIStopSignal(ctx context.Context, image Image, defaultSignal string) (string, error) {
	_, err := signal.ParseSignal(defaultSignal)
	if err != nil {
		return "", err
	}
	ic, err := image.Config(ctx)
	if err != nil {
		return "", err
	}
	var (
		ociimage v1.Image
		config   v1.ImageConfig
	)
	switch ic.MediaType {
	case v1.MediaTypeImageConfig, images.MediaTypeDockerSchema2Config:
		p, err := content.ReadBlob(ctx, image.ContentStore(), ic)
		if err != nil {
			return "", err
		}

		if err := json.Unmarshal(p, &ociimage); err != nil {
			return "", err
		}
		config = ociimage.Config
	default:
		return "", fmt.Errorf("unknown image config media type %s", ic.MediaType)
	}

	if config.StopSignal == "" {
		return defaultSignal, nil
	}

	return config.StopSignal, nil
}

// ParseSignal parses a given string into a syscall.Signal
// the rawSignal can be a string with "SIG" prefix,
// or a signal number in string format.
//
// Deprecated: Use github.com/moby/sys/signal instead.
