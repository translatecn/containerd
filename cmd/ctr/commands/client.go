package commands

import (
	gocontext "context"
	epoch2 "demo/over/epoch"
	"demo/over/log"
	"demo/over/namespaces"
	ptypes "demo/over/protobuf/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"os"
	"strconv"

	"demo/containerd"
	"github.com/urfave/cli"
)

// AppContext returns the context for a command. Should only be called once per
// command, near the start.
//
// This will ensure the namespace is picked up and set the timeout, if one is
// defined.
func AppContext(context *cli.Context) (gocontext.Context, gocontext.CancelFunc) {
	var (
		ctx       = gocontext.Background()
		timeout   = context.GlobalDuration("timeout")
		namespace = context.GlobalString("namespace")
		cancel    gocontext.CancelFunc
	)
	ctx = namespaces.WithNamespace(ctx, namespace)
	if timeout > 0 {
		ctx, cancel = gocontext.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = gocontext.WithCancel(ctx)
	}
	if tm, err := epoch2.SourceDateEpoch(); err != nil {
		log.L.WithError(err).Warn("Failed to read SOURCE_DATE_EPOCH")
	} else if tm != nil {
		log.L.Debugf("Using SOURCE_DATE_EPOCH: %v", tm)
		ctx = epoch2.WithSourceDateEpoch(ctx, tm)
	}
	return ctx, cancel
}

// NewClient returns a new containerd client
func NewClient(context *cli.Context, opts ...containerd.ClientOpt) (*containerd.Client, gocontext.Context, gocontext.CancelFunc, error) {
	timeoutOpt := containerd.WithTimeout(context.GlobalDuration("connect-timeout"))
	opts = append(opts, timeoutOpt)
	conn, err := grpc.Dial(context.GlobalString("address"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, nil, err
	}
	client, err := containerd.NewWithConn(conn)
	if err != nil {
		return nil, nil, nil, err
	}
	ctx, cancel := AppContext(context)
	var suppressDeprecationWarnings bool
	if s := os.Getenv("CONTAINERD_SUPPRESS_DEPRECATION_WARNINGS"); s != "" {
		suppressDeprecationWarnings, err = strconv.ParseBool(s)
		if err != nil {
			log.L.WithError(err).Warn("Failed to parse CONTAINERD_SUPPRESS_DEPRECATION_WARNINGS=" + s)
		}
	}
	if !suppressDeprecationWarnings {
		resp, err := client.IntrospectionService().Server(ctx, &ptypes.Empty{})
		if err != nil {
			log.L.WithError(err).Warn("Failed to check deprecations")
		} else {
			for _, d := range resp.Deprecations {
				log.L.Warn("DEPRECATION: " + d.Message)
			}
		}
	}
	return client, ctx, cancel, nil
}
