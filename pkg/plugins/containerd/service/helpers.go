package service

import (
	api "demo/pkg/api/services/containers/v1"
	imagesapi "demo/pkg/api/services/images/v1"
	"demo/pkg/api/types"
	"demo/pkg/containers"
	"demo/pkg/images"
	"demo/pkg/protobuf"
	ptypes "demo/pkg/protobuf/types"
	"demo/pkg/typeurl/v2"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func imagesToProto(images []images.Image) []*imagesapi.Image {
	var imagespb []*imagesapi.Image

	for _, image := range images {
		image := image
		imagespb = append(imagespb, imageToProto(&image))
	}

	return imagespb
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

func containersToProto(containers []containers.Container) []*api.Container {
	var containerspb []*api.Container

	for _, image := range containers {
		image := image
		containerspb = append(containerspb, containerToProto(&image))
	}

	return containerspb
}

func containerToProto(container *containers.Container) *api.Container {
	extensions := make(map[string]*ptypes.Any)
	for k, v := range container.Extensions {
		extensions[k] = protobuf.FromAny(v)
	}
	return &api.Container{
		ID:     container.ID,
		Labels: container.Labels,
		Image:  container.Image,
		Runtime: &api.Container_Runtime{
			Name:    container.Runtime.Name,
			Options: protobuf.FromAny(container.Runtime.Options),
		},
		Spec:        protobuf.FromAny(container.Spec),
		Snapshotter: container.Snapshotter,
		SnapshotKey: container.SnapshotKey,
		CreatedAt:   protobuf.ToTimestamp(container.CreatedAt),
		UpdatedAt:   protobuf.ToTimestamp(container.UpdatedAt),
		Extensions:  extensions,
		Sandbox:     container.SandboxID,
	}
}

func containerFromProto(containerpb *api.Container) containers.Container {
	var runtime containers.RuntimeInfo
	if containerpb.Runtime != nil {
		runtime = containers.RuntimeInfo{
			Name:    containerpb.Runtime.Name,
			Options: containerpb.Runtime.Options,
		}
	}
	extensions := make(map[string]typeurl.Any)
	for k, v := range containerpb.Extensions {
		v := v
		extensions[k] = v
	}
	return containers.Container{
		ID:          containerpb.ID,
		Labels:      containerpb.Labels,
		Image:       containerpb.Image,
		Runtime:     runtime,
		Spec:        containerpb.Spec,
		Snapshotter: containerpb.Snapshotter,
		SnapshotKey: containerpb.SnapshotKey,
		Extensions:  extensions,
		SandboxID:   containerpb.Sandbox,
	}
}
