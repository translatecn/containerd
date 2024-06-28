package sbserver

import (
	"context"

	runtime "demo/pkg/api/cri/v1"
)

// ListImages lists existing images.
// TODO(random-liu): Add image list filters after CRI defines this more clear, and kubelet
// actually needs it.
func (c *CriService) ListImages(ctx context.Context, r *runtime.ListImagesRequest) (*runtime.ListImagesResponse, error) {
	imagesInStore := c.imageStore.List()

	var images []*runtime.Image
	for _, image := range imagesInStore {
		// TODO(random-liu): [P0] Make sure corresponding snapshot exists. What if snapshot
		// doesn't exist?
		images = append(images, toCRIImage(image))
	}

	return &runtime.ListImagesResponse{Images: images}, nil
}
