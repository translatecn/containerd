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
	"fmt"
	"io"

	"demo/cmd/ctr/commands/content"
	"demo/cmd/ctr/commands/over_events"
	"demo/cmd/ctr/commands/over_install"
	"demo/cmd/ctr/commands/over_leases"
	namespacesCmd "demo/cmd/ctr/commands/over_namespaces"
	ociCmd "demo/cmd/ctr/commands/over_oci"
	"demo/cmd/ctr/commands/over_plugins"
	"demo/cmd/ctr/commands/over_snapshots"
	versionCmd "demo/cmd/ctr/commands/over_version"
	"demo/cmd/ctr/commands/pprof"
	"demo/cmd/ctr/commands/tasks"
	"demo/others/imgcrypt/cmd/ctr/commands/containers"
	"demo/others/imgcrypt/cmd/ctr/commands/images"
	"demo/others/imgcrypt/cmd/ctr/commands/run"
	"demo/pkg/defaults"
	"demo/pkg/namespaces"
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
			Usage: "enable debug output in logs",
		},
		cli.StringFlag{
			Name:   "address, a",
			Usage:  "address for containerd's GRPC server",
			Value:  defaults.DefaultAddress,
			EnvVar: "CONTAINERD_ADDRESS",
		},
		cli.DurationFlag{
			Name:  "timeout",
			Usage: "total timeout for ctr commands",
		},
		cli.DurationFlag{
			Name:  "connect-timeout",
			Usage: "timeout for connecting to containerd",
		},
		cli.StringFlag{
			Name:   "namespace, n",
			Usage:  "namespace to use with commands",
			Value:  namespaces.Default,
			EnvVar: namespaces.NamespaceEnvVar,
		},
	}
	app.Commands = append([]cli.Command{
		over_plugins.Command,
		versionCmd.Command,
		containers.Command,
		content.Command,
		over_events.Command,
		images.Command,
		over_leases.Command,
		namespacesCmd.Command,
		pprof.Command,
		run.Command,
		over_snapshots.Command,
		tasks.Command,
		over_install.Command,
		ociCmd.Command,
	}, extraCmds...)
	app.Before = func(context *cli.Context) error {
		if context.GlobalBool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		}
		return nil
	}
	return app
}
