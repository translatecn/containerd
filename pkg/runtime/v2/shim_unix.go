package v2

import (
	"context"
	fifo2 "demo/pkg/fifo"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"
)

func openShimLog(ctx context.Context, bundle *Bundle, _ func(string, time.Duration) (net.Conn, error)) (io.ReadCloser, error) {
	return fifo2.OpenFifo(ctx, filepath.Join(bundle.Path, "log"), unix.O_RDWR|unix.O_CREAT|unix.O_NONBLOCK, 0700)
}

func checkCopyShimLogError(ctx context.Context, err error) error {
	select {
	case <-ctx.Done():
		if err == fifo2.ErrReadClosed || errors.Is(err, os.ErrClosed) {
			return nil
		}
	default:
	}
	return err
}
