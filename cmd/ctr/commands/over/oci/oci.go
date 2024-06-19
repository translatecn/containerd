package oci

import (
	"fmt"

	"github.com/urfave/cli"

	"demo/cmd/ctr/commands"
	"demo/over/containers"
	"demo/over/platforms"
	"demo/pkg/oci"
)

// Command is the parent for all OCI related tools under 'oci'
var Command = cli.Command{
	Name:  "oci",
	Usage: "OCI tools",
	Subcommands: []cli.Command{
		defaultSpecCommand,
	},
}

var defaultSpecCommand = cli.Command{
	Name:  "spec",
	Usage: "See the output of the default OCI spec",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "platform",
			Usage: "Platform of the spec to print (Examples: 'linux/arm64', 'windows/amd64')",
		},
	},
	Action: func(context *cli.Context) error {
		ctx, cancel := commands.AppContext(context)
		defer cancel()

		platform := platforms.DefaultString()
		if plat := context.String("platform"); plat != "" {
			platform = plat
		}

		spec, err := oci.GenerateSpecWithPlatform(ctx, nil, platform, &containers.Container{})
		if err != nil {
			return fmt.Errorf("failed to generate spec: %w", err)
		}

		commands.PrintAsJSON(spec)
		return nil
	},
}
