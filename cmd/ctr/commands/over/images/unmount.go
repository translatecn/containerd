package images

import (
	"demo/over/leases"
	"fmt"

	"demo/cmd/ctr/commands"
	"demo/over/errdefs"
	"demo/over/mount"
	"github.com/urfave/cli"
)

var unmountCommand = cli.Command{
	Name:        "unmount",
	Usage:       "Unmount the image from the target",
	ArgsUsage:   "[flags] <target>",
	Description: "Unmount the image rootfs from the specified target.",
	Flags: append(append(commands.RegistryFlags, append(commands.SnapshotterFlags, commands.LabelFlag)...),
		cli.BoolFlag{
			Name:  "rm",
			Usage: "Remove the snapshot after a successful unmount",
		},
	),
	Action: func(context *cli.Context) error {
		var (
			target = context.Args().First()
		)
		if target == "" {
			return fmt.Errorf("please provide a target path to unmount from")
		}

		client, ctx, cancel, err := commands.NewClient(context)
		if err != nil {
			return err
		}
		defer cancel()

		if err := mount.UnmountAll(target, 0); err != nil {
			return err
		}

		if context.Bool("rm") {
			snapshotter := context.String("snapshotter")
			s := client.SnapshotService(snapshotter)
			if err := client.LeasesService().Delete(ctx, leases.Lease{ID: target}); err != nil && !errdefs.IsNotFound(err) {
				return fmt.Errorf("error deleting lease: %w", err)
			}
			if err := s.Remove(ctx, target); err != nil && !errdefs.IsNotFound(err) {
				return fmt.Errorf("error removing snapshot: %w", err)
			}
		}

		fmt.Fprintln(context.App.Writer, target)
		return nil
	},
}
