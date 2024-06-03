/*
   Copyright The containerd Authors.

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

package run

import (
	gocontext "context"
	"errors"
	"strings"

	"demo/cmd/ctr/commands"
	"demo/containerd"
	"demo/others/console"
	"demo/over/oci"
	"demo/pkg/netns"
	"demo/snapshots"
	"demo/third_party/github.com/Microsoft/hcsshim/cmd/containerd-shim-runhcs-v1/options"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var platformRunFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "isolated",
		Usage: "Run the container with vm isolation",
	},
}

// NewContainer creates a new container
func NewContainer(ctx gocontext.Context, client *containerd.Client, context *cli.Context) (containerd.Container, error) {
	var (
		id    string
		opts  []over_oci.SpecOpts
		cOpts []containerd.NewContainerOpts
		spec  containerd.NewContainerOpts

		config = context.IsSet("config")
	)

	if sandbox := context.String("sandbox"); sandbox != "" {
		cOpts = append(cOpts, containerd.WithSandbox(sandbox))
	}

	if config {
		id = context.Args().First()
		opts = append(opts, over_oci.WithSpecFromFile(context.String("config")))
		cOpts = append(cOpts, containerd.WithContainerLabels(commands.LabelArgs(context.StringSlice("label"))))
	} else {
		var (
			ref  = context.Args().First()
			args = context.Args()[2:]
		)

		id = context.Args().Get(1)
		snapshotter := context.String("snapshotter")
		if snapshotter == "windows-lcow" {
			opts = append(opts, over_oci.WithDefaultSpecForPlatform("linux/amd64"))
			// Clear the rootfs section.
			opts = append(opts, over_oci.WithRootFSPath(""))
		} else {
			opts = append(opts, over_oci.WithDefaultSpec())
			opts = append(opts, over_oci.WithWindowNetworksAllowUnqualifiedDNSQuery())
			opts = append(opts, over_oci.WithWindowsIgnoreFlushesDuringBoot())
		}
		if ef := context.String("env-file"); ef != "" {
			opts = append(opts, over_oci.WithEnvFile(ef))
		}
		opts = append(opts, over_oci.WithEnv(context.StringSlice("env")))
		opts = append(opts, withMounts(context))

		image, err := client.GetImage(ctx, ref)
		if err != nil {
			return nil, err
		}
		unpacked, err := image.IsUnpacked(ctx, snapshotter)
		if err != nil {
			return nil, err
		}
		if !unpacked {
			if err := image.Unpack(ctx, snapshotter); err != nil {
				return nil, err
			}
		}
		opts = append(opts, over_oci.WithImageConfig(image))
		labels := buildLabels(commands.LabelArgs(context.StringSlice("label")), image.Labels())
		cOpts = append(cOpts,
			containerd.WithImage(image),
			containerd.WithImageConfigLabels(image),
			containerd.WithSnapshotter(snapshotter),
			containerd.WithNewSnapshot(
				id,
				image,
				snapshots.WithLabels(commands.LabelArgs(context.StringSlice("snapshotter-label")))),
			containerd.WithAdditionalContainerLabels(labels))

		if len(args) > 0 {
			opts = append(opts, over_oci.WithProcessArgs(args...))
		}
		if cwd := context.String("cwd"); cwd != "" {
			opts = append(opts, over_oci.WithProcessCwd(cwd))
		}
		if user := context.String("user"); user != "" {
			opts = append(opts, over_oci.WithUser(user))
		}
		if context.Bool("tty") {
			opts = append(opts, over_oci.WithTTY)

			con := console.Current()
			size, err := con.Size()
			if err != nil {
				logrus.WithError(err).Error("console size")
			}
			opts = append(opts, over_oci.WithTTYSize(int(size.Width), int(size.Height)))
		}
		if context.Bool("net-host") {
			return nil, errors.New("Cannot use host mode networking with Windows containers")
		}
		if context.Bool("cni") {
			ns, err := netns.NewNetNS("")
			if err != nil {
				return nil, err
			}
			opts = append(opts, over_oci.WithWindowsNetworkNamespace(ns.GetPath()))
			cniMeta := &commands.NetworkMetaData{EnableCni: true}
			cOpts = append(cOpts, containerd.WithContainerExtension(commands.CtrCniMetadataExtension, cniMeta))
		}
		if context.Bool("isolated") {
			opts = append(opts, over_oci.WithWindowsHyperV)
		}
		limit := context.Uint64("memory-limit")
		if limit != 0 {
			opts = append(opts, over_oci.WithMemoryLimit(limit))
		}
		ccount := context.Uint64("cpu-count")
		if ccount != 0 {
			opts = append(opts, over_oci.WithWindowsCPUCount(ccount))
		}
		cshares := context.Uint64("cpu-shares")
		if cshares != 0 {
			opts = append(opts, over_oci.WithWindowsCPUShares(uint16(cshares)))
		}
		cmax := context.Uint64("cpu-max")
		if cmax != 0 {
			opts = append(opts, over_oci.WithWindowsCPUMaximum(uint16(cmax)))
		}
		for _, dev := range context.StringSlice("device") {
			idType, devID, ok := strings.Cut(dev, "://")
			if !ok {
				return nil, errors.New("devices must be in the format IDType://ID")
			}
			if idType == "" {
				return nil, errors.New("devices must have a non-empty IDType")
			}
			opts = append(opts, over_oci.WithWindowsDevice(idType, devID))
		}
	}

	runtime := context.String("runtime")
	var runtimeOpts interface{}
	if runtime == "io.containerd.runhcs.v1" {
		runtimeOpts = &options.Options{
			Debug: context.GlobalBool("debug"),
		}
	}
	cOpts = append(cOpts, containerd.WithRuntime(runtime, runtimeOpts))

	var s specs.Spec
	spec = containerd.WithSpec(&s, opts...)

	cOpts = append(cOpts, spec)

	return client.NewContainer(ctx, id, cOpts...)
}

func getNetNSPath(ctx gocontext.Context, t containerd.Task) (string, error) {
	s, err := t.Spec(ctx)
	if err != nil {
		return "", err
	}
	if s.Windows == nil || s.Windows.Network == nil {
		return "", nil
	}
	return s.Windows.Network.NetworkNamespace, nil
}
