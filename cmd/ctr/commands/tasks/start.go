package tasks

import (
	"errors"

	"demo/cmd/ctr/commands"
	"demo/containerd"
	"demo/pkg/cio"
	"demo/pkg/console"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var startCommand = cli.Command{
	Name:      "start",
	Usage:     "Start a container that has been created",
	ArgsUsage: "CONTAINER",
	Flags: append(platformStartFlags, []cli.Flag{
		cli.BoolFlag{
			Name:  "null-io",
			Usage: "Send all IO to /dev/null",
		},
		cli.StringFlag{
			Name:  "log-uri",
			Usage: "Log uri",
		},
		cli.StringFlag{
			Name:  "fifo-dir",
			Usage: "Directory used for storing IO FIFOs",
		},
		cli.StringFlag{
			Name:  "pid-file",
			Usage: "File path to write the task's pid",
		},
		cli.BoolFlag{
			Name:  "detach,d",
			Usage: "Detach from the task after it has started execution",
		},
	}...),
	Action: func(context *cli.Context) error {
		var (
			err    error
			id     = context.Args().Get(0)
			detach = context.Bool("detach")
		)
		if id == "" {
			return errors.New("container id must be provided")
		}
		client, ctx, cancel, err := commands.NewClient(context)
		if err != nil {
			return err
		}
		defer cancel()
		container, err := client.LoadContainer(ctx, id)
		if err != nil {
			return err
		}

		spec, err := container.Spec(ctx)
		if err != nil {
			return err
		}
		var (
			tty    = spec.Process.Terminal
			opts   = GetNewTaskOpts(context)
			ioOpts = []cio.Opt{cio.WithFIFODir(context.String("fifo-dir"))}
		)
		var con console.Console
		if tty {
			con = console.Current()
			defer con.Reset()
			if err := con.SetRaw(); err != nil {
				return err
			}
		}

		task, err := NewTask(ctx, client, container, "", con, context.Bool("null-io"), context.String("log-uri"), ioOpts, opts...)
		if err != nil {
			return err
		}
		var statusC <-chan containerd.ExitStatus
		if !detach {
			defer task.Delete(ctx)
			if statusC, err = task.Wait(ctx); err != nil {
				return err
			}
		}
		if context.IsSet("pid-file") {
			if err := commands.WritePidFile(context.String("pid-file"), int(task.Pid())); err != nil {
				return err
			}
		}

		if err := task.Start(ctx); err != nil {
			return err
		}
		if detach {
			return nil
		}
		if tty {
			if err := HandleConsoleResize(ctx, task, con); err != nil {
				logrus.WithError(err).Error("console resize")
			}
		} else {
			sigc := commands.ForwardAllSignals(ctx, task)
			defer commands.StopCatch(sigc)
		}

		status := <-statusC
		code, _, err := status.Result()
		if err != nil {
			return err
		}
		if _, err := task.Delete(ctx); err != nil {
			return err
		}
		if code != 0 {
			return cli.NewExitError("", int(code))
		}
		return nil
	},
}
