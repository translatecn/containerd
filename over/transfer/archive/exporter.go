package archive

import (
	"context"
	"demo/over/log"
	"demo/over/streaming"
	"demo/over/typeurl/v2"
	"io"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	"demo/over/api/types"
	transfertypes "demo/over/api/types/transfer"
	"demo/over/content"
	"demo/over/images"
	"demo/over/images/archive"
	"demo/over/platforms"
	"demo/over/transfer/plugins"
	tstreaming "demo/over/transfer/streaming"
)

func init() {
	// TODO: Move this to separate package?
	plugins.Register(&transfertypes.ImageExportStream{}, &ImageExportStream{})
	plugins.Register(&transfertypes.ImageImportStream{}, &ImageImportStream{})
}

type ExportOpt func(*ImageExportStream)

func WithPlatform(p v1.Platform) ExportOpt {
	return func(s *ImageExportStream) {
		s.platforms = append(s.platforms, p)
	}
}

func WithAllPlatforms(s *ImageExportStream) {
	s.allPlatforms = true
}

func WithSkipCompatibilityManifest(s *ImageExportStream) {
	s.skipCompatibilityManifest = true
}

func WithSkipNonDistributableBlobs(s *ImageExportStream) {
	s.skipNonDistributable = true
}

// NewImageExportStream returns an image exporter via tar stream
func NewImageExportStream(stream io.WriteCloser, mediaType string, opts ...ExportOpt) *ImageExportStream {
	s := &ImageExportStream{
		stream:    stream,
		mediaType: mediaType,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

type ImageExportStream struct {
	stream    io.WriteCloser
	mediaType string

	platforms                 []v1.Platform
	allPlatforms              bool
	skipCompatibilityManifest bool
	skipNonDistributable      bool
}

func (iis *ImageExportStream) ExportStream(context.Context) (io.WriteCloser, string, error) {
	return iis.stream, iis.mediaType, nil
}

func (iis *ImageExportStream) Export(ctx context.Context, cs content.Store, imgs []images.Image) error {
	opts := []archive.ExportOpt{
		archive.WithImages(imgs),
	}

	if len(iis.platforms) > 0 {
		opts = append(opts, archive.WithPlatform(platforms.Ordered(iis.platforms...)))
	} else {
		opts = append(opts, archive.WithPlatform(platforms.DefaultStrict()))
	}
	if iis.allPlatforms {
		opts = append(opts, archive.WithAllPlatforms())
	}
	if iis.skipCompatibilityManifest {
		opts = append(opts, archive.WithSkipDockerManifest())
	}
	if iis.skipNonDistributable {
		opts = append(opts, archive.WithSkipNonDistributableBlobs())
	}
	return archive.Export(ctx, cs, iis.stream, opts...)
}

func (iis *ImageExportStream) MarshalAny(ctx context.Context, sm streaming.StreamCreator) (typeurl.Any, error) {
	sid := tstreaming.GenerateID("export")
	stream, err := sm.Create(ctx, sid)
	if err != nil {
		return nil, err
	}

	// Receive stream and copy to writer
	go func() {
		if _, err := io.Copy(iis.stream, tstreaming.ReceiveStream(ctx, stream)); err != nil {
			log.G(ctx).WithError(err).WithField("streamid", sid).Errorf("error copying stream")
		}
		iis.stream.Close()
	}()

	var specified []*types.Platform
	for _, p := range iis.platforms {
		specified = append(specified, &types.Platform{
			OS:           p.OS,
			Architecture: p.Architecture,
			Variant:      p.Variant,
		})
	}
	s := &transfertypes.ImageExportStream{
		Stream:                    sid,
		MediaType:                 iis.mediaType,
		Platforms:                 specified,
		AllPlatforms:              iis.allPlatforms,
		SkipCompatibilityManifest: iis.skipCompatibilityManifest,
		SkipNonDistributable:      iis.skipNonDistributable,
	}

	return typeurl.MarshalAny(s)
}

func (iis *ImageExportStream) UnmarshalAny(ctx context.Context, sm streaming.StreamGetter, any typeurl.Any) error {
	var s transfertypes.ImageExportStream
	if err := typeurl.UnmarshalTo(any, &s); err != nil {
		return err
	}

	stream, err := sm.Get(ctx, s.Stream)
	if err != nil {
		log.G(ctx).WithError(err).WithField("stream", s.Stream).Debug("failed to get export stream")
		return err
	}

	var specified []v1.Platform
	for _, p := range s.Platforms {
		specified = append(specified, v1.Platform{
			OS:           p.OS,
			Architecture: p.Architecture,
			Variant:      p.Variant,
		})
	}

	iis.stream = tstreaming.WriteByteStream(ctx, stream)
	iis.mediaType = s.MediaType
	iis.platforms = specified
	iis.allPlatforms = s.AllPlatforms
	iis.skipCompatibilityManifest = s.SkipCompatibilityManifest
	iis.skipNonDistributable = s.SkipNonDistributable

	return nil
}
