package containerd

import (
	"context"
	"demo/pkg/epoch"
	"demo/pkg/protobuf"
	ptypes "demo/pkg/protobuf/types"

	imagesapi "demo/pkg/api/services/images/v1"
	"demo/pkg/api/types"
	"demo/pkg/errdefs"
	"demo/pkg/images"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type remoteImages struct {
	client imagesapi.ImagesClient
}

// NewImageStoreFromClient returns a new image store client
func NewImageStoreFromClient(client imagesapi.ImagesClient) images.Store {
	return &remoteImages{
		client: client,
	}
}

func (s *remoteImages) List(ctx context.Context, filters ...string) ([]images.Image, error) {
	resp, err := s.client.List(ctx, &imagesapi.ListImagesRequest{
		Filters: filters,
	})
	if err != nil {
		return nil, errdefs.FromGRPC(err)
	}

	return imagesFromProto(resp.Images), nil
}

func (s *remoteImages) Update(ctx context.Context, image images.Image, fieldpaths ...string) (images.Image, error) {
	var updateMask *ptypes.FieldMask
	if len(fieldpaths) > 0 {
		updateMask = &ptypes.FieldMask{
			Paths: fieldpaths,
		}
	}
	req := &imagesapi.UpdateImageRequest{
		Image:      imageToProto(&image),
		UpdateMask: updateMask,
	}
	if tm := epoch.FromContext(ctx); tm != nil {
		req.SourceDateEpoch = timestamppb.New(*tm)
	}
	updated, err := s.client.Update(ctx, req)
	if err != nil {
		return images.Image{}, errdefs.FromGRPC(err)
	}

	return imageFromProto(updated.Image), nil
}

func (s *remoteImages) Delete(ctx context.Context, name string, opts ...images.DeleteOpt) error {
	var do images.DeleteOptions
	for _, opt := range opts {
		if err := opt(ctx, &do); err != nil {
			return err
		}
	}
	_, err := s.client.Delete(ctx, &imagesapi.DeleteImageRequest{
		Name: name,
		Sync: do.Synchronous,
	})

	return errdefs.FromGRPC(err)
}

func imageToProto(image *images.Image) *imagesapi.Image {
	return &imagesapi.Image{
		Name:      image.Name,
		Labels:    image.Labels,
		Target:    descToProto(&image.Target),
		CreatedAt: protobuf.ToTimestamp(image.CreatedAt),
		UpdatedAt: protobuf.ToTimestamp(image.UpdatedAt),
	}
}

func imageFromProto(imagepb *imagesapi.Image) images.Image {
	return images.Image{
		Name:      imagepb.Name,
		Labels:    imagepb.Labels,
		Target:    descFromProto(imagepb.Target),
		CreatedAt: protobuf.FromTimestamp(imagepb.CreatedAt),
		UpdatedAt: protobuf.FromTimestamp(imagepb.UpdatedAt),
	}
}

func imagesFromProto(imagespb []*imagesapi.Image) []images.Image {
	var images []images.Image

	for _, image := range imagespb {
		image := image
		images = append(images, imageFromProto(image))
	}

	return images
}

func descFromProto(desc *types.Descriptor) ocispec.Descriptor {
	return ocispec.Descriptor{
		MediaType:   desc.MediaType,
		Size:        desc.Size,
		Digest:      digest.Digest(desc.Digest),
		Annotations: desc.Annotations,
	}
}

func descToProto(desc *ocispec.Descriptor) *types.Descriptor {
	return &types.Descriptor{
		MediaType:   desc.MediaType,
		Size:        desc.Size,
		Digest:      desc.Digest.String(),
		Annotations: desc.Annotations,
	}
}
func (s *remoteImages) Create(ctx context.Context, image images.Image) (images.Image, error) {
	req := &imagesapi.CreateImageRequest{
		Image: imageToProto(&image),
	}
	if tm := epoch.FromContext(ctx); tm != nil {
		req.SourceDateEpoch = timestamppb.New(*tm)
	}
	created, err := s.client.Create(ctx, req)
	if err != nil {
		return images.Image{}, errdefs.FromGRPC(err)
	}

	return imageFromProto(created.Image), nil
}

func (s *remoteImages) Get(ctx context.Context, name string) (images.Image, error) {
	resp, err := s.client.Get(ctx, &imagesapi.GetImageRequest{
		Name: name,
	})
	if err != nil {
		return images.Image{}, errdefs.FromGRPC(err)
	}

	return imageFromProto(resp.Image), nil
}