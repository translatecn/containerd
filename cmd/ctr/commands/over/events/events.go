package events

import (
	"demo/cmd/ctr/commands"
	"demo/over/events"
	"demo/over/log"
	"demo/over/typeurl/v2"
	"encoding/json"
	"fmt"
	"github.com/urfave/cli"

	// Register grpc event types
	_ "demo/over/api/events"
)

// Command is the cli command for displaying containerd events
var Command = cli.Command{
	Name:    "events",
	Aliases: []string{"event"},
	Usage:   "Display containerd events",
	Action: func(context *cli.Context) error {
		client, ctx, cancel, err := commands.NewClient(context)
		if err != nil {
			return err
		}
		defer cancel()
		eventsClient := client.EventService()
		eventsCh, errCh := eventsClient.Subscribe(ctx, context.Args()...)
		for {
			var e *events.Envelope
			select {
			case e = <-eventsCh:
			case err = <-errCh:
				return err
			}
			if e != nil {
				var out []byte
				if e.Event != nil {
					v, err := typeurl.UnmarshalAny(e.Event)
					if err != nil {
						log.G(ctx).WithError(err).Warn("cannot unmarshal an event from Any")
						continue
					}
					out, err = json.Marshal(v)
					if err != nil {
						log.G(ctx).WithError(err).Warn("cannot marshal Any into JSON")
						continue
					}
				}
				if _, err := fmt.Println(
					e.Timestamp,
					e.Namespace,
					e.Topic,
					string(out),
				); err != nil {
					return err
				}
			}
		}
	},
}
