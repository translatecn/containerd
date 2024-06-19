package images

import (
	"fmt"

	"github.com/urfave/cli"

	"demo/cmd/ctr/commands"
	"demo/over/errdefs"
	"demo/over/transfer/image"
)

var tagCommand = cli.Command{
	Name:        "tag",
	Usage:       "Tag an image",
	ArgsUsage:   "[flags] <source_ref> <target_ref> [<target_ref>, ...]",
	Description: `Tag an image for use in containerd.`,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "force",
			Usage: "Force target_ref to be created, regardless if it already exists",
		},
		cli.BoolTFlag{
			Name:  "local",
			Usage: "Run tag locally rather than through transfer API",
		},
	},
	Action: func(context *cli.Context) error {
		var (
			ref = context.Args().First()
		)
		if ref == "" {
			return fmt.Errorf("please provide an image reference to tag from")
		}
		if context.NArg() <= 1 {
			return fmt.Errorf("please provide an image reference to tag to")
		}

		client, ctx, cancel, err := commands.NewClient(context)
		if err != nil {
			return err
		}
		defer cancel()

		if !context.BoolT("local") {
			for _, targetRef := range context.Args()[1:] {
				err = client.Transfer(ctx, image.NewStore(ref), image.NewStore(targetRef))
				if err != nil {
					return err
				}
				fmt.Println(targetRef)
			}
			return nil
		}

		ctx, done, err := client.WithLease(ctx)
		if err != nil {
			return err
		}
		defer done(ctx)

		imageService := client.ImageService()
		image, err := imageService.Get(ctx, ref)
		if err != nil {
			return err
		}
		// Support multiple references for one command run
		for _, targetRef := range context.Args()[1:] {
			image.Name = targetRef
			// Attempt to create the image first
			if _, err = imageService.Create(ctx, image); err != nil {
				// If user has specified force and the image already exists then
				// delete the original image and attempt to create the new one
				if errdefs.IsAlreadyExists(err) && context.Bool("force") {
					if err = imageService.Delete(ctx, targetRef); err != nil {
						return err
					}
					if _, err = imageService.Create(ctx, image); err != nil {
						return err
					}
				} else {
					return err
				}
			}
			fmt.Println(targetRef)
		}
		return nil
	},
}
