/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	internalapi "demo/pkg/api/cri"
	pb "demo/pkg/api/cri/v1"
	errorUtils "k8s.io/apimachinery/pkg/util/errors"
)

type sandboxByCreated []*pb.PodSandbox

func (a sandboxByCreated) Len() int      { return len(a) }
func (a sandboxByCreated) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a sandboxByCreated) Less(i, j int) bool {
	return a[i].CreatedAt > a[j].CreatedAt
}

var stopPodCommand = &cli.Command{
	Name:      "stopp",
	Usage:     "Stop one or more running pods",
	ArgsUsage: "POD-ID [POD-ID...]",
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
			err := StopPodSandbox(runtimeClient, id)
			if err != nil {
				return fmt.Errorf("stopping the pod sandbox %q: %w", id, err)
			}
		}
		return nil
	},
}

var removePodCommand = &cli.Command{
	Name:                   "rmp",
	Usage:                  "Remove one or more pods",
	ArgsUsage:              "POD-ID [POD-ID...]",
	UseShortOptionHandling: true,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "force",
			Aliases: []string{"f"},
			Usage:   "Force removal of the pod sandbox, disregarding if running",
		},
		&cli.BoolFlag{
			Name:    "all",
			Aliases: []string{"a"},
			Usage:   "Remove all pods",
		},
	},
	Action: func(ctx *cli.Context) error {
		runtimeClient, err := getRuntimeService(ctx, 0)
		if err != nil {
			return err
		}

		ids := ctx.Args().Slice()
		if ctx.Bool("all") {
			r, err := runtimeClient.ListPodSandbox(context.TODO(), nil)
			if err != nil {
				return err
			}
			ids = nil
			for _, sb := range r {
				ids = append(ids, sb.GetId())
			}
		}

		lenIDs := len(ids)
		if lenIDs == 0 {
			return cli.ShowSubcommandHelp(ctx)
		}

		funcs := []func() error{}
		for _, id := range ids {
			podId := id
			funcs = append(funcs, func() error {
				resp, err := runtimeClient.PodSandboxStatus(context.TODO(), podId, false)
				if err != nil {
					return fmt.Errorf("getting sandbox status of pod %q: %w", podId, err)
				}
				if resp.Status.State == pb.PodSandboxState_SANDBOX_READY {
					if ctx.Bool("force") {
						if err := StopPodSandbox(runtimeClient, podId); err != nil {
							return fmt.Errorf("stopping the pod sandbox %q failed: %w", podId, err)
						}
					} else {
						return fmt.Errorf("pod sandbox %q is running, please stop it first", podId)
					}
				}

				err = RemovePodSandbox(runtimeClient, podId)
				if err != nil {
					return fmt.Errorf("removing the pod sandbox %q: %w", podId, err)
				}

				return nil
			})
		}

		return errorUtils.AggregateGoroutines(funcs...)
	},
}

// StopPodSandbox sends a StopPodSandboxRequest to the server, and parses
// the returned StopPodSandboxResponse.
func StopPodSandbox(client internalapi.RuntimeService, id string) error {
	if id == "" {
		return fmt.Errorf("ID cannot be empty")
	}
	if err := client.StopPodSandbox(context.TODO(), id); err != nil {
		return err
	}

	fmt.Printf("Stopped sandbox %s\n", id)
	return nil
}

// RemovePodSandbox sends a RemovePodSandboxRequest to the server, and parses
// the returned RemovePodSandboxResponse.
func RemovePodSandbox(client internalapi.RuntimeService, id string) error {
	if id == "" {
		return fmt.Errorf("ID cannot be empty")
	}
	if err := client.RemovePodSandbox(context.TODO(), id); err != nil {
		return err
	}
	fmt.Printf("Removed sandbox %s\n", id)
	return nil
}

// marshalPodSandboxStatus converts pod sandbox status into string and converts
// the timestamps into readable format.
func marshalPodSandboxStatus(ps *pb.PodSandboxStatus) (string, error) {
	statusStr, err := protobufObjectToJSON(ps)
	if err != nil {
		return "", err
	}
	jsonMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(statusStr), &jsonMap)
	if err != nil {
		return "", err
	}
	jsonMap["createdAt"] = time.Unix(0, ps.CreatedAt).Format(time.RFC3339Nano)
	return marshalMapInOrder(jsonMap, *ps)
}

// PodSandboxStatus sends a PodSandboxStatusRequest to the server, and parses
// the returned PodSandboxStatusResponse.
func PodSandboxStatus(client internalapi.RuntimeService, id, output string, quiet bool, tmplStr string) error {
	verbose := !(quiet)
	if output == "" { // default to json output
		output = "json"
	}
	if id == "" {
		return fmt.Errorf("ID cannot be empty")
	}

	request := &pb.PodSandboxStatusRequest{
		PodSandboxId: id,
		Verbose:      verbose,
	}
	logrus.Debugf("PodSandboxStatusRequest: %v", request)
	r, err := client.PodSandboxStatus(context.TODO(), id, verbose)
	logrus.Debugf("PodSandboxStatusResponse: %v", r)
	if err != nil {
		return err
	}

	status, err := marshalPodSandboxStatus(r.Status)
	if err != nil {
		return err
	}
	switch output {
	case "json", "yaml", "go-template":
		return outputStatusInfo(status, r.Info, output, tmplStr)
	case "table": // table output is after this switch block
	default:
		return fmt.Errorf("output option cannot be %s", output)
	}

	// output in table format by default.
	fmt.Printf("ID: %s\n", r.Status.Id)
	if r.Status.Metadata != nil {
		if r.Status.Metadata.Name != "" {
			fmt.Printf("Name: %s\n", r.Status.Metadata.Name)
		}
		if r.Status.Metadata.Uid != "" {
			fmt.Printf("UID: %s\n", r.Status.Metadata.Uid)
		}
		if r.Status.Metadata.Namespace != "" {
			fmt.Printf("Namespace: %s\n", r.Status.Metadata.Namespace)
		}
		fmt.Printf("Attempt: %v\n", r.Status.Metadata.Attempt)
	}
	fmt.Printf("Status: %s\n", r.Status.State)
	ctm := time.Unix(0, r.Status.CreatedAt)
	fmt.Printf("Created: %v\n", ctm)

	if r.Status.Network != nil {
		fmt.Printf("IP Addresses: %v\n", r.Status.Network.Ip)
		for _, ip := range r.Status.Network.AdditionalIps {
			fmt.Printf("Additional IP: %v\n", ip.Ip)
		}
	}
	if r.Status.Labels != nil {
		for _, k := range getSortedKeys(r.Status.Labels) {
			fmt.Printf("\t%s -> %s\n", k, r.Status.Labels[k])
		}
	}
	if r.Status.Annotations != nil {
		for _, k := range getSortedKeys(r.Status.Annotations) {
			fmt.Printf("\t%s -> %s\n", k, r.Status.Annotations[k])
		}
	}
	if verbose {
		fmt.Printf("Info: %v\n", r.GetInfo())
	}

	return nil
}

func convertPodState(state pb.PodSandboxState) string {
	switch state {
	case pb.PodSandboxState_SANDBOX_READY:
		return "Ready"
	case pb.PodSandboxState_SANDBOX_NOTREADY:
		return "NotReady"
	default:
		log.Fatalf("Unknown pod state %q", state)
		return ""
	}
}

func getSandboxesRuntimeHandler(sandbox *pb.PodSandbox) string {
	if sandbox.RuntimeHandler == "" {
		return "(default)"
	}
	return sandbox.RuntimeHandler
}
