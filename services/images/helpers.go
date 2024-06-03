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

package images

import (
	"demo/over/images"
	"demo/over/protobuf"
	imagesapi "demo/pkg/api/services/images/v1"
	"demo/pkg/api/types"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func imagesToProto(images []over_images.Image) []*imagesapi.Image {
	var imagespb []*imagesapi.Image

	for _, image := range images {
		image := image
		imagespb = append(imagespb, imageToProto(&image))
	}

	return imagespb
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
