/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package app

import (
	"demo/cmd/ctr/commands/over_events"
	"demo/cmd/ctr/commands/over_images"
	"demo/cmd/ctr/commands/over_install"
	"demo/cmd/ctr/commands/over_leases"
	namespacesCmd "demo/cmd/ctr/commands/over_namespaces"
	"demo/cmd/ctr/commands/over_plugins"
	"demo/cmd/ctr/commands/over_snapshots"
	versionCmd "demo/cmd/ctr/commands/over_version"
	"demo/pkg/namespaces"
	"fmt"
	"io"

	"demo/cmd/ctr/commands/containers"
	"demo/cmd/ctr/commands/content"
	"demo/cmd/ctr/commands/over_info"
	ociCmd "demo/cmd/ctr/commands/over_oci"
	"demo/cmd/ctr/commands/pprof"
	"demo/cmd/ctr/commands/run"
	"demo/cmd/ctr/commands/sandboxes"
	"demo/cmd/ctr/commands/tasks"
	"demo/version"
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
			Value:  "172.16.244.147:6789",
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
		containers.Command,
		content.Command,
		pprof.Command,
		run.Command,
		tasks.Command,
		sandboxes.Command,

		over_install.Command,
		ociCmd.Command,
		over_plugins.Command,
		over_snapshots.Command,
		over_events.Command,
		namespacesCmd.Command,
		versionCmd.Command,
		over_leases.Command,
		over_info.Command,
		over_images.Command,
	}, extraCmds...)
	app.Before = func(context *cli.Context) error {
		if context.GlobalBool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		}
		return nil
	}
	return app
}
