package images

import (
	leases2 "demo/over/leases"
	"fmt"
	"time"

	"demo/cmd/ctr/commands"
	"demo/containerd"
	"demo/over/errdefs"
	"demo/over/mount"
	"demo/over/platforms"
	"github.com/opencontainers/image-spec/identity"
	"github.com/urfave/cli"
)

var mountCommand = cli.Command{
	Name:      "mount",
	Usage:     "Mount an image to a target path",
	ArgsUsage: "[flags] <ref> <target>",
	Description: `Mount an image rootfs to a specified path.

When you are done, use the unmount command.
`,
	Flags: append(append(commands.RegistryFlags, append(commands.SnapshotterFlags, commands.LabelFlag)...),
		cli.BoolFlag{
			Name:  "rw",
			Usage: "Enable write support on the mount",
		},
		cli.StringFlag{
			Name:  "platform",
			Usage: "Mount the image for the specified platform",
			Value: platforms.DefaultString(),
		},
	),
	Action: func(context *cli.Context) (retErr error) {
		var (
			ref    = context.Args().First()
			target = context.Args().Get(1)
		)
		if ref == "" {
			return fmt.Errorf("please provide an image reference to mount")
		}
		if target == "" {
			return fmt.Errorf("please provide a target path to mount to")
		}

		client, ctx, cancel, err := commands.NewClient(context)
		if err != nil {
			return err
		}
		defer cancel()

		snapshotter := context.String("snapshotter")
		if snapshotter == "" {
			snapshotter = containerd.DefaultSnapshotter // ✅
		}

		ctx, done, err := client.WithLease(ctx,
			leases2.WithID(target),
			leases2.WithExpiration(24*time.Hour),
			leases2.WithLabels(map[string]string{
				"containerd.io/gc.ref.snapshot." + snapshotter: target,
			}),
		)
		if err != nil && !errdefs.IsAlreadyExists(err) {
			return err
		}

		defer func() {
			if retErr != nil && done != nil {
				done(ctx)
			}
		}()

		ps := context.String("platform")
		p, err := platforms.Parse(ps)
		if err != nil {
			return fmt.Errorf("unable to parse platform %s: %w", ps, err)
		}

		img, err := client.ImageService().Get(ctx, ref)
		if err != nil {
			return err
		}

		i := containerd.NewImageWithPlatform(client, img, platforms.Only(p))
		if err := i.Unpack(ctx, snapshotter); err != nil {
			return fmt.Errorf("error unpacking image: %w", err)
		}

		diffIDs, err := i.RootFS(ctx)
		if err != nil {
			return err
		}
		chainID := identity.ChainID(diffIDs).String()
		fmt.Println(chainID)

		s := client.SnapshotService(snapshotter)

		var mounts []mount.Mount
		if context.Bool("rw") {
			mounts, err = s.Prepare(ctx, target, chainID)
		} else {
			mounts, err = s.View(ctx, target, chainID)
		}
		if err != nil {
			if errdefs.IsAlreadyExists(err) {
				mounts, err = s.Mounts(ctx, target)
			}
			if err != nil {
				return err
			}
		}

		if err := mount.All(mounts, target); err != nil {
			if err := s.Remove(ctx, target); err != nil && !errdefs.IsNotFound(err) {
				fmt.Fprintln(context.App.ErrWriter, "Error cleaning up snapshot after mount error:", err)
			}
			return err
		}

		fmt.Fprintln(context.App.Writer, target)
		return nil
	},
}
