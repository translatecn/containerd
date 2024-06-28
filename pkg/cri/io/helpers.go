package io

import (
	"context"
	"demo/pkg/my_mk"
	"io"
	"path/filepath"
	"sync"
	"syscall"

	runtime "demo/pkg/api/cri/v1"
	"demo/pkg/cio"
)

// AttachOptions specifies how to attach to a container.
type AttachOptions struct {
	Stdin     io.Reader
	Stdout    io.WriteCloser
	Stderr    io.WriteCloser
	Tty       bool
	StdinOnce bool
	// CloseStdin is the function to close container stdin.
	CloseStdin func() error
}

// StreamType is the type of the stream, stdout/stderr.
type StreamType string

const (
	// Stdin stream type.
	Stdin StreamType = "stdin"
	// Stdout stream type.
	Stdout = StreamType(runtime.Stdout)
	// Stderr stream type.
	Stderr = StreamType(runtime.Stderr)
)

type wgCloser struct {
	ctx    context.Context
	wg     *sync.WaitGroup
	set    []io.Closer
	cancel context.CancelFunc
}

func (g *wgCloser) Wait() {
	g.wg.Wait()
}

func (g *wgCloser) Close() {
	for _, f := range g.set {
		f.Close()
	}
}

func (g *wgCloser) Cancel() {
	g.cancel()
}

// newFifos creates fifos directory for a container.
func newFifos(root, id string, tty, stdin bool) (*cio.FIFOSet, error) {
	root = filepath.Join(root, "io")
	if err := my_mk.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	fifos, err := cio.NewFIFOSetInDir(root, id, tty)
	if err != nil {
		return nil, err
	}
	if !stdin {
		fifos.Stdin = ""
	}
	return fifos, nil
}

type stdioPipes struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
}

// newStdioPipes creates actual fifos for stdio.
func newStdioPipes(fifos *cio.FIFOSet) (_ *stdioPipes, _ *wgCloser, err error) {
	var (
		f           io.ReadWriteCloser
		set         []io.Closer
		ctx, cancel = context.WithCancel(context.Background())
		p           = &stdioPipes{}
	)
	defer func() {
		if err != nil {
			for _, f := range set {
				f.Close()
			}
			cancel()
		}
	}()

	if fifos.Stdin != "" {
		if f, err = openPipe(ctx, fifos.Stdin, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_NONBLOCK, 0700); err != nil {
			return nil, nil, err
		}
		p.stdin = f
		set = append(set, f)
	}

	if fifos.Stdout != "" {
		if f, err = openPipe(ctx, fifos.Stdout, syscall.O_RDONLY|syscall.O_CREAT|syscall.O_NONBLOCK, 0700); err != nil {
			return nil, nil, err
		}
		p.stdout = f
		set = append(set, f)
	}

	if fifos.Stderr != "" {
		if f, err = openPipe(ctx, fifos.Stderr, syscall.O_RDONLY|syscall.O_CREAT|syscall.O_NONBLOCK, 0700); err != nil {
			return nil, nil, err
		}
		p.stderr = f
		set = append(set, f)
	}

	return p, &wgCloser{
		wg:     &sync.WaitGroup{},
		set:    set,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}
