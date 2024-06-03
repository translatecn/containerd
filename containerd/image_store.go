/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package containerd

import (
	"context"
	"demo/over/protobuf"
	ptypes "demo/over/protobuf/types"

	"demo/over/errdefs"
	"demo/over/images"
	imagesapi "demo/pkg/api/services/images/v1"
	"demo/pkg/api/types"
	"demo/pkg/epoch"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type remoteImages struct {
	client imagesapi.ImagesClient
}

// NewImageStoreFromClient returns a new image store client
func NewImageStoreFromClient(client imagesapi.ImagesClient) over_images.Store {
	return &remoteImages{
		client: client,
	}
}

func (s *remoteImages) Get(ctx context.Context, name string) (over_images.Image, error) {
	resp, err := s.client.Get(ctx, &imagesapi.GetImageRequest{
		Name: name,
	})
	if err != nil {
		return over_images.Image{}, over_errdefs.FromGRPC(err)
	}

	return imageFromProto(resp.Image), nil
}

func (s *remoteImages) List(ctx context.Context, filters ...string) ([]over_images.Image, error) {
	resp, err := s.client.List(ctx, &imagesapi.ListImagesRequest{
		Filters: filters,
	})
	if err != nil {
		return nil, over_errdefs.FromGRPC(err)
	}

	return imagesFromProto(resp.Images), nil
}

func (s *remoteImages) Create(ctx context.Context, image over_images.Image) (over_images.Image, error) {
	req := &imagesapi.CreateImageRequest{
		Image: imageToProto(&image),
	}
	if tm := epoch.FromContext(ctx); tm != nil {
		req.SourceDateEpoch = timestamppb.New(*tm)
	}
	created, err := s.client.Create(ctx, req)
	if err != nil {
		return over_images.Image{}, over_errdefs.FromGRPC(err)
	}

	return imageFromProto(created.Image), nil
}

func (s *remoteImages) Update(ctx context.Context, image over_images.Image, fieldpaths ...string) (over_images.Image, error) {
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
		return over_images.Image{}, over_errdefs.FromGRPC(err)
	}

	return imageFromProto(updated.Image), nil
}

func (s *remoteImages) Delete(ctx context.Context, name string, opts ...over_images.DeleteOpt) error {
	var do over_images.DeleteOptions
	for _, opt := range opts {
		if err := opt(ctx, &do); err != nil {
			return err
		}
	}
	_, err := s.client.Delete(ctx, &imagesapi.DeleteImageRequest{
		Name: name,
		Sync: do.Synchronous,
	})

	return over_errdefs.FromGRPC(err)
}

func imageToProto(image *over_images.Image) *imagesapi.Image {
	return &imagesapi.Image{
		Name:      image.Name,
		Labels:    image.Labels,
		Target:    descToProto(&image.Target),
		CreatedAt: over_protobuf.ToTimestamp(image.CreatedAt),
		UpdatedAt: over_protobuf.ToTimestamp(image.UpdatedAt),
	}
}

func imageFromProto(imagepb *imagesapi.Image) over_images.Image {
	return over_images.Image{
		Name:      imagepb.Name,
		Labels:    imagepb.Labels,
		Target:    descFromProto(imagepb.Target),
		CreatedAt: over_protobuf.FromTimestamp(imagepb.CreatedAt),
		UpdatedAt: over_protobuf.FromTimestamp(imagepb.UpdatedAt),
	}
}

func imagesFromProto(imagespb []*imagesapi.Image) []over_images.Image {
	var images []over_images.Image

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
