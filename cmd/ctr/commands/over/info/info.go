package info

import (
	"demo/cmd/ctr/commands"
	api "demo/pkg/api/services/introspection/v1"
	ptypes "demo/pkg/protobuf/types"
	"github.com/urfave/cli"
)

type Info struct {
	Server *api.ServerResponse `json:"server"`
}

// Command is a cli command to output the containerd server info
var Command = cli.Command{
	Name:  "info",
	Usage: "Print the server info",
	Action: func(context *cli.Context) error {
		client, ctx, cancel, err := commands.NewClient(context)
		if err != nil {
			return err
		}
		defer cancel()
		var info Info
		info.Server, err = client.IntrospectionService().Server(ctx, &ptypes.Empty{})
		if err != nil {
			return err
		}
		commands.PrintAsJSON(info)
		return nil
	},
}
