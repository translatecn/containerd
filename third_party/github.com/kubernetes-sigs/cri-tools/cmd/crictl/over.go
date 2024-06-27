package main

import (
	"context"
	internalapi "demo/over/api/cri"
	pb "demo/over/api/cri/v1"
	"demo/over/cri/remote"
	"errors"
	"fmt"
	"github.com/docker/go-units"
	"github.com/opencontainers/go-digest"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"
	"log"
	"sort"
	"strings"
	"syscall"
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
	Usage: "显示运行时版本信息",
	Action: func(c *cli.Context) error {
		if c.NArg() != 0 {
			cli.ShowSubcommandHelp(c)
		}

		runtimeClient, err := getRuntimeService(c, 0)
		if err != nil {
			return err
		}
		if err := Version(runtimeClient, string(CRIVersionV1)); err != nil {
			return fmt.Errorf("getting the runtime version: %w", err)
		}
		return nil
	},
}

const (
	// CRIVersionV1 references the v1 CRI API.
	CRIVersionV1 CRIVersion = "v1"
)

type CRIVersion string

func getRuntimeService(context *cli.Context, timeout time.Duration) (res internalapi.RuntimeService, err error) {
	if RuntimeEndpointIsSet && RuntimeEndpoint == "" {
		return nil, fmt.Errorf("--runtime-endpoint is not set")
	}
	logrus.Debug("get runtime connection")
	// Check if a custom timeout is provided.
	t := time.Hour
	// If no EP set then use the default endpoint types
	return remote.NewRemoteRuntimeService(RuntimeEndpoint, t, nil)
}

var runtimeStatusCommand = &cli.Command{
	Name:                   "info",
	Usage:                  "显示容器运行时信息",
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

var listPodCommand = &cli.Command{
	Name:                   "pods",
	Usage:                  "显示所有 pods",
	UseShortOptionHandling: true,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "id",
			Value: "",
			Usage: "filter by pod id",
		},
		&cli.StringFlag{
			Name:  "name",
			Value: "",
			Usage: "filter by pod name regular expression pattern",
		},
		&cli.StringFlag{
			Name:  "namespace",
			Value: "",
			Usage: "filter by pod namespace regular expression pattern",
		},
		&cli.StringFlag{
			Name:    "state",
			Aliases: []string{"s"},
			Value:   "",
			Usage:   "filter by pod state",
		},
		&cli.StringSliceFlag{
			Name:  "label",
			Usage: "filter by key=value label",
		},
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"v"},
			Usage:   "show verbose info for pods",
		},
		&cli.BoolFlag{
			Name:    "quiet",
			Aliases: []string{"q"},
			Usage:   "list only pod IDs",
		},
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Output format, One of: json|yaml|table",
			Value:   "table",
		},
		&cli.BoolFlag{
			Name:    "latest",
			Aliases: []string{"l"},
			Usage:   "Show the most recently created pod",
		},
		&cli.IntFlag{
			Name:    "last",
			Aliases: []string{"n"},
			Usage:   "Show last n recently created pods. Set 0 for unlimited",
		},
		&cli.BoolFlag{
			Name:  "no-trunc",
			Usage: "Show output without truncating the ID",
		},
	},
	Action: func(c *cli.Context) error {
		var err error
		runtimeClient, err := getRuntimeService(c, 0)
		if err != nil {
			return err
		}

		opts := listOptions{
			id:                 c.String("id"),
			state:              c.String("state"),
			verbose:            c.Bool("verbose"),
			quiet:              c.Bool("quiet"),
			output:             c.String("output"),
			latest:             c.Bool("latest"),
			last:               c.Int("last"),
			noTrunc:            c.Bool("no-trunc"),
			nameRegexp:         c.String("name"),
			podNamespaceRegexp: c.String("namespace"),
		}
		opts.labels, err = parseLabelStringSlice(c.StringSlice("label"))
		if err != nil {
			return err
		}
		if err = ListPodSandboxes(runtimeClient, opts); err != nil {
			return fmt.Errorf("listing pod sandboxes: %w", err)
		}
		return nil
	},
}

// ListPodSandboxes sends a ListPodSandboxRequest to the server, and parses
// the returned ListPodSandboxResponse.
func ListPodSandboxes(client internalapi.RuntimeService, opts listOptions) error {
	filter := &pb.PodSandboxFilter{}
	if opts.id != "" {
		filter.Id = opts.id
	}
	if opts.state != "" {
		st := &pb.PodSandboxStateValue{}
		st.State = pb.PodSandboxState_SANDBOX_NOTREADY
		switch strings.ToLower(opts.state) {
		case "ready":
			st.State = pb.PodSandboxState_SANDBOX_READY
			filter.State = st
		case "notready":
			st.State = pb.PodSandboxState_SANDBOX_NOTREADY
			filter.State = st
		default:
			log.Fatalf("--state should be ready or notready")
		}
	}
	if opts.labels != nil {
		filter.LabelSelector = opts.labels
	}
	request := &pb.ListPodSandboxRequest{
		Filter: filter,
	}
	logrus.Debugf("ListPodSandboxRequest: %v", request)
	r, err := client.ListPodSandbox(context.TODO(), filter)
	logrus.Debugf("ListPodSandboxResponse: %v", r)
	if err != nil {
		return err
	}
	r = getSandboxesList(r, opts)

	switch opts.output {
	case "json":
		return outputProtobufObjAsJSON(&pb.ListPodSandboxResponse{Items: r})
	case "yaml":
		return outputProtobufObjAsYAML(&pb.ListPodSandboxResponse{Items: r})
	case "table":
	// continue; output will be generated after the switch block ends.
	default:
		return fmt.Errorf("unsupported output format %q", opts.output)
	}

	display := newTableDisplay(20, 1, 3, ' ', 0)
	if !opts.verbose && !opts.quiet {
		display.AddRow([]string{
			columnPodID,
			columnCreated,
			columnState,
			columnName,
			columnNamespace,
			columnAttempt,
			columnPodRuntime,
		})
	}
	for _, pod := range r {
		if opts.quiet {
			fmt.Printf("%s\n", pod.Id)
			continue
		}
		if !opts.verbose {
			createdAt := time.Unix(0, pod.CreatedAt)
			ctm := units.HumanDuration(time.Now().UTC().Sub(createdAt)) + " ago"
			id := pod.Id
			if !opts.noTrunc {
				id = getTruncatedID(id, "")
			}
			display.AddRow([]string{
				id,
				ctm,
				convertPodState(pod.State),
				pod.Metadata.Name,
				pod.Metadata.Namespace,
				fmt.Sprintf("%d", pod.Metadata.Attempt),
				getSandboxesRuntimeHandler(pod),
			})
			continue
		}

		fmt.Printf("ID: %s\n", pod.Id)
		if pod.Metadata != nil {
			if pod.Metadata.Name != "" {
				fmt.Printf("Name: %s\n", pod.Metadata.Name)
			}
			if pod.Metadata.Uid != "" {
				fmt.Printf("UID: %s\n", pod.Metadata.Uid)
			}
			if pod.Metadata.Namespace != "" {
				fmt.Printf("Namespace: %s\n", pod.Metadata.Namespace)
			}
			if pod.Metadata.Attempt != 0 {
				fmt.Printf("Attempt: %v\n", pod.Metadata.Attempt)
			}
		}
		fmt.Printf("Status: %s\n", convertPodState(pod.State))
		ctm := time.Unix(0, pod.CreatedAt)
		fmt.Printf("Created: %v\n", ctm)
		if pod.Labels != nil {
			for _, k := range getSortedKeys(pod.Labels) {
				fmt.Printf("\t%s -> %s\n", k, pod.Labels[k])
			}
		}
		if pod.Annotations != nil {
			for _, k := range getSortedKeys(pod.Annotations) {
				fmt.Printf("\t%s -> %s\n", k, pod.Annotations[k])
			}
		}
		fmt.Printf("%s: %s\n",
			strings.Title(strings.ToLower(columnPodRuntime)),
			getSandboxesRuntimeHandler(pod))

	}

	display.Flush()
	return nil
}
func getSandboxesList(sandboxesList []*pb.PodSandbox, opts listOptions) []*pb.PodSandbox {
	filtered := []*pb.PodSandbox{}
	for _, p := range sandboxesList {
		// Filter by pod name/namespace regular expressions.
		if matchesRegex(opts.nameRegexp, p.Metadata.Name) &&
			matchesRegex(opts.podNamespaceRegexp, p.Metadata.Namespace) {
			filtered = append(filtered, p)
		}
	}

	sort.Sort(sandboxByCreated(filtered))
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

var listImageCommand = &cli.Command{
	Name:                   "images",
	Aliases:                []string{"image", "img"},
	Usage:                  "List images",
	ArgsUsage:              "[REPOSITORY[:TAG]]",
	UseShortOptionHandling: true,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"v"},
			Usage:   "Show verbose info for images",
		},
		&cli.BoolFlag{
			Name:    "quiet",
			Aliases: []string{"q"},
			Usage:   "Only show image IDs",
		},
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Output format, One of: json|yaml|table",
		},
		&cli.BoolFlag{
			Name:  "digests",
			Usage: "Show digests",
		},
		&cli.BoolFlag{
			Name:  "no-trunc",
			Usage: "Show output without truncating the ID",
		},
	},
	Action: func(c *cli.Context) error {
		if c.NArg() > 1 {
			return cli.ShowSubcommandHelp(c)
		}

		imageClient, err := getImageService(c)
		if err != nil {
			return err
		}

		r, err := ListImages(imageClient, c.Args().First())
		if err != nil {
			return fmt.Errorf("listing images: %w", err)
		}
		sort.Sort(imageByRef(r.Images))

		switch c.String("output") {
		case "json":
			return outputProtobufObjAsJSON(r)
		case "yaml":
			return outputProtobufObjAsYAML(r)
		}

		// output in table format by default.
		display := newTableDisplay(20, 1, 3, ' ', 0)
		verbose := c.Bool("verbose")
		showDigest := c.Bool("digests")
		quiet := c.Bool("quiet")
		noTrunc := c.Bool("no-trunc")
		if !verbose && !quiet {
			if showDigest {
				display.AddRow([]string{columnImage, columnTag, columnDigest, columnImageID, columnSize})
			} else {
				display.AddRow([]string{columnImage, columnTag, columnImageID, columnSize})
			}
		}
		for _, image := range r.Images {
			if quiet {
				fmt.Printf("%s\n", image.Id)
				continue
			}
			if !verbose {
				imageName, repoDigest := normalizeRepoDigest(image.RepoDigests)
				repoTagPairs := normalizeRepoTagPair(image.RepoTags, imageName)
				size := units.HumanSizeWithPrecision(float64(image.GetSize_()), 3)
				id := image.Id
				if !noTrunc {
					id = getTruncatedID(id, "sha256:")
					repoDigest = getTruncatedID(repoDigest, "sha256:")
				}
				for _, repoTagPair := range repoTagPairs {
					if showDigest {
						display.AddRow([]string{repoTagPair[0], repoTagPair[1], repoDigest, id, size})
					} else {
						display.AddRow([]string{repoTagPair[0], repoTagPair[1], id, size})
					}
				}
				continue
			}
			fmt.Printf("ID: %s\n", image.Id)
			for _, tag := range image.RepoTags {
				fmt.Printf("RepoTags: %s\n", tag)
			}
			for _, digest := range image.RepoDigests {
				fmt.Printf("RepoDigests: %s\n", digest)
			}
			if image.Size_ != 0 {
				fmt.Printf("Size: %d\n", image.Size_)
			}
			if image.Uid != nil {
				fmt.Printf("Uid: %v\n", image.Uid)
			}
			if image.Username != "" {
				fmt.Printf("Username: %v\n", image.Username)
			}
			fmt.Printf("\n")
		}
		display.Flush()
		return nil
	},
}
var pullImageCommand = &cli.Command{
	Name:                   "pull",
	Usage:                  "Pull an image from a registry",
	UseShortOptionHandling: true,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "creds",
			Value:   "",
			Usage:   "Use `USERNAME[:PASSWORD]` for accessing the registry",
			EnvVars: []string{"CRICTL_CREDS"},
		},
		&cli.StringFlag{
			Name:    "auth",
			Value:   "",
			Usage:   "Use `AUTH_STRING` for accessing the registry. AUTH_STRING is a base64 encoded 'USERNAME[:PASSWORD]'",
			EnvVars: []string{"CRICTL_AUTH"},
		},
		&cli.StringFlag{
			Name:    "username",
			Aliases: []string{"u"},
			Value:   "",
			Usage:   "Use `USERNAME` for accessing the registry. The password will be requested on the command line",
		},
		&cli.StringFlag{
			Name:      "pod-config",
			Value:     "",
			Usage:     "Use `pod-config.[json|yaml]` to override the the pull c",
			TakesFile: true,
		},
		&cli.StringSliceFlag{
			Name:    "annotation",
			Aliases: []string{"a"},
			Usage:   "Annotation to be set on the pulled image",
		},
	},
	ArgsUsage: "NAME[:TAG|@DIGEST]",
	Action: func(c *cli.Context) error {
		imageName := c.Args().First()
		if imageName == "" {
			return fmt.Errorf("Image name cannot be empty")
		}

		if c.NArg() > 1 {
			return cli.ShowSubcommandHelp(c)
		}

		imageClient, err := getImageService(c)
		if err != nil {
			return err
		}

		auth, err := getAuth(c.String("creds"), c.String("auth"), c.String("username"))
		if err != nil {
			return err
		}
		var sandbox *pb.PodSandboxConfig
		if c.IsSet("pod-config") {
			sandbox, err = loadPodSandboxConfig(c.String("pod-config"))
			if err != nil {
				return fmt.Errorf("load podSandboxConfig: %w", err)
			}
		}
		var ann map[string]string
		if c.IsSet("annotation") {
			annotationFlags := c.StringSlice("annotation")
			ann, err = parseLabelStringSlice(annotationFlags)
			if err != nil {
				return err
			}
		}
		r, err := PullImageWithSandbox(imageClient, imageName, auth, sandbox, ann)
		if err != nil {
			return fmt.Errorf("pulling image: %w", err)
		}
		fmt.Printf("Image is up to date for %s\n", r.ImageRef)
		return nil
	},
}

func getAuth(creds string, auth string, username string) (*pb.AuthConfig, error) {
	if username != "" {
		fmt.Print("Enter Password:")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Print("\n")
		if err != nil {
			return nil, err
		}
		password := string(bytePassword)
		return &pb.AuthConfig{
			Username: username,
			Password: password,
		}, nil
	}
	if creds != "" && auth != "" {
		return nil, errors.New("both `--creds` and `--auth` are specified")
	}
	if creds != "" {
		username, password, err := parseCreds(creds)
		if err != nil {
			return nil, err
		}
		return &pb.AuthConfig{
			Username: username,
			Password: password,
		}, nil
	}
	if auth != "" {
		return &pb.AuthConfig{
			Auth: auth,
		}, nil
	}
	return nil, nil
}

var statsCommand = &cli.Command{
	Name:                   "stats",
	Usage:                  "列出容器资源使用统计信息",
	UseShortOptionHandling: true,
	ArgsUsage:              "[ID]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "all",
			Aliases: []string{"a"},
			Usage:   "Show all containers (default shows just running)",
		},
		&cli.StringFlag{
			Name:  "id",
			Value: "",
			Usage: "Filter by container id",
		},
		&cli.StringFlag{
			Name:    "pod",
			Aliases: []string{"p"},
			Value:   "",
			Usage:   "Filter by pod id",
		},
		&cli.StringSliceFlag{
			Name:  "label",
			Usage: "Filter by key=value label",
		},
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Output format, One of: json|yaml|table",
		},
		&cli.IntFlag{
			Name:    "seconds",
			Aliases: []string{"s"},
			Value:   1,
			Usage:   "Sample duration for CPU usage in seconds",
		},
		&cli.BoolFlag{
			Name:    "watch",
			Aliases: []string{"w"},
			Usage:   "Watch pod resources",
		},
	},
	Action: func(c *cli.Context) error {
		if c.NArg() > 1 {
			return cli.ShowSubcommandHelp(c)
		}

		runtimeClient, err := getRuntimeService(c, 0)
		if err != nil {
			return err
		}

		id := c.String("id")
		if id == "" && c.NArg() > 0 {
			id = c.Args().First()
		}

		opts := statsOptions{
			all:    c.Bool("all"),
			id:     id,
			podID:  c.String("pod"),
			sample: time.Duration(c.Int("seconds")) * time.Second,
			output: c.String("output"),
			watch:  c.Bool("watch"),
		}
		opts.labels, err = parseLabelStringSlice(c.StringSlice("label"))
		if err != nil {
			return err
		}

		if err = ContainerStats(runtimeClient, opts); err != nil {
			return fmt.Errorf("get container stats: %w", err)
		}
		return nil
	},
}

func ContainerStats(client internalapi.RuntimeService, opts statsOptions) error {
	filter := &pb.ContainerStatsFilter{}
	if opts.id != "" {
		filter.Id = opts.id
	}
	if opts.podID != "" {
		filter.PodSandboxId = opts.podID
	}
	if opts.labels != nil {
		filter.LabelSelector = opts.labels
	}
	request := &pb.ListContainerStatsRequest{
		Filter: filter,
	}

	display := newTableDisplay(20, 1, 3, ' ', 0)
	if !opts.watch {
		if err := displayStats(context.TODO(), client, request, display, opts); err != nil {
			return err
		}
	} else {
		displayErrCh := make(chan error, 1)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		watchCtx, cancelFn := context.WithCancel(context.Background())
		defer cancelFn()
		// Put the displayStats in another goroutine.
		// because it might be time consuming with lots of containers.
		// and we want to cancel it ASAP when user hit CtrlC
		go func() {
			for range ticker.C {
				if err := displayStats(watchCtx, client, request, display, opts); err != nil {
					displayErrCh <- err
					break
				}
			}
		}()
		// listen for CtrlC or error
		select {
		case <-SetupInterruptSignalHandler():
			cancelFn()
			return nil
		case err := <-displayErrCh:
			return err
		}
	}

	return nil
}

var runPodCommand = &cli.Command{
	Name:      "runp",
	Usage:     "运行 Pod 沙盒",
	ArgsUsage: "pod-config.[json|yaml]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "runtime",
			Aliases: []string{"r"},
			Usage:   "Runtime handler to use. Available options are defined by the container runtime.",
		},
		&cli.DurationFlag{
			Name:    "cancel-timeout",
			Aliases: []string{"T"},
			Value:   time.Hour,
			Usage:   "Seconds to wait for a run pod sandbox request to complete before cancelling the request",
		},
	},

	Action: func(c *cli.Context) error {
		sandboxSpec := c.Args().First()
		if c.NArg() != 1 || sandboxSpec == "" {
			return cli.ShowSubcommandHelp(c)
		}

		runtimeClient, err := getRuntimeService(c, c.Duration("cancel-timeout"))
		if err != nil {
			return err
		}

		podSandboxConfig, err := loadPodSandboxConfig(sandboxSpec)
		if err != nil {
			return fmt.Errorf("load podSandboxConfig: %w", err)
		}

		// Test RuntimeServiceClient.RunPodSandbox
		podID, err := RunPodSandbox(runtimeClient, podSandboxConfig, c.String("runtime"))
		if err != nil {
			return fmt.Errorf("run pod sandbox: %w", err)
		}
		fmt.Println(podID)
		return nil
	},
}

func RunPodSandbox(client internalapi.RuntimeService, config *pb.PodSandboxConfig, runtime string) (string, error) {
	request := &pb.RunPodSandboxRequest{
		Config:         config,
		RuntimeHandler: runtime,
	}
	logrus.Debugf("RunPodSandboxRequest: %v", request)
	r, err := client.RunPodSandbox(context.TODO(), config, runtime)
	logrus.Debugf("RunPodSandboxResponse: %v", r)
	if err != nil {
		return "", err
	}
	return r, nil
}

var createContainerCommand = &cli.Command{
	Name:      "create",
	Usage:     "创建一个容器",
	ArgsUsage: "POD container-config.[json|yaml] pod-config.[json|yaml]",
	Flags: append(createPullFlags, &cli.DurationFlag{
		Name:  "cancel-timeout",
		Value: time.Hour,
		Usage: "Seconds to wait for a container create request to complete before cancelling the request",
	}),

	Action: func(c *cli.Context) (err error) {
		if c.Args().Len() != 3 {
			return cli.ShowSubcommandHelp(c)
		}
		if c.Bool("no-pull") == true && c.Bool("with-pull") == true {
			return errors.New("confict: no-pull and with-pull are both set")
		}

		withPull := (!c.Bool("no-pull") && PullImageOnCreate) || c.Bool("with-pull")

		var imageClient internalapi.ImageManagerService
		if withPull {
			imageClient, err = getImageService(c)
			if err != nil {
				return err
			}
		}

		opts := createOptions{
			podID: c.Args().Get(0),
			runOptions: &runOptions{
				configPath: c.Args().Get(1),
				podConfig:  c.Args().Get(2),
				pullOptions: &pullOptions{
					withPull: withPull,
					creds:    c.String("creds"),
					auth:     c.String("auth"),
					username: c.String("username"),
				},
				timeout: c.Duration("cancel-timeout"),
			},
		}

		runtimeClient, err := getRuntimeService(c, opts.timeout)
		if err != nil {
			return err
		}

		ctrID, err := CreateContainer(imageClient, runtimeClient, opts)
		if err != nil {
			return fmt.Errorf("creating container: %w", err)
		}
		fmt.Println(ctrID)
		return nil
	},
}
var startContainerCommand = &cli.Command{
	Name:      "start",
	Usage:     "Start one or more created containers",
	ArgsUsage: "CONTAINER-ID [CONTAINER-ID...]",
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
			if err := StartContainer(runtimeClient, containerID); err != nil {
				return fmt.Errorf("starting the container %q: %w", containerID, err)
			}
		}
		return nil
	},
}

func getImageService(context *cli.Context) (res internalapi.ImageManagerService, err error) {
	if ImageEndpoint == "" {
		if RuntimeEndpointIsSet && RuntimeEndpoint == "" {
			return nil, fmt.Errorf("--image-endpoint is not set")
		}
		ImageEndpoint = RuntimeEndpoint
		ImageEndpointIsSet = RuntimeEndpointIsSet
	}

	logrus.Debugf("get image connection")
	// If no EP set then use the default endpoint types
	if !ImageEndpointIsSet {
		//logrus.Warningf("image connect using default endpoints: %v. "+
		//	"As the default settings are now deprecated, you should set the "+
		//	"endpoint instead.", defaultRuntimeEndpoints)
		logrus.Debug("Note that performance maybe affected as each default " +
			"connection attempt takes n-seconds to complete before timing out " +
			"and going to the next in sequence.")

		return res, err
	}
	return remote.NewRemoteImageService(ImageEndpoint, Timeout, nil)
}

var listContainersCommand = &cli.Command{
	Name:                   "ps",
	Usage:                  "显示所有容器",
	UseShortOptionHandling: true,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"v"},
			Usage:   "Show verbose information for containers",
		},
		&cli.StringFlag{
			Name:  "id",
			Value: "",
			Usage: "Filter by container id",
		},
		&cli.StringFlag{
			Name:  "name",
			Value: "",
			Usage: "filter by container name regular expression pattern",
		},
		&cli.StringFlag{
			Name:    "pod",
			Aliases: []string{"p"},
			Value:   "",
			Usage:   "Filter by pod id",
		},
		&cli.StringFlag{
			Name:  "image",
			Value: "",
			Usage: "Filter by container image",
		},
		&cli.StringFlag{
			Name:    "state",
			Aliases: []string{"s"},
			Value:   "",
			Usage:   "Filter by container state",
		},
		&cli.StringSliceFlag{
			Name:  "label",
			Usage: "Filter by key=value label",
		},
		&cli.BoolFlag{
			Name:    "quiet",
			Aliases: []string{"q"},
			Usage:   "Only display container IDs",
		},
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Output format, One of: json|yaml|table",
			Value:   "table",
		},
		&cli.BoolFlag{
			Name:    "all",
			Aliases: []string{"a"},
			Usage:   "Show all containers",
		},
		&cli.BoolFlag{
			Name:    "latest",
			Aliases: []string{"l"},
			Usage:   "Show the most recently created container (includes all states)",
		},
		&cli.IntFlag{
			Name:    "last",
			Aliases: []string{"n"},
			Usage:   "Show last n recently created containers (includes all states). Set 0 for unlimited.",
		},
		&cli.BoolFlag{
			Name:  "no-trunc",
			Usage: "Show output without truncating the ID",
		},
		&cli.BoolFlag{
			Name:    "resolve-image-path",
			Aliases: []string{"r"},
			Usage:   "Show image path instead of image id",
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

		imageClient, err := getImageService(c)
		if err != nil {
			return err
		}

		opts := listOptions{
			id:               c.String("id"),
			podID:            c.String("pod"),
			state:            c.String("state"),
			verbose:          c.Bool("verbose"),
			quiet:            c.Bool("quiet"),
			output:           c.String("output"),
			all:              c.Bool("all"),
			nameRegexp:       c.String("name"),
			latest:           c.Bool("latest"),
			last:             c.Int("last"),
			noTrunc:          c.Bool("no-trunc"),
			image:            c.String("image"),
			resolveImagePath: c.Bool("resolve-image-path"),
		}
		opts.labels, err = parseLabelStringSlice(c.StringSlice("label"))
		if err != nil {
			return err
		}

		if err = ListContainers(runtimeClient, imageClient, opts); err != nil {
			return fmt.Errorf("listing containers: %w", err)
		}
		return nil
	},
}

// ListContainers sends a ListContainerRequest to the server, and parses
// the returned ListContainerResponse.
func ListContainers(runtimeClient internalapi.RuntimeService, imageClient internalapi.ImageManagerService, opts listOptions) error {
	filter := &pb.ContainerFilter{}
	if opts.id != "" {
		filter.Id = opts.id
	}
	if opts.podID != "" {
		filter.PodSandboxId = opts.podID
	}
	st := &pb.ContainerStateValue{}
	if !opts.all && opts.state == "" {
		st.State = pb.ContainerState_CONTAINER_RUNNING
		filter.State = st
	}
	if opts.state != "" {
		st.State = pb.ContainerState_CONTAINER_UNKNOWN
		switch strings.ToLower(opts.state) {
		case "created":
			st.State = pb.ContainerState_CONTAINER_CREATED
			filter.State = st
		case "running":
			st.State = pb.ContainerState_CONTAINER_RUNNING
			filter.State = st
		case "exited":
			st.State = pb.ContainerState_CONTAINER_EXITED
			filter.State = st
		case "unknown":
			st.State = pb.ContainerState_CONTAINER_UNKNOWN
			filter.State = st
		default:
			log.Fatalf("--state should be one of created, running, exited or unknown")
		}
	}
	if opts.latest || opts.last > 0 {
		// Do not filter by state if latest/last is specified.
		filter.State = nil
	}
	if opts.labels != nil {
		filter.LabelSelector = opts.labels
	}
	r, err := runtimeClient.ListContainers(context.TODO(), filter)
	logrus.Debugf("ListContainerResponse: %v", r)
	if err != nil {
		return err
	}
	r = getContainersList(r, opts)

	switch opts.output {
	case "json":
		return outputProtobufObjAsJSON(&pb.ListContainersResponse{Containers: r})
	case "yaml":
		return outputProtobufObjAsYAML(&pb.ListContainersResponse{Containers: r})
	case "table":
	// continue; output will be generated after the switch block ends.
	default:
		return fmt.Errorf("unsupported output format %q", opts.output)
	}

	display := newTableDisplay(20, 1, 3, ' ', 0)
	if !opts.verbose && !opts.quiet {
		display.AddRow([]string{columnContainer, columnImage, columnCreated, columnState, columnName, columnAttempt, columnPodID, columnPodname})
	}
	for _, c := range r {
		if match, err := matchesImage(imageClient, opts.image, c.GetImage().GetImage()); err != nil {
			return fmt.Errorf("check image match: %w", err)
		} else if !match {
			continue
		}
		if opts.quiet {
			fmt.Printf("%s\n", c.Id)
			continue
		}

		createdAt := time.Unix(0, c.CreatedAt)
		ctm := units.HumanDuration(time.Now().UTC().Sub(createdAt)) + " ago"
		if !opts.verbose {
			id := c.Id
			image := c.Image.Image
			podID := c.PodSandboxId
			if !opts.noTrunc {
				id = getTruncatedID(id, "")
				podID = getTruncatedID(podID, "")
				// Now c.Image.Image is imageID in kubelet.
				if digest, err := digest.Parse(image); err == nil {
					image = getTruncatedID(digest.String(), string(digest.Algorithm())+":")
				}
			}
			if opts.resolveImagePath {
				orig, err := getRepoImage(imageClient, image)
				if err != nil {
					return fmt.Errorf("failed to fetch repo image %v", err)
				}
				image = orig
			}
			podName := getPodNameFromLabels(c.Labels)
			display.AddRow([]string{id, image, ctm, convertContainerState(c.State), c.Metadata.Name,
				fmt.Sprintf("%d", c.Metadata.Attempt), podID, podName})
			continue
		}

		fmt.Printf("ID: %s\n", c.Id)
		fmt.Printf("PodID: %s\n", c.PodSandboxId)
		if c.Metadata != nil {
			if c.Metadata.Name != "" {
				fmt.Printf("Name: %s\n", c.Metadata.Name)
			}
			fmt.Printf("Attempt: %v\n", c.Metadata.Attempt)
		}
		fmt.Printf("State: %s\n", convertContainerState(c.State))
		if c.Image != nil {
			fmt.Printf("Image: %s\n", c.Image.Image)
		}
		fmt.Printf("Created: %v\n", ctm)
		if c.Labels != nil {
			for _, k := range getSortedKeys(c.Labels) {
				fmt.Printf("\t%s -> %s\n", k, c.Labels[k])
			}
		}
		if c.Annotations != nil {
			for _, k := range getSortedKeys(c.Annotations) {
				fmt.Printf("\t%s -> %s\n", k, c.Annotations[k])
			}
		}
		fmt.Println()
	}

	display.Flush()
	return nil
}

func convertContainerState(state pb.ContainerState) string {
	switch state {
	case pb.ContainerState_CONTAINER_CREATED:
		return "Created"
	case pb.ContainerState_CONTAINER_RUNNING:
		return "Running"
	case pb.ContainerState_CONTAINER_EXITED:
		return "Exited"
	case pb.ContainerState_CONTAINER_UNKNOWN:
		return "Unknown"
	default:
		log.Fatalf("Unknown container state %q", state)
		return ""
	}
}
