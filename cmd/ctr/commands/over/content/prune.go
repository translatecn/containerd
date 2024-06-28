package content

import (
	leases2 "demo/pkg/leases"
	"demo/pkg/log"
	"strings"
	"time"
	"unicode"

	"demo/cmd/ctr/commands"
	"demo/pkg/content"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

const (
	layerPrefix   = "containerd.io/gc.ref.content.l."
	contentPrefix = "containerd.io/gc.ref.content."
)

var pruneFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "async",
		Usage: "Allow garbage collection to cleanup asynchronously",
	},
	cli.BoolFlag{
		Name:  "dry",
		Usage: "Just show updates without applying (enables debug logging)",
	},
}

var pruneCommand = cli.Command{
	Name:  "prune",
	Usage: "Prunes content from the content store",
	Subcommands: cli.Commands{
		pruneReferencesCommand,
	},
}

var pruneReferencesCommand = cli.Command{
	Name:  "references",
	Usage: "Prunes preference labels from the content store (layers only by default)",
	Flags: pruneFlags,
	Action: func(clicontext *cli.Context) error {
		client, ctx, cancel, err := commands.NewClient(clicontext)
		if err != nil {
			return err
		}
		defer cancel()

		dryRun := clicontext.Bool("dry")
		if dryRun {
			log.G(ctx).Logger.SetLevel(logrus.DebugLevel)
			log.G(ctx).Debug("dry run, no changes will be applied")
		}

		var deleteOpts []leases2.DeleteOpt
		if !clicontext.Bool("async") {
			deleteOpts = append(deleteOpts, leases2.SynchronousDelete)
		}

		cs := client.ContentStore()
		if err := cs.Walk(ctx, func(info content.Info) error {
			var fields []string

			for k := range info.Labels {
				if isLayerLabel(k) {
					log.G(ctx).WithFields(log.Fields{
						"digest": info.Digest,
						"label":  k,
					}).Debug("Removing label")
					if dryRun {
						continue
					}
					fields = append(fields, "labels."+k)
					delete(info.Labels, k)
				}
			}

			if len(fields) == 0 {
				return nil
			}

			_, err := cs.Update(ctx, info, fields...)
			return err
		}); err != nil {
			return err
		}

		ls := client.LeasesService()
		l, err := ls.Create(ctx, leases2.WithRandomID(), leases2.WithExpiration(time.Hour))
		if err != nil {
			return err
		}
		return ls.Delete(ctx, l, deleteOpts...)
	},
}

func isLayerLabel(key string) bool {
	if strings.HasPrefix(key, layerPrefix) {
		return true
	}
	if !strings.HasPrefix(key, contentPrefix) {
		return false
	}

	// handle legacy labels which used content prefix and index (0 always for config)
	key = key[len(contentPrefix):]
	if isInteger(key) && key != "0" {
		return true
	}

	return false
}

func isInteger(key string) bool {
	for _, r := range key {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
