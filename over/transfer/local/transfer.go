package local

import (
	"context"
	"demo/over/kmutex"
	"demo/over/leases"
	"demo/over/typeurl/v2"
	"demo/over/unpack"
	"fmt"
	"io"
	"time"

	"golang.org/x/sync/semaphore"

	"demo/over/content"
	"demo/over/errdefs"
	"demo/over/images"
	"demo/over/transfer"
)

type localTransferService struct {
	leases  leases.Manager
	content content.Store
	images  images.Store
	// limiter for upload
	limiterU *semaphore.Weighted
	// limiter for download operation
	limiterD *semaphore.Weighted
	config   TransferConfig
}

func NewTransferService(lm leases.Manager, cs content.Store, is images.Store, tc *TransferConfig) transfer.Transferrer {
	ts := &localTransferService{
		leases:  lm,
		content: cs,
		images:  is,
		config:  *tc,
	}
	if tc.MaxConcurrentUploadedLayers > 0 {
		ts.limiterU = semaphore.NewWeighted(int64(tc.MaxConcurrentUploadedLayers))
	}
	if tc.MaxConcurrentDownloads > 0 {
		ts.limiterD = semaphore.NewWeighted(int64(tc.MaxConcurrentDownloads))
	}
	return ts
}

func (ts *localTransferService) Transfer(ctx context.Context, src interface{}, dest interface{}, opts ...transfer.Opt) error {
	topts := &transfer.Config{}
	for _, opt := range opts {
		opt(topts)
	}

	// Figure out matrix of whether source destination combination is supported
	switch s := src.(type) {
	case transfer.ImageFetcher:
		switch d := dest.(type) {
		case transfer.ImageStorer:
			return ts.pull(ctx, s, d, topts)
		}
	case transfer.ImageGetter:
		switch d := dest.(type) {
		case transfer.ImagePusher:
			return ts.push(ctx, s, d, topts)
		case transfer.ImageExporter:
			return ts.exportStream(ctx, s, d, topts)
		case transfer.ImageStorer:
			return ts.tag(ctx, s, d, topts)
		}
	case transfer.ImageImporter:
		switch d := dest.(type) {
		case transfer.ImageExportStreamer:
			return ts.echo(ctx, s, d, topts)
		case transfer.ImageStorer:
			return ts.importStream(ctx, s, d, topts)
		}
	}
	return fmt.Errorf("unable to transfer from %s to %s: %w", name(src), name(dest), errdefs.ErrNotImplemented)
}

func name(t interface{}) string {
	switch s := t.(type) {
	case fmt.Stringer:
		return s.String()
	case typeurl.Any:
		return s.GetTypeUrl()
	default:
		return fmt.Sprintf("%T", t)
	}
}

// echo is mostly used for testing, it implements an import->export which is
// a no-op which only roundtrips the bytes.
func (ts *localTransferService) echo(ctx context.Context, i transfer.ImageImporter, e transfer.ImageExportStreamer, tops *transfer.Config) error {
	iis, ok := i.(transfer.ImageImportStreamer)
	if !ok {
		return fmt.Errorf("echo requires access to raw stream: %w", errdefs.ErrNotImplemented)
	}
	r, _, err := iis.ImportStream(ctx)
	if err != nil {
		return err
	}
	wc, _, err := e.ExportStream(ctx)
	if err != nil {
		return err
	}

	// TODO: Use fixed buffer? Send write progress?
	_, err = io.Copy(wc, r)
	if werr := wc.Close(); werr != nil && err == nil {
		err = werr
	}
	return err
}

// WithLease attaches a lease on the context
func (ts *localTransferService) withLease(ctx context.Context, opts ...leases.Opt) (context.Context, func(context.Context) error, error) {
	nop := func(context.Context) error { return nil }

	_, ok := leases.FromContext(ctx)
	if ok {
		return ctx, nop, nil
	}

	ls := ts.leases

	if len(opts) == 0 {
		// Use default lease configuration if no options provided
		opts = []leases.Opt{
			leases.WithRandomID(),
			leases.WithExpiration(24 * time.Hour),
		}
	}

	l, err := ls.Create(ctx, opts...)
	if err != nil {
		return ctx, nop, err
	}

	ctx = leases.WithLease(ctx, l.ID)
	return ctx, func(ctx context.Context) error {
		return ls.Delete(ctx, l)
	}, nil
}

type TransferConfig struct {
	// MaxConcurrentDownloads is the max concurrent content downloads for pull.
	MaxConcurrentDownloads int
	// MaxConcurrentUploadedLayers is the max concurrent uploads for push
	MaxConcurrentUploadedLayers int

	// DuplicationSuppressor is used to make sure that there is only one
	// in-flight fetch request or unpack handler for a given descriptor's
	// digest or chain ID.
	DuplicationSuppressor kmutex.KeyedLocker

	// BaseHandlers are a set of handlers which get are called on dispatch.
	// These handlers always get called before any operation specific
	// handlers.
	BaseHandlers []images.Handler

	// UnpackPlatforms are used to specify supported combination of platforms and snapshotters
	UnpackPlatforms []unpack.Platform

	// RegistryConfigPath is a path to the root directory containing registry-specific configurations
	RegistryConfigPath string
}
