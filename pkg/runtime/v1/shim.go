package v1

import (
	"context"
	"demo/pkg/fifo"
	"io"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func OpenShimStdoutLog(ctx context.Context, logDirPath string) (io.ReadWriteCloser, error) {
	return fifo.OpenFifo(ctx, filepath.Join(logDirPath, "shim.stdout.log"), unix.O_RDWR|unix.O_CREAT, 0700)
}

func OpenShimStderrLog(ctx context.Context, logDirPath string) (io.ReadWriteCloser, error) {
	return fifo.OpenFifo(ctx, filepath.Join(logDirPath, "shim.stderr.log"), unix.O_RDWR|unix.O_CREAT, 0700)
}
