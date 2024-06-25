package io

import (
	"context"
	"demo/over/fifo"
	"io"
	"os"
)

func openPipe(ctx context.Context, fn string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
	return fifo.OpenFifo(ctx, fn, flag, perm)
}
