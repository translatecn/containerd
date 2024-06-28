package tasks

import (
	gocontext "context"
	"demo/pkg/log"
	"errors"
	"net/url"
	"os"
	"os/signal"

	"demo/containerd"
	"demo/pkg/cio"
	"demo/pkg/console"
	"github.com/urfave/cli"
	"golang.org/x/sys/unix"
)

var platformStartFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "no-pivot",
		Usage: "Disable use of pivot-root (linux only)",
	},
}

// HandleConsoleResize resizes the console 处理控制台调整大小
func HandleConsoleResize(ctx gocontext.Context, task resizer, con console.Console) error {
	// do an initial resize of the console
	size, err := con.Size()
	if err != nil {
		return err
	}
	if err := task.Resize(ctx, uint32(size.Width), uint32(size.Height)); err != nil {
		log.G(ctx).WithError(err).Error("resize pty")
	}
	s := make(chan os.Signal, 16)
	signal.Notify(s, unix.SIGWINCH) // 窗口大小改变时发出的信号
	go func() {
		for range s {
			size, err := con.Size()
			if err != nil {
				log.G(ctx).WithError(err).Error("get pty size")
				continue
			}
			if err := task.Resize(ctx, uint32(size.Width), uint32(size.Height)); err != nil {
				log.G(ctx).WithError(err).Error("resize pty")
			}
		}
	}()
	return nil
}

func NewTask(ctx gocontext.Context, client *containerd.Client, container containerd.Container, checkpoint string, con console.Console, nullIO bool, logURI string, ioOpts []cio.Opt, opts ...containerd.NewTaskOpts) (containerd.Task, error) {
	stdinC := &stdinCloser{
		stdin: os.Stdin,
	}
	if checkpoint != "" {
		im, err := client.GetImage(ctx, checkpoint)
		if err != nil {
			return nil, err
		}
		opts = append(opts, containerd.WithTaskCheckpoint(im))
	}

	spec, err := container.Spec(ctx)
	if err != nil {
		return nil, err
	}
	if spec.Linux != nil {
		if len(spec.Linux.UIDMappings) != 0 {
			opts = append(opts, containerd.WithUIDOwner(spec.Linux.UIDMappings[0].HostID))
		}
		if len(spec.Linux.GIDMappings) != 0 {
			opts = append(opts, containerd.WithGIDOwner(spec.Linux.GIDMappings[0].HostID))
		}
	}

	var ioCreator cio.Creator
	if con != nil {
		if nullIO {
			return nil, errors.New("tty and null-io cannot be used together")
		}
		ioCreator = cio.NewCreator(append([]cio.Opt{cio.WithStreams(con, con, nil), cio.WithTerminal}, ioOpts...)...) // 终端和 文件之间的读写
	} else if nullIO {
		ioCreator = cio.NullIO
	} else if logURI != "" {
		u, err := url.Parse(logURI)
		if err != nil {
			return nil, err
		}
		ioCreator = cio.LogURI(u)
	} else {
		ioCreator = cio.NewCreator(append([]cio.Opt{cio.WithStreams(stdinC, os.Stdout, os.Stderr)}, ioOpts...)...)
	}
	t, err := container.NewTask(ctx, ioCreator, opts...) // 通过containerd-shim-runc-v1 创建task
	if err != nil {
		return nil, err
	}
	stdinC.closer = func() {
		t.CloseIO(ctx, containerd.WithStdinCloser)
	}
	return t, nil
}

// GetNewTaskOpts resolves containerd.NewTaskOpts from cli.Context
func GetNewTaskOpts(context *cli.Context) []containerd.NewTaskOpts {
	if context.Bool("no-pivot") {
		return []containerd.NewTaskOpts{containerd.WithNoPivotRoot}
	}
	return nil
}
