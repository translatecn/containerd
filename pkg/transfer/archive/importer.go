package archive

import (
	"context"
	"demo/pkg/log"
	"demo/pkg/streaming"
	"demo/pkg/typeurl/v2"
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	transferapi "demo/pkg/api/types/transfer"
	"demo/pkg/archive/compression"
	"demo/pkg/content"
	"demo/pkg/images/archive"
	tstreaming "demo/pkg/transfer/streaming"
)

type ImportOpt func(*ImageImportStream)

func WithForceCompression(s *ImageImportStream) {
	s.forceCompress = true
}

// NewImageImportStream returns a image importer via tar stream
func NewImageImportStream(stream io.Reader, mediaType string, opts ...ImportOpt) *ImageImportStream {
	s := &ImageImportStream{
		stream:    stream,
		mediaType: mediaType,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

type ImageImportStream struct {
	stream        io.Reader
	mediaType     string
	forceCompress bool
}

func (iis *ImageImportStream) ImportStream(context.Context) (io.Reader, string, error) {
	return iis.stream, iis.mediaType, nil
}

func (iis *ImageImportStream) Import(ctx context.Context, store content.Store) (ocispec.Descriptor, error) {
	var opts []archive.ImportOpt
	if iis.forceCompress {
		opts = append(opts, archive.WithImportCompression())
	}

	r := iis.stream
	if iis.mediaType == "" {
		d, err := compression.DecompressStream(iis.stream)
		if err != nil {
			return ocispec.Descriptor{}, err
		}
		defer d.Close()
		r = d
	}

	return archive.ImportIndex(ctx, store, r, opts...)
}

func (iis *ImageImportStream) MarshalAny(ctx context.Context, sm streaming.StreamCreator) (typeurl.Any, error) {
	sid := tstreaming.GenerateID("import")
	stream, err := sm.Create(ctx, sid)
	if err != nil {
		return nil, err
	}
	tstreaming.SendStream(ctx, iis.stream, stream)

	s := &transferapi.ImageImportStream{
		Stream:        sid,
		MediaType:     iis.mediaType,
		ForceCompress: iis.forceCompress,
	}

	return typeurl.MarshalAny(s)
}

func (iis *ImageImportStream) UnmarshalAny(ctx context.Context, sm streaming.StreamGetter, any typeurl.Any) error {
	var s transferapi.ImageImportStream
	if err := typeurl.UnmarshalTo(any, &s); err != nil {
		return err
	}

	stream, err := sm.Get(ctx, s.Stream)
	if err != nil {
		log.G(ctx).WithError(err).WithField("stream", s.Stream).Debug("failed to get import stream")
		return err
	}

	iis.stream = tstreaming.ReceiveStream(ctx, stream)
	iis.mediaType = s.MediaType
	iis.forceCompress = s.ForceCompress

	return nil
}
