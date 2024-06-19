package command

import (
	gocontext "context"
	dialer2 "demo/over/dialer"
	"demo/over/namespaces"
	"demo/over/protobuf/proto"
	"demo/over/protobuf/types"
	"fmt"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
	"io"
	"net"
	"os"
	"time"

	eventsapi "demo/over/api/services/events/v1"
	"demo/over/errdefs"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
)

var publishCommand = cli.Command{
	Name:  "publish",
	Usage: "Binary to publish events to containerd",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "namespace",
			Usage: "Namespace to publish to",
		},
		cli.StringFlag{
			Name:  "topic",
			Usage: "Topic of the event",
		},
	},
	Action: func(context *cli.Context) error {
		ctx := namespaces.WithNamespace(gocontext.Background(), context.String("namespace"))
		topic := context.String("topic")
		if topic == "" {
			return fmt.Errorf("topic required to publish event: %w", errdefs.ErrInvalidArgument)
		}
		payload, err := getEventPayload(os.Stdin)
		if err != nil {
			return err
		}
		client, err := connectEvents(context.GlobalString("address"))
		if err != nil {
			return err
		}
		if _, err := client.Publish(ctx, &eventsapi.PublishRequest{
			Topic: topic,
			Event: payload,
		}); err != nil {
			return errdefs.FromGRPC(err)
		}
		return nil
	},
}

func getEventPayload(r io.Reader) (*types.Any, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var any types.Any
	if err := proto.Unmarshal(data, &any); err != nil {
		return nil, err
	}
	return &any, nil
}

func connectEvents(address string) (eventsapi.EventsClient, error) {
	conn, err := connect(address, dialer2.ContextDialer)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %q: %w", address, err)
	}
	return eventsapi.NewEventsClient(conn), nil
}

func connect(address string, d func(gocontext.Context, string) (net.Conn, error)) (*grpc.ClientConn, error) {
	backoffConfig := backoff.DefaultConfig
	backoffConfig.MaxDelay = 3 * time.Second
	connParams := grpc.ConnectParams{
		Backoff: backoffConfig,
	}
	gopts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(d),
		grpc.FailOnNonTempDialError(true),
		grpc.WithConnectParams(connParams),
	}
	ctx, cancel := gocontext.WithTimeout(gocontext.Background(), 2*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, dialer2.DialAddress(address), gopts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %q: %w", address, err)
	}
	return conn, nil
}
