package service

import (
	"context"
	"demo/over/deprecation"
	"demo/over/epoch"
	"demo/over/errdefs"
	"demo/over/gc"
	"demo/over/images"
	"demo/over/log"
	metadata2 "demo/over/metadata"
	"demo/over/plugin"
	ptypes "demo/over/protobuf/types"
	"demo/plugins"
	"demo/plugins/containerd/over/warning"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	eventstypes "demo/over/api/events"
	imagesapi "demo/over/api/services/images/v1"
	"demo/over/events"
)

func init() {
	plugin.Register(&plugin.Registration{
		Type: plugin.ServicePlugin,
		ID:   plugins.ImagesService,
		Requires: []plugin.Type{
			plugin.MetadataPlugin,
			plugin.GCPlugin,
			plugin.WarningPlugin,
		},
		InitFn: func(ic *plugin.InitContext) (interface{}, error) {
			m, err := ic.Get(plugin.MetadataPlugin)
			if err != nil {
				return nil, err
			}
			g, err := ic.Get(plugin.GCPlugin)
			if err != nil {
				return nil, err
			}
			w, err := ic.Get(plugin.WarningPlugin)
			if err != nil {
				return nil, err
			}

			return &ImagesService{
				store:     metadata2.NewImageStore(m.(*metadata2.DB)),
				publisher: ic.Events,
				gc:        g.(gcScheduler),
				warnings:  w.(warning.Service),
			}, nil
		},
	})
}

type gcScheduler interface {
	ScheduleAndWait(context.Context) (gc.Stats, error)
}

type ImagesService struct {
	store     images.Store
	gc        gcScheduler
	publisher events.Publisher
	warnings  warning.Service
}

var _ imagesapi.ImagesClient = &ImagesService{}

func (l *ImagesService) Get(ctx context.Context, req *imagesapi.GetImageRequest, _ ...grpc.CallOption) (*imagesapi.GetImageResponse, error) {
	image, err := l.store.Get(ctx, req.Name)
	if err != nil {
		return nil, errdefs.ToGRPC(err)
	}

	imagepb := imageToProto(&image)
	return &imagesapi.GetImageResponse{
		Image: imagepb,
	}, nil
}

func (l *ImagesService) List(ctx context.Context, req *imagesapi.ListImagesRequest, _ ...grpc.CallOption) (*imagesapi.ListImagesResponse, error) {
	images, err := l.store.List(ctx, req.Filters...)
	if err != nil {
		return nil, errdefs.ToGRPC(err)
	}

	return &imagesapi.ListImagesResponse{
		Images: imagesToProto(images),
	}, nil
}

func (l *ImagesService) Create(ctx context.Context, req *imagesapi.CreateImageRequest, _ ...grpc.CallOption) (*imagesapi.CreateImageResponse, error) {
	log.G(ctx).WithField("name", req.Image.Name).WithField("target", req.Image.Target.Digest).Debugf("create image")
	if req.Image.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Image.Name required")
	}

	var (
		image = imageFromProto(req.Image)
		resp  imagesapi.CreateImageResponse
	)
	if req.SourceDateEpoch != nil {
		tm := req.SourceDateEpoch.AsTime()
		ctx = epoch.WithSourceDateEpoch(ctx, &tm)
	}
	created, err := l.store.Create(ctx, image)
	if err != nil {
		return nil, errdefs.ToGRPC(err)
	}

	resp.Image = imageToProto(&created)

	if err := l.publisher.Publish(ctx, "/images/create", &eventstypes.ImageCreate{
		Name:   resp.Image.Name,
		Labels: resp.Image.Labels,
	}); err != nil {
		return nil, err
	}

	l.emitSchema1DeprecationWarning(ctx, &image)
	return &resp, nil

}

func (l *ImagesService) Update(ctx context.Context, req *imagesapi.UpdateImageRequest, _ ...grpc.CallOption) (*imagesapi.UpdateImageResponse, error) {
	if req.Image.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Image.Name required")
	}

	var (
		image      = imageFromProto(req.Image)
		resp       imagesapi.UpdateImageResponse
		fieldpaths []string
	)

	if req.UpdateMask != nil && len(req.UpdateMask.Paths) > 0 {
		fieldpaths = append(fieldpaths, req.UpdateMask.Paths...)
	}

	if req.SourceDateEpoch != nil {
		tm := req.SourceDateEpoch.AsTime()
		ctx = epoch.WithSourceDateEpoch(ctx, &tm)
	}

	updated, err := l.store.Update(ctx, image, fieldpaths...)
	if err != nil {
		return nil, errdefs.ToGRPC(err)
	}

	resp.Image = imageToProto(&updated)

	if err := l.publisher.Publish(ctx, "/images/update", &eventstypes.ImageUpdate{
		Name:   resp.Image.Name,
		Labels: resp.Image.Labels,
	}); err != nil {
		return nil, err
	}

	l.emitSchema1DeprecationWarning(ctx, &image)
	return &resp, nil
}

func (l *ImagesService) Delete(ctx context.Context, req *imagesapi.DeleteImageRequest, _ ...grpc.CallOption) (*ptypes.Empty, error) {
	log.G(ctx).WithField("name", req.Name).Debugf("delete image")

	if err := l.store.Delete(ctx, req.Name); err != nil {
		return nil, errdefs.ToGRPC(err)
	}

	if err := l.publisher.Publish(ctx, "/images/delete", &eventstypes.ImageDelete{
		Name: req.Name,
	}); err != nil {
		return nil, err
	}

	if req.Sync {
		if _, err := l.gc.ScheduleAndWait(ctx); err != nil {
			return nil, err
		}
	}

	return &ptypes.Empty{}, nil
}

func (l *ImagesService) emitSchema1DeprecationWarning(ctx context.Context, image *images.Image) {
	if image == nil {
		return
	}
	dgst, ok := image.Labels[images.ConvertedDockerSchema1LabelKey]
	if !ok {
		return
	}
	log.G(ctx).WithField("name", image.Name).WithField("schema1digest", dgst).Warn("conversion from schema 1 images is deprecated")
	l.warnings.Emit(ctx, deprecation.PullSchema1Image)
}
