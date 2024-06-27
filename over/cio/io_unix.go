package cio

import (
	"context"
	"demo/over/fifo"
	"demo/over/my_mk"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"syscall"
)

func NewFIFOSetInDir(root, id string, terminal bool) (*FIFOSet, error) {
	if root != "" {
		if err := my_mk.MkdirAll(root, 0700); err != nil {
			return nil, err
		}
	}
	dir, err := my_mk.MkdirTemp(root, "")
	if err != nil {
		return nil, err
	}
	closer := func() error {
		return os.RemoveAll(dir)
	}
	return NewFIFOSet(Config{
		Stdin:    filepath.Join(dir, id+"-stdin"), //
		Stdout:   filepath.Join(dir, id+"-stdout"),
		Stderr:   filepath.Join(dir, id+"-stderr"),
		Terminal: terminal,
	}, closer), nil
}

func openFifos(ctx context.Context, fifos *FIFOSet) (f pipes, retErr error) {
	defer func() {
		if retErr != nil {
			fifos.Close()
		}
	}()

	if fifos.Stdin != "" {
		if f.Stdin, retErr = fifo.OpenFifo(ctx, fifos.Stdin, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_NONBLOCK, 0700); retErr != nil {
			return f, fmt.Errorf("failed to open stdin fifo: %w", retErr)
		}
		defer func() {
			if retErr != nil && f.Stdin != nil {
				f.Stdin.Close()
			}
		}()
	}
	if fifos.Stdout != "" {
		if f.Stdout, retErr = fifo.OpenFifo(ctx, fifos.Stdout, syscall.O_RDONLY|syscall.O_CREAT|syscall.O_NONBLOCK, 0700); retErr != nil {
			return f, fmt.Errorf("failed to open stdout fifo: %w", retErr)
		}
		defer func() {
			if retErr != nil && f.Stdout != nil {
				f.Stdout.Close()
			}
		}()
	}
	if !fifos.Terminal && fifos.Stderr != "" {
		if f.Stderr, retErr = fifo.OpenFifo(ctx, fifos.Stderr, syscall.O_RDONLY|syscall.O_CREAT|syscall.O_NONBLOCK, 0700); retErr != nil {
			return f, fmt.Errorf("failed to open stderr fifo: %w", retErr)
		}
	}
	return f, nil
}

func copyIO(fifos *FIFOSet, terminalIo *Streams) (*cio, error) {
	var ctx, cancel = context.WithCancel(context.Background())
	filePipes, err := openFifos(ctx, fifos)
	if err != nil {
		cancel()
		return nil, err
	}

	if fifos.Stdin != "" {
		go func() {
			p := bufPool.Get().(*[]byte)
			defer bufPool.Put(p)

			io.CopyBuffer(filePipes.Stdin, terminalIo.Stdin, *p)
			filePipes.Stdin.Close()
		}()
	}

	var wg = &sync.WaitGroup{}
	if fifos.Stdout != "" {
		wg.Add(1)
		go func() {
			p := bufPool.Get().(*[]byte)
			defer bufPool.Put(p)

			io.CopyBuffer(terminalIo.Stdout, filePipes.Stdout, *p)
			filePipes.Stdout.Close()
			wg.Done()
		}()
	}

	if !fifos.Terminal && fifos.Stderr != "" {
		wg.Add(1)
		go func() {
			p := bufPool.Get().(*[]byte)
			defer bufPool.Put(p)

			io.CopyBuffer(terminalIo.Stderr, filePipes.Stderr, *p)
			filePipes.Stderr.Close()
			wg.Done()
		}()
	}
	return &cio{
		config:  fifos.Config,
		wg:      wg,
		closers: append(filePipes.closers(), fifos),
		cancel: func() {
			cancel()
			for _, c := range filePipes.closers() {
				if c != nil {
					c.Close()
				}
			}
		},
	}, nil
}
