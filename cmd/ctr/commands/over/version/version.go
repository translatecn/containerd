package version

import (
	"demo/over/version"
	"fmt"
	"os"

	"demo/cmd/ctr/commands"
	"github.com/urfave/cli"
)

// Command is a cli command to output the client and containerd server version
var Command = cli.Command{
	Name:  "version",
	Usage: "Print the client and server versions",
	Action: func(context *cli.Context) error {
		if context.NArg() != 0 {
			return fmt.Errorf("extra arguments: %v", context.Args())
		}

		fmt.Println("Client:")
		fmt.Println("  Version: ", version.Version)
		fmt.Println("  Revision:", version.Revision)
		fmt.Println("  Go version:", version.GoVersion)
		fmt.Println("")
		client, ctx, cancel, err := commands.NewClient(context)
		if err != nil {
			return err
		}
		defer cancel()
		v, err := client.Version(ctx)
		if err != nil {
			return err
		}
		fmt.Println("Server:")
		fmt.Println("  Version: ", v.Version)
		fmt.Println("  Revision:", v.Revision)
		di, err := client.Server(ctx)
		if err != nil {
			return err
		}
		fmt.Println("  UUID:", di.UUID)
		if v.Version != version.Version {
			fmt.Fprintln(os.Stderr, "WARNING: version mismatch")
		}
		if v.Revision != version.Revision {
			fmt.Fprintln(os.Stderr, "WARNING: revision mismatch")
		}
		return nil
	},
}
