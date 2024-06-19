package tasks

import (
	gocontext "context"

	"github.com/urfave/cli"
)

type resizer interface {
	Resize(ctx gocontext.Context, w, h uint32) error
}

// Command is the cli command for managing tasks
var Command = cli.Command{
	Name:    "tasks",
	Usage:   "Manage tasks",
	Aliases: []string{"t", "task"},
	Subcommands: []cli.Command{
		attachCommand,
		checkpointCommand,
		execCommand,
		killCommand,
		// ------------
		resumeCommand,
		pauseCommand,
		metricsCommand,
		deleteCommand,
		psCommand,
		listCommand,
		startCommand,
	},
}
