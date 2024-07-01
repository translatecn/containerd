package tasks

import (
	"demo/config/runc"
	"demo/pkg/api/runctypes"
	"demo/pkg/plugin"
	"errors"
	"fmt"

	"demo/cmd/ctr/commands"
	"demo/pkg/containerd"
	"github.com/urfave/cli"
)

var checkpointCommand = cli.Command{
	Name:      "checkpoint",
	Usage:     "Checkpoint a container",
	ArgsUsage: "[flags] CONTAINER",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "exit",
			Usage: "Stop the container after the checkpoint",
		},
		cli.StringFlag{
			Name:  "image-path",
			Usage: "Path to criu image files",
		},
		cli.StringFlag{
			Name:  "work-path",
			Usage: "Path to criu work files and logs",
		},
	},
	Action: func(context *cli.Context) error {
		id := context.Args().First()
		if id == "" {
			return errors.New("container id must be provided")
		}
		client, ctx, cancel, err := commands.NewClient(context, containerd.WithDefaultRuntime(context.String("runtime")))
		if err != nil {
			return err
		}
		defer cancel()
		container, err := client.LoadContainer(ctx, id)
		if err != nil {
			return err
		}
		task, err := container.Task(ctx, nil)
		if err != nil {
			return err
		}
		info, err := container.Info(ctx)
		if err != nil {
			return err
		}
		opts := []containerd.CheckpointTaskOpts{withCheckpointOpts(info.Runtime.Name, context)}
		checkpoint, err := task.Checkpoint(ctx, opts...)
		if err != nil {
			return err
		}
		if context.String("image-path") == "" {
			fmt.Println(checkpoint.Name())
		}
		return nil
	},
}

// withCheckpointOpts only suitable for runc runtime now
func withCheckpointOpts(rt string, context *cli.Context) containerd.CheckpointTaskOpts {
	return func(r *containerd.CheckpointTaskInfo) error {
		imagePath := context.String("image-path")
		workPath := context.String("work-path")

		switch rt {
		case plugin.RuntimeRuncV1, plugin.RuntimeRuncV2:
			if r.Options == nil {
				r.Options = &runc.CheckpointOptions{}
			}
			opts, _ := r.Options.(*runc.CheckpointOptions)

			if context.Bool("exit") {
				opts.Exit = true
			}
			if imagePath != "" {
				opts.ImagePath = imagePath
			}
			if workPath != "" {
				opts.WorkPath = workPath
			}
		case plugin.RuntimeLinuxV1:
			if r.Options == nil {
				r.Options = &runctypes.CheckpointOptions{}
			}
			opts, _ := r.Options.(*runctypes.CheckpointOptions)

			if context.Bool("exit") {
				opts.Exit = true
			}
			if imagePath != "" {
				opts.ImagePath = imagePath
			}
			if workPath != "" {
				opts.WorkPath = workPath
			}
		}
		return nil
	}
}
