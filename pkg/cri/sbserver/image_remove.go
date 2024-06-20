package sbserver

import (
	"context"
	"demo/over/ctr_tracing"
	"fmt"

	runtime "demo/over/api/cri/v1"
	"demo/over/errdefs"
	"demo/over/images"
)

// RemoveImage removes the image.
// TODO(random-liu): Update CRI to pass image reference instead of ImageSpec. (See
// kubernetes/kubernetes#46255)
// TODO(random-liu): We should change CRI to distinguish image id and image spec.
// Remove the whole image no matter the it's image id or reference. This is the
// semantic defined in CRI now.
func (c *CriService) RemoveImage(ctx context.Context, r *runtime.RemoveImageRequest) (*runtime.RemoveImageResponse, error) {
	span := tracing.SpanFromContext(ctx)
	image, err := c.localResolve(r.GetImage().GetImage())
	if err != nil {
		if errdefs.IsNotFound(err) {
			span.AddEvent(err.Error())
			// return empty without error when image not found.
			return &runtime.RemoveImageResponse{}, nil
		}
		return nil, fmt.Errorf("can not resolve %q locally: %w", r.GetImage().GetImage(), err)
	}
	span.SetAttributes(tracing.Attribute("image.id", image.ID))
	// Remove all image references.
	for i, ref := range image.References {
		var opts []images.DeleteOpt
		if i == len(image.References)-1 {
			// Delete the last image reference synchronously to trigger garbage collection.
			// This is best effort. It is possible that the image reference is deleted by
			// someone else before this point.
			opts = []images.DeleteOpt{images.SynchronousDelete()}
		}
		err = c.client.ImageService().Delete(ctx, ref, opts...)
		if err == nil || errdefs.IsNotFound(err) {
			// Update image store to reflect the newest state in containerd.
			if err := c.imageStore.Update(ctx, ref); err != nil {
				return nil, fmt.Errorf("failed to update image reference %q for %q: %w", ref, image.ID, err)
			}
			continue
		}
		return nil, fmt.Errorf("failed to delete image reference %q for %q: %w", ref, image.ID, err)
	}
	return &runtime.RemoveImageResponse{}, nil
}
