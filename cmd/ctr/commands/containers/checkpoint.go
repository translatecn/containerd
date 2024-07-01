package containers

import (
	"errors"
	"fmt"

	"demo/cmd/ctr/commands"
	"demo/pkg/containerd"
	"demo/pkg/errdefs"
	"github.com/urfave/cli"
)

var checkpointCommand = cli.Command{
	Name:      "checkpoint",
	Usage:     "Checkpoint a container",
	ArgsUsage: "CONTAINER REF",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "rw",
			Usage: "Include the rw layer in the checkpoint",
		},
		cli.BoolFlag{
			Name:  "image",
			Usage: "Include the image in the checkpoint",
		},
		cli.BoolFlag{
			Name:  "task",
			Usage: "Checkpoint container task",
		},
	},
	Action: func(context *cli.Context) error {
		id := context.Args().First()
		if id == "" {
			return errors.New("container id must be provided")
		}
		ref := context.Args().Get(1)
		if ref == "" {
			return errors.New("ref must be provided")
		}
		client, ctx, cancel, err := commands.NewClient(context)
		if err != nil {
			return err
		}
		defer cancel()
		opts := []containerd.CheckpointOpts{
			containerd.WithCheckpointRuntime,
		}

		if context.Bool("image") {
			opts = append(opts, containerd.WithCheckpointImage)
		}
		if context.Bool("rw") {
			opts = append(opts, containerd.WithCheckpointRW)
		}
		if context.Bool("task") {
			opts = append(opts, containerd.WithCheckpointTask)
		}
		container, err := client.LoadContainer(ctx, id)
		if err != nil {
			return err
		}
		task, err := container.Task(ctx, nil)
		if err != nil {
			if !errdefs.IsNotFound(err) {
				return err
			}
		}
		// pause if running
		if task != nil {
			if err := task.Pause(ctx); err != nil {
				return err
			}
			defer func() {
				if err := task.Resume(ctx); err != nil {
					fmt.Println(fmt.Errorf("error resuming task: %w", err))
				}
			}()
		}

		if _, err := container.Checkpoint(ctx, ref, opts...); err != nil {
			return err
		}

		return nil
	},
}
