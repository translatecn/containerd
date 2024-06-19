package main

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	internalapi "k8s.io/cri-api/pkg/apis"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/kubernetes/pkg/kubelet/cri/remote"
	"time"
)

var podStatusCommand = &cli.Command{
	Name:                   "inspectp",
	Usage:                  "Display the status of one or more pods",
	ArgsUsage:              "POD-ID [POD-ID...]",
	UseShortOptionHandling: true,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Output format, One of: json|yaml|go-template|table",
		},
		&cli.BoolFlag{
			Name:    "quiet",
			Aliases: []string{"q"},
			Usage:   "Do not show verbose information",
		},
		&cli.StringFlag{
			Name:  "template",
			Usage: "The template string is only used when output is go-template; The Template format is golang template",
		},
	},
	Action: func(c *cli.Context) error {
		if c.NArg() == 0 {
			return cli.ShowSubcommandHelp(c)
		}
		runtimeClient, err := getRuntimeService(c, 0)
		if err != nil {
			return err
		}
		for i := 0; i < c.NArg(); i++ {
			id := c.Args().Get(i)

			err := PodSandboxStatus(runtimeClient, id, c.String("output"), c.Bool("quiet"), c.String("template"))
			if err != nil {
				return fmt.Errorf("getting the pod sandbox status for %q: %w", id, err)
			}
		}
		return nil
	},
}
var overRuntimeVersionCommand = &cli.Command{
	Name:  "version",
	Usage: "Display runtime version information",
	Action: func(c *cli.Context) error {
		if c.NArg() != 0 {
			cli.ShowSubcommandHelp(c)
		}

		runtimeClient, err := getRuntimeService(c, 0)
		if err != nil {
			return err
		}
		if err := Version(runtimeClient, string(remote.CRIVersionV1)); err != nil {
			return fmt.Errorf("getting the runtime version: %w", err)
		}
		return nil
	},
}

func getRuntimeService(context *cli.Context, timeout time.Duration) (res internalapi.RuntimeService, err error) {
	if RuntimeEndpointIsSet && RuntimeEndpoint == "" {
		return nil, fmt.Errorf("--runtime-endpoint is not set")
	}
	logrus.Debug("get runtime connection")
	// Check if a custom timeout is provided.
	t := time.Hour
	// If no EP set then use the default endpoint types
	fmt.Println(RuntimeEndpoint)
	return remote.NewRemoteRuntimeService(RuntimeEndpoint, t, nil)
}

var runtimeStatusCommand = &cli.Command{
	Name:                   "info",
	Usage:                  "Display information of the container runtime",
	ArgsUsage:              "",
	UseShortOptionHandling: true,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Value:   "json",
			Usage:   "Output format, One of: json|yaml|go-template",
		},
		&cli.BoolFlag{
			Name:    "quiet",
			Aliases: []string{"q"},
			Usage:   "Do not show verbose information",
		},
		&cli.StringFlag{
			Name:  "template",
			Usage: "The template string is only used when output is go-template; The Template format is golang template",
		},
	},
	Action: func(c *cli.Context) error {
		if c.NArg() != 0 {
			return cli.ShowSubcommandHelp(c)
		}

		runtimeClient, err := getRuntimeService(c, 0)
		if err != nil {
			return err
		}

		if err = Info(c, runtimeClient); err != nil {
			return fmt.Errorf("getting status of runtime: %w", err)
		}
		return nil
	},
}

// Info sends a StatusRequest to the server, and parses the returned StatusResponse.
func Info(cliContext *cli.Context, client internalapi.RuntimeService) error {
	request := &pb.StatusRequest{Verbose: !cliContext.Bool("quiet")}
	logrus.Debugf("StatusRequest: %v", request)
	r, err := client.Status(context.TODO(), request.Verbose)
	logrus.Debugf("StatusResponse: %v", r)
	if err != nil {
		return err
	}

	status, err := protobufObjectToJSON(r.Status)
	if err != nil {
		return err
	}
	return outputStatusInfo(status, r.Info, cliContext.String("output"), cliContext.String("template"))
}
