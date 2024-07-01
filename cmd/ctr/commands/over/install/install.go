package install

import (
	"demo/cmd/ctr/commands"
	"demo/pkg/containerd"
	"github.com/urfave/cli"
)

// Command to install binary packages
var Command = cli.Command{
	Name:        "install",
	Usage:       "Install a new package",
	ArgsUsage:   "<ref>",
	Description: "install a new package",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "libs,l",
			Usage: "Install libs from the image",
		},
		cli.BoolFlag{
			Name:  "replace,r",
			Usage: "Replace any binaries or libs in the opt directory",
		},
		cli.StringFlag{
			Name:  "path",
			Usage: "Set an optional install path other than the managed opt directory",
		},
	},
	Action: func(context *cli.Context) error {
		client, ctx, cancel, err := commands.NewClient(context)
		if err != nil {
			return err
		}
		defer cancel()
		ref := context.Args().First()
		image, err := client.GetImage(ctx, ref)
		if err != nil {
			return err
		}
		var opts []containerd.InstallOpts
		if context.Bool("libs") {
			opts = append(opts, containerd.WithInstallLibs)
		}
		if context.Bool("replace") {
			opts = append(opts, containerd.WithInstallReplace)
		}
		if path := context.String("path"); path != "" {
			opts = append(opts, containerd.WithInstallPath(path))
		}
		return client.Install(ctx, image, opts...)
	},
}
