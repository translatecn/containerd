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
	"fmt"
	"github.com/kubernetes-sigs/cri-tools/pkg/remote"
	"os"
	"runtime"
	"sort"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	internalapi "demo/over/api/cri"

	"github.com/kubernetes-sigs/cri-tools/pkg/version"
)

const defaultTimeout = 2 * time.Second
const defaultTimeoutWindows = 200 * time.Second

var (
	// RuntimeEndpoint is CRI server runtime endpoint
	RuntimeEndpoint string
	// RuntimeEndpointIsSet is true when RuntimeEndpoint is configured
	RuntimeEndpointIsSet bool
	// ImageEndpoint is CRI server image endpoint, default same as runtime endpoint
	ImageEndpoint string
	// ImageEndpointIsSet is true when ImageEndpoint is configured
	ImageEndpointIsSet bool
	// Timeout  of connecting to server (default: 2s on Linux, 200s on Windows)
	Timeout time.Duration
	// Debug enable debug output
	Debug bool
	// PullImageOnCreate enables pulling image on create requests
	PullImageOnCreate bool
	// DisablePullOnRun disable pulling image on run requests
	DisablePullOnRun bool
)

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

		//for _, endPoint := range defaultRuntimeEndpoints {
		//	logrus.Debugf("Connect using endpoint %q with %q timeout", endPoint, Timeout)
		//
		//	res, err = remote.NewRemoteImageService(endPoint, Timeout, nil)
		//	if err != nil {
		//		logrus.Error(err)
		//		continue
		//	}
		//
		//	logrus.Debugf("Connected successfully using endpoint: %s", endPoint)
		//	break
		//}
		return res, err
	}
	return remote.NewRemoteImageService(ImageEndpoint, Timeout, nil)
}

func getTimeout(timeDuration time.Duration) time.Duration {
	if timeDuration.Seconds() > 0 {
		return timeDuration
	}

	if runtime.GOOS == "windows" {
		return defaultTimeoutWindows
	}

	return defaultTimeout // use default
}

func main() {
	app := cli.NewApp()
	app.Name = "crictl"
	app.Usage = "client for CRI"
	app.Version = version.Version

	app.Commands = []*cli.Command{
		runtimeAttachCommand,
		createContainerCommand,
		runtimeExecCommand,
		listImageCommand,
		containerStatusCommand,
		imageStatusCommand,
		imageFsInfoCommand,
		podStatusCommand,
		logsCommand,
		runtimePortForwardCommand,
		listContainersCommand,
		pullImageCommand,
		runContainerCommand,
		runPodCommand,
		removeContainerCommand,
		removeImageCommand,
		removePodCommand,
		listPodCommand,
		startContainerCommand,
		runtimeStatusCommand,      // ✅
		overRuntimeVersionCommand, // ✅
		stopContainerCommand,
		stopPodCommand,
		updateContainerCommand,
		configCommand,
		statsCommand,
		podStatsCommand,
		completionCommand,
		checkpointContainerCommand,
	}

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			EnvVars: []string{"CRI_CONFIG_FILE"},
			Value:   defaultConfigPath,
			Usage:   "Location of the client config file. If not specified and the default does not exist, the program's directory is searched as well",
		},
		&cli.StringFlag{
			Name:  "runtime-endpoint",
			Value: "vm:6789",
		},
		&cli.StringFlag{
			Name:  "image-endpoint",
			Value: "vm:6789",
		},
		&cli.DurationFlag{
			Name:  "timeout",
			Value: defaultTimeout,
			Usage: "Timeout of connecting to the server in seconds (e.g. 2s, 20s.). 0 or less is set to default",
		},
		&cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable debug mode",
		},
	}

	app.Before = func(context *cli.Context) (err error) {

		RuntimeEndpoint = context.String("runtime-endpoint")
		RuntimeEndpointIsSet = true
		ImageEndpoint = context.String("image-endpoint")
		ImageEndpointIsSet = true
		if context.IsSet("timeout") {
			Timeout = getTimeout(context.Duration("timeout"))
		} else {
			Timeout = context.Duration("timeout")
		}
		Debug = context.Bool("debug")
		DisablePullOnRun = false

		if Debug {
			logrus.SetLevel(logrus.DebugLevel)
		}
		return nil
	}
	// sort all flags
	for _, cmd := range app.Commands {
		sort.Sort(cli.FlagsByName(cmd.Flags))
	}
	sort.Sort(cli.FlagsByName(app.Flags))

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}
