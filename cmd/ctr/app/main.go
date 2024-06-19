package app

import (
	"demo/cmd/ctr/commands/containers"
	"demo/cmd/ctr/commands/over/events"
	"demo/cmd/ctr/commands/over/images"
	"demo/cmd/ctr/commands/over/info"
	"demo/cmd/ctr/commands/over/install"
	"demo/cmd/ctr/commands/over/leases"
	namespacesCmd "demo/cmd/ctr/commands/over/namespaces"
	"demo/cmd/ctr/commands/over/oci"
	"demo/cmd/ctr/commands/over/plugins"
	"demo/cmd/ctr/commands/over/snapshots"
	versionCmd "demo/cmd/ctr/commands/over/version"
	"demo/cmd/ctr/commands/pprof"
	"demo/cmd/ctr/commands/run"
	"demo/cmd/ctr/commands/tasks"
	"demo/over/namespaces"
	"demo/over/version"
	"fmt"
	"io"

	"demo/cmd/ctr/commands/content"
	"demo/cmd/ctr/commands/sandboxes"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"google.golang.org/grpc/grpclog"
)

var extraCmds = []cli.Command{}

func init() {
	// Discard grpc logs so that they don't mess with our stdio
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(io.Discard, io.Discard, io.Discard))

	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Println(c.App.Name, version.Package, c.App.Version)
	}
}

// New returns a *cli.App instance.
func New() *cli.App {
	app := cli.NewApp()
	app.Name = "ctr"
	app.Version = version.Version
	app.Description = `
ctr is an unsupported debug and administrative client for interacting
with the containerd daemon. Because it is unsupported, the commands,
options, and operations are not guaranteed to be backward compatible or
stable from release to release of the containerd project.`
	app.Usage = `
        __
  _____/ /______
 / ___/ __/ ___/
/ /__/ /_/ /
\___/\__/_/

containerd CLI
`
	app.EnableBashCompletion = true
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable debug output in logs",
		},
		cli.StringFlag{
			Name:   "address, a",
			Usage:  "Address for containerd's GRPC server",
			Value:  "vm:6789",
			EnvVar: "CONTAINERD_ADDRESS",
		},
		cli.DurationFlag{
			Name:  "timeout",
			Usage: "Total timeout for ctr commands",
		},
		cli.DurationFlag{
			Name:  "connect-timeout",
			Usage: "Timeout for connecting to containerd",
		},
		cli.StringFlag{
			Name:   "namespace, n",
			Usage:  "Namespace to use with commands",
			Value:  namespaces.Default,
			EnvVar: namespaces.NamespaceEnvVar,
		},
	}
	app.Commands = append([]cli.Command{
		content.Command,
		sandboxes.Command,
		pprof.Command,
		run.Command,
		tasks.Command,
		containers.Command,
		install.Command,
		oci.Command,
		plugins.Command,
		snapshots.Command,
		events.Command,
		namespacesCmd.Command,
		versionCmd.Command,
		leases.Command,
		info.Command,
		images.Command,
	}, extraCmds...)
	app.Before = func(context *cli.Context) error {
		if context.GlobalBool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		}
		return nil
	}
	return app
}
