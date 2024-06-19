package tasks

import (
	"demo/cmd/ctr/commands"
	"github.com/urfave/cli"
)

var resumeCommand = cli.Command{
	Name:      "resume",
	Usage:     "Resume a paused container",
	ArgsUsage: "CONTAINER",
	Action: func(context *cli.Context) error {
		client, ctx, cancel, err := commands.NewClient(context)
		if err != nil {
			return err
		}
		defer cancel()
		container, err := client.LoadContainer(ctx, context.Args().First())
		if err != nil {
			return err
		}
		task, err := container.Task(ctx, nil)
		if err != nil {
			return err
		}
		return task.Resume(ctx)
	},
}
