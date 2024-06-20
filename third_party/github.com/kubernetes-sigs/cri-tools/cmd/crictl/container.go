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
	"errors"
	"fmt"
	goruntime "runtime"
	"sort"
	"time"

	"demo/third_party/k8s.io/kubernetes/pkg/kubelet/types"

	internalapi "demo/over/api/cri"
	pb "demo/over/api/cri/v1"
	"github.com/docker/go-units"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

type containerByCreated []*pb.Container

func (a containerByCreated) Len() int      { return len(a) }
func (a containerByCreated) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a containerByCreated) Less(i, j int) bool {
	return a[i].CreatedAt > a[j].CreatedAt
}

type createOptions struct {
	// podID of the container
	podID string

	// the config and pod options
	*runOptions
}

type runOptions struct {
	// configPath is path to the config for container
	configPath string

	// podConfig is path to the config for sandbox
	podConfig string

	// the create timeout
	timeout time.Duration

	// the image pull options
	*pullOptions
}

type pullOptions struct {
	// pull the image on container creation; overrides default
	withPull bool

	// creds is string in the format `USERNAME[:PASSWORD]` for accessing the
	// registry during image pull
	creds string

	// auth is a base64 encoded 'USERNAME[:PASSWORD]' string used for
	// authentication with a registry when pulling an image
	auth string

	// Username to use for accessing the registry
	// password will be requested on the command line
	username string
}

var createPullFlags = []cli.Flag{
	&cli.BoolFlag{
		Name:  "no-pull",
		Usage: "Do not pull the image on container creation (overrides pull-image-on-create=true in config)",
	},
	&cli.BoolFlag{
		Name:  "with-pull",
		Usage: "Pull the image on container creation (overrides pull-image-on-create=false in config)",
	},
	&cli.StringFlag{
		Name:  "creds",
		Value: "",
		Usage: "Use `USERNAME[:PASSWORD]` for accessing the registry",
	},
	&cli.StringFlag{
		Name:  "auth",
		Value: "",
		Usage: "Use `AUTH_STRING` for accessing the registry. AUTH_STRING is a base64 encoded 'USERNAME[:PASSWORD]'",
	},
	&cli.StringFlag{
		Name:  "username",
		Value: "",
		Usage: "Use `USERNAME` for accessing the registry. The password will be requested on the command line",
	},
}

var runPullFlags = []cli.Flag{
	&cli.BoolFlag{
		Name:  "no-pull",
		Usage: "Do not pull the image (overrides disable-pull-on-run=false in config)",
	},
	&cli.BoolFlag{
		Name:  "with-pull",
		Usage: "Pull the image (overrides disable-pull-on-run=true in config)",
	},
	&cli.StringFlag{
		Name:  "creds",
		Value: "",
		Usage: "Use `USERNAME[:PASSWORD]` for accessing the registry",
	},
	&cli.StringFlag{
		Name:  "auth",
		Value: "",
		Usage: "Use `AUTH_STRING` for accessing the registry. AUTH_STRING is a base64 encoded 'USERNAME[:PASSWORD]'",
	},
	&cli.StringFlag{
		Name:  "username",
		Value: "",
		Usage: "Use `USERNAME` for accessing the registry. password will be requested",
	},
}

var updateContainerCommand = &cli.Command{
	Name:      "update",
	Usage:     "Update one or more running containers",
	ArgsUsage: "CONTAINER-ID [CONTAINER-ID...]",
	Flags: []cli.Flag{
		&cli.Int64Flag{
			Name:  "cpu-count",
			Usage: "(Windows only) Number of CPUs available to the container",
		},
		&cli.Int64Flag{
			Name:  "cpu-maximum",
			Usage: "(Windows only) Portion of CPU cycles specified as a percentage * 100",
		},
		&cli.Int64Flag{
			Name:  "cpu-period",
			Usage: "CPU CFS period to be used for hardcapping (in usecs). 0 to use system default",
		},
		&cli.Int64Flag{
			Name:  "cpu-quota",
			Usage: "CPU CFS hardcap limit (in usecs). Allowed cpu time in a given period",
		},
		&cli.Int64Flag{
			Name:  "cpu-share",
			Usage: "CPU shares (relative weight vs. other containers)",
		},
		&cli.Int64Flag{
			Name:  "memory",
			Usage: "Memory limit (in bytes)",
		},
		&cli.StringFlag{
			Name:  "cpuset-cpus",
			Usage: "CPU(s) to use",
		},
		&cli.StringFlag{
			Name:  "cpuset-mems",
			Usage: "Memory node(s) to use",
		},
	},
	Action: func(c *cli.Context) error {
		if c.NArg() == 0 {
			return fmt.Errorf("ID cannot be empty")
		}
		runtimeClient, err := getRuntimeService(c, 0)
		if err != nil {
			return err
		}

		options := updateOptions{
			CPUCount:           c.Int64("cpu-count"),
			CPUMaximum:         c.Int64("cpu-maximum"),
			CPUPeriod:          c.Int64("cpu-period"),
			CPUQuota:           c.Int64("cpu-quota"),
			CPUShares:          c.Int64("cpu-share"),
			CpusetCpus:         c.String("cpuset-cpus"),
			CpusetMems:         c.String("cpuset-mems"),
			MemoryLimitInBytes: c.Int64("memory"),
		}

		for i := 0; i < c.NArg(); i++ {
			containerID := c.Args().Get(i)
			if err := UpdateContainerResources(runtimeClient, containerID, options); err != nil {
				return fmt.Errorf("updating container resources for %q: %w", containerID, err)
			}
		}
		return nil
	},
}

var removeContainerCommand = &cli.Command{
	Name:                   "rm",
	Usage:                  "Remove one or more containers",
	ArgsUsage:              "CONTAINER-ID [CONTAINER-ID...]",
	UseShortOptionHandling: true,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "force",
			Aliases: []string{"f"},
			Usage:   "Force removal of the container, disregarding if running",
		},
		&cli.BoolFlag{
			Name:    "all",
			Aliases: []string{"a"},
			Usage:   "Remove all containers",
		},
	},
	Action: func(ctx *cli.Context) error {
		runtimeClient, err := getRuntimeService(ctx, 0)
		if err != nil {
			return err
		}

		ids := ctx.Args().Slice()
		if ctx.Bool("all") {
			r, err := runtimeClient.ListContainers(context.TODO(), nil)
			if err != nil {
				return err
			}
			ids = nil
			for _, ctr := range r {
				ids = append(ids, ctr.GetId())
			}
		}

		if len(ids) == 0 {
			return cli.ShowSubcommandHelp(ctx)
		}

		errored := false
		for _, id := range ids {
			resp, err := runtimeClient.ContainerStatus(context.TODO(), id, false)
			if err != nil {
				logrus.Error(err)
				errored = true
				continue
			}
			if resp.GetStatus().GetState() == pb.ContainerState_CONTAINER_RUNNING {
				if ctx.Bool("force") {
					if err := StopContainer(runtimeClient, id, 0); err != nil {
						logrus.Errorf("stopping the container %q failed: %v", id, err)
						errored = true
						continue
					}
				} else {
					logrus.Errorf("container %q is running, please stop it first", id)
					errored = true
					continue
				}
			}

			err = RemoveContainer(runtimeClient, id)
			if err != nil {
				logrus.Errorf("removing container %q failed: %v", id, err)
				errored = true
				continue
			}
		}

		if errored {
			return fmt.Errorf("unable to remove container(s)")
		}
		return nil
	},
}

var containerStatusCommand = &cli.Command{
	Name:      "inspect",
	Usage:     "Display the status of one or more containers",
	ArgsUsage: "CONTAINER-ID [CONTAINER-ID...]",
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
			return fmt.Errorf("ID cannot be empty")
		}
		runtimeClient, err := getRuntimeService(c, 0)
		if err != nil {
			return err
		}

		for i := 0; i < c.NArg(); i++ {
			containerID := c.Args().Get(i)
			if err := ContainerStatus(runtimeClient, containerID, c.String("output"), c.String("template"), c.Bool("quiet")); err != nil {
				return fmt.Errorf("getting the status of the container %q: %w", containerID, err)
			}
		}
		return nil
	},
}

var runContainerCommand = &cli.Command{
	Name:      "run",
	Usage:     "Run a new container inside a sandbox",
	ArgsUsage: "container-config.[json|yaml] pod-config.[json|yaml]",
	Flags: append(runPullFlags, &cli.StringFlag{
		Name:    "runtime",
		Aliases: []string{"r"},
		Usage:   "Runtime handler to use. Available options are defined by the container runtime.",
	}, &cli.DurationFlag{
		Name:    "timeout",
		Aliases: []string{"t"},
		Usage:   "Seconds to wait for a container create request before cancelling the request",
	}),

	Action: func(c *cli.Context) (err error) {
		if c.Args().Len() != 2 {
			return cli.ShowSubcommandHelp(c)
		}
		if c.Bool("no-pull") == true && c.Bool("with-pull") == true {
			return errors.New("confict: no-pull and with-pull are both set")
		}

		withPull := (!DisablePullOnRun && !c.Bool("no-pull")) || c.Bool("with-pull")

		var imageClient internalapi.ImageManagerService
		if withPull {
			imageClient, err = getImageService(c)
			if err != nil {
				return err
			}
		}

		opts := runOptions{
			configPath: c.Args().Get(0),
			podConfig:  c.Args().Get(1),
			pullOptions: &pullOptions{
				withPull: withPull,
				creds:    c.String("creds"),
				auth:     c.String("auth"),
				username: c.String("username"),
			},
			timeout: c.Duration("timeout"),
		}

		runtimeClient, err := getRuntimeService(c, opts.timeout)
		if err != nil {
			return err
		}

		if err = RunContainer(imageClient, runtimeClient, opts, c.String("runtime")); err != nil {
			return fmt.Errorf("running container: %w", err)
		}
		return nil
	},
}

var checkpointContainerCommand = &cli.Command{
	Name:                   "checkpoint",
	Usage:                  "Checkpoint one or more running containers",
	ArgsUsage:              "CONTAINER-ID [CONTAINER-ID...]",
	UseShortOptionHandling: true,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "export",
			Aliases: []string{"e"},
			Usage:   "Specify the name of the archive used to export the checkpoint image.",
		},
	},
	Action: func(c *cli.Context) error {
		if c.NArg() == 0 {
			return fmt.Errorf("ID cannot be empty")
		}
		runtimeClient, err := getRuntimeService(c, 0)
		if err != nil {
			return err
		}

		for i := 0; i < c.NArg(); i++ {
			containerID := c.Args().Get(i)
			err := CheckpointContainer(
				runtimeClient,
				containerID,
				c.String("export"),
			)
			if err != nil {
				return fmt.Errorf("checkpointing the container %q failed: %w", containerID, err)
			}
		}
		return nil
	},
}

// RunContainer starts a container in the provided sandbox
func RunContainer(
	iClient internalapi.ImageManagerService,
	rClient internalapi.RuntimeService,
	opts runOptions,
	runtime string,
) error {
	// Create the pod
	podSandboxConfig, err := loadPodSandboxConfig(opts.podConfig)
	if err != nil {
		return fmt.Errorf("load podSandboxConfig: %w", err)
	}
	// set the timeout for the RunPodSandbox request to 0, because the
	// timeout option is documented as being for container creation.
	podID, err := RunPodSandbox(rClient, podSandboxConfig, runtime)
	if err != nil {
		return fmt.Errorf("run pod sandbox: %w", err)
	}

	// Create the container
	containerOptions := createOptions{podID, &opts}
	ctrID, err := CreateContainer(iClient, rClient, containerOptions)
	if err != nil {
		return fmt.Errorf("creating container failed: %w", err)
	}

	// Start the container
	err = StartContainer(rClient, ctrID)
	if err != nil {
		return fmt.Errorf("starting the container %q: %w", ctrID, err)
	}
	return nil
}

type updateOptions struct {
	// (Windows only) Number of CPUs available to the container.
	CPUCount int64
	// (Windows only) Portion of CPU cycles specified as a percentage * 100.
	CPUMaximum int64
	// CPU CFS (Completely Fair Scheduler) period. Default: 0 (not specified).
	CPUPeriod int64
	// CPU CFS (Completely Fair Scheduler) quota. Default: 0 (not specified).
	CPUQuota int64
	// CPU shares (relative weight vs. other containers). Default: 0 (not specified).
	CPUShares int64
	// Memory limit in bytes. Default: 0 (not specified).
	MemoryLimitInBytes int64
	// OOMScoreAdj adjusts the oom-killer score. Default: 0 (not specified).
	OomScoreAdj int64
	// CpusetCpus constrains the allowed set of logical CPUs. Default: "" (not specified).
	CpusetCpus string
	// CpusetMems constrains the allowed set of memory nodes. Default: "" (not specified).
	CpusetMems string
}

// UpdateContainerResources sends an UpdateContainerResourcesRequest to the server, and parses
// the returned UpdateContainerResourcesResponse.
func UpdateContainerResources(client internalapi.RuntimeService, id string, opts updateOptions) error {
	if id == "" {
		return fmt.Errorf("ID cannot be empty")
	}
	request := &pb.UpdateContainerResourcesRequest{
		ContainerId: id,
	}
	if goruntime.GOOS != "windows" {
		request.Linux = &pb.LinuxContainerResources{
			CpuPeriod:          opts.CPUPeriod,
			CpuQuota:           opts.CPUQuota,
			CpuShares:          opts.CPUShares,
			CpusetCpus:         opts.CpusetCpus,
			CpusetMems:         opts.CpusetMems,
			MemoryLimitInBytes: opts.MemoryLimitInBytes,
			OomScoreAdj:        opts.OomScoreAdj,
		}
	} else {
		request.Windows = &pb.WindowsContainerResources{
			CpuCount:           opts.CPUCount,
			CpuMaximum:         opts.CPUMaximum,
			CpuShares:          opts.CPUShares,
			MemoryLimitInBytes: opts.MemoryLimitInBytes,
		}
	}
	logrus.Debugf("UpdateContainerResourcesRequest: %v", request)
	resources := &pb.ContainerResources{Linux: request.Linux}
	if err := client.UpdateContainerResources(context.TODO(), id, resources); err != nil {
		return err
	}
	fmt.Println(id)
	return nil
}

// CheckpointContainer sends a CheckpointContainerRequest to the server
func CheckpointContainer(
	rClient internalapi.RuntimeService,
	ID string,
	export string,
) error {
	if ID == "" {
		return errors.New("ID cannot be empty")
	}
	request := &pb.CheckpointContainerRequest{
		ContainerId: ID,
		Location:    export,
	}
	logrus.Debugf("CheckpointContainerRequest: %v", request)
	err := rClient.CheckpointContainer(context.TODO(), request)
	if err != nil {
		return err
	}
	fmt.Println(ID)
	return nil
}

// RemoveContainer sends a RemoveContainerRequest to the server, and parses
// the returned RemoveContainerResponse.
func RemoveContainer(client internalapi.RuntimeService, id string) error {
	if id == "" {
		return fmt.Errorf("ID cannot be empty")
	}
	if err := client.RemoveContainer(context.TODO(), id); err != nil {
		return err
	}
	fmt.Println(id)
	return nil
}

// marshalContainerStatus converts container status into string and converts
// the timestamps into readable format.
func marshalContainerStatus(cs *pb.ContainerStatus) (string, error) {
	statusStr, err := protobufObjectToJSON(cs)
	if err != nil {
		return "", err
	}
	jsonMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(statusStr), &jsonMap)
	if err != nil {
		return "", err
	}

	jsonMap["createdAt"] = time.Unix(0, cs.CreatedAt).Format(time.RFC3339Nano)
	var startedAt, finishedAt time.Time
	if cs.State != pb.ContainerState_CONTAINER_CREATED {
		// If container is not in the created state, we have tried and
		// started the container. Set the startedAt.
		startedAt = time.Unix(0, cs.StartedAt)
	}
	if cs.State == pb.ContainerState_CONTAINER_EXITED ||
		(cs.State == pb.ContainerState_CONTAINER_UNKNOWN && cs.FinishedAt > 0) {
		// If container is in the exit state, set the finishedAt.
		// Or if container is in the unknown state and FinishedAt > 0, set the finishedAt
		finishedAt = time.Unix(0, cs.FinishedAt)
	}
	jsonMap["startedAt"] = startedAt.Format(time.RFC3339Nano)
	jsonMap["finishedAt"] = finishedAt.Format(time.RFC3339Nano)
	return marshalMapInOrder(jsonMap, *cs)
}

// ContainerStatus sends a ContainerStatusRequest to the server, and parses
// the returned ContainerStatusResponse.
func ContainerStatus(client internalapi.RuntimeService, id, output string, tmplStr string, quiet bool) error {
	verbose := !(quiet)
	if output == "" { // default to json output
		output = "json"
	}
	if id == "" {
		return fmt.Errorf("ID cannot be empty")
	}
	request := &pb.ContainerStatusRequest{
		ContainerId: id,
		Verbose:     verbose,
	}
	logrus.Debugf("ContainerStatusRequest: %v", request)
	r, err := client.ContainerStatus(context.TODO(), id, verbose)
	logrus.Debugf("ContainerStatusResponse: %v", r)
	if err != nil {
		return err
	}

	status, err := marshalContainerStatus(r.Status)
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

	// output in table format
	fmt.Printf("ID: %s\n", r.Status.Id)
	if r.Status.Metadata != nil {
		if r.Status.Metadata.Name != "" {
			fmt.Printf("Name: %s\n", r.Status.Metadata.Name)
		}
		if r.Status.Metadata.Attempt != 0 {
			fmt.Printf("Attempt: %v\n", r.Status.Metadata.Attempt)
		}
	}
	fmt.Printf("State: %s\n", r.Status.State)
	ctm := time.Unix(0, r.Status.CreatedAt)
	fmt.Printf("Created: %v\n", units.HumanDuration(time.Now().UTC().Sub(ctm))+" ago")
	if r.Status.State != pb.ContainerState_CONTAINER_CREATED {
		stm := time.Unix(0, r.Status.StartedAt)
		fmt.Printf("Started: %v\n", units.HumanDuration(time.Now().UTC().Sub(stm))+" ago")
	}
	if r.Status.State == pb.ContainerState_CONTAINER_EXITED {
		if r.Status.FinishedAt > 0 {
			ftm := time.Unix(0, r.Status.FinishedAt)
			fmt.Printf("Finished: %v\n", units.HumanDuration(time.Now().UTC().Sub(ftm))+" ago")
		}
		fmt.Printf("Exit Code: %v\n", r.Status.ExitCode)
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

func getPodNameFromLabels(label map[string]string) string {
	podName, ok := label[types.KubernetesPodNameLabel]
	if ok {
		return podName
	}
	return "unknown"
}

func getContainersList(containersList []*pb.Container, opts listOptions) []*pb.Container {
	filtered := []*pb.Container{}
	for _, c := range containersList {
		// Filter by pod name/namespace regular expressions.
		if matchesRegex(opts.nameRegexp, c.Metadata.Name) {
			filtered = append(filtered, c)
		}
	}

	sort.Sort(containerByCreated(filtered))
	n := len(filtered)
	if opts.latest {
		n = 1
	}
	if opts.last > 0 {
		n = opts.last
	}
	n = func(a, b int) int {
		if a < b {
			return a
		}
		return b
	}(n, len(filtered))

	return filtered[:n]
}

// CreateContainer sends a CreateContainerRequest to the server, and parses
// the returned CreateContainerResponse.
func CreateContainer(
	iClient internalapi.ImageManagerService,
	rClient internalapi.RuntimeService,
	opts createOptions,
) (string, error) {

	config, err := loadContainerConfig(opts.configPath)
	if err != nil {
		return "", err
	}
	var podConfig *pb.PodSandboxConfig
	if opts.podConfig != "" {
		podConfig, err = loadPodSandboxConfig(opts.podConfig)
		if err != nil {
			return "", err
		}
	}

	// When there is a with-pull request or the image default mode is to
	// pull-image-on-create(true) and no-pull was not set we pull the image when
	// they ask for a create as a helper on the cli to reduce extra steps. As a
	// reminder if the image is already in cache only the manifest will be pulled
	// down to verify.
	if opts.withPull {
		auth, err := getAuth(opts.creds, opts.auth, opts.username)
		if err != nil {
			return "", err
		}

		// Try to pull the image before container creation
		image := config.GetImage().GetImage()
		ann := config.GetImage().GetAnnotations()
		if _, err := PullImageWithSandbox(iClient, image, auth, podConfig, ann); err != nil {
			return "", err
		}
	}

	request := &pb.CreateContainerRequest{
		PodSandboxId:  opts.podID,
		Config:        config,
		SandboxConfig: podConfig,
	}
	logrus.Debugf("CreateContainerRequest: %v", request)
	r, err := rClient.CreateContainer(context.TODO(), opts.podID, config, podConfig)
	logrus.Debugf("CreateContainerResponse: %v", r)
	if err != nil {
		return "", err
	}
	return r, nil
}

// StartContainer sends a StartContainerRequest to the server, and parses
// the returned StartContainerResponse.
func StartContainer(client internalapi.RuntimeService, id string) error {
	if id == "" {
		return fmt.Errorf("ID cannot be empty")
	}
	if err := client.StartContainer(context.TODO(), id); err != nil {
		return err
	}
	fmt.Println(id)
	return nil
}

var stopContainerCommand = &cli.Command{
	Name:                   "stop",
	Usage:                  "Stop one or more running containers",
	ArgsUsage:              "CONTAINER-ID [CONTAINER-ID...]",
	UseShortOptionHandling: true,
	Flags: []cli.Flag{
		&cli.Int64Flag{
			Name:    "timeout",
			Aliases: []string{"t"},
			Usage:   "Seconds to wait to kill the container after a graceful stop is requested",
		},
	},
	Action: func(c *cli.Context) error {
		if c.NArg() == 0 {
			return fmt.Errorf("ID cannot be empty")
		}
		runtimeClient, err := getRuntimeService(c, 0)
		if err != nil {
			return err
		}

		for i := 0; i < c.NArg(); i++ {
			containerID := c.Args().Get(i)
			if err := StopContainer(runtimeClient, containerID, c.Int64("timeout")); err != nil {
				return fmt.Errorf("stopping the container %q: %w", containerID, err)
			}
		}
		return nil
	},
}

// StopContainer sends a StopContainerRequest to the server, and parses
// the returned StopContainerResponse.
func StopContainer(client internalapi.RuntimeService, id string, timeout int64) error {
	if id == "" {
		return fmt.Errorf("ID cannot be empty")
	}
	if err := client.StopContainer(context.TODO(), id, timeout); err != nil {
		return err
	}
	fmt.Println(id)
	return nil
}
