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

package cri

import (
	"demo/others/log"
	over_plugin2 "demo/over/plugin"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
	"k8s.io/klog/v2"

	"demo/containerd"
	"demo/over/platforms"
	criconfig "demo/pkg/cri/config"
	"demo/pkg/cri/constants"
	"demo/pkg/cri/nri"
	"demo/pkg/cri/sbserver"
	"demo/pkg/cri/server"
	nriservice "demo/pkg/nri"
	"demo/services/warning"
)

// Register CRI service plugin
func init() {
	config := criconfig.DefaultConfig()
	over_plugin2.Register(&over_plugin2.Registration{
		Type:   over_plugin2.GRPCPlugin,
		ID:     "cri",
		Config: &config,
		Requires: []over_plugin2.Type{
			over_plugin2.EventPlugin,
			over_plugin2.ServicePlugin,
			over_plugin2.NRIApiPlugin,
			over_plugin2.WarningPlugin,
			over_plugin2.SnapshotPlugin,
		},
		InitFn: initCRIService,
	})
}

func initCRIService(ic *over_plugin2.InitContext) (interface{}, error) {
	ic.Meta.Platforms = []imagespec.Platform{over_platforms.DefaultSpec()}
	ic.Meta.Exports = map[string]string{"CRIVersion": constants.CRIVersion, "CRIVersionAlpha": constants.CRIVersionAlpha}
	ctx := ic.Context
	pluginConfig := ic.Config.(*criconfig.PluginConfig)
	ws, err := ic.Get(over_plugin2.WarningPlugin)
	if err != nil {
		return nil, err
	}
	warn := ws.(warning.Service)

	if warnings, err := criconfig.ValidatePluginConfig(ctx, pluginConfig); err != nil {
		return nil, fmt.Errorf("invalid plugin config: %w", err)
	} else if len(warnings) > 0 {
		for _, w := range warnings {
			warn.Emit(ctx, w)
		}
	}

	c := criconfig.Config{
		PluginConfig:       *pluginConfig,
		ContainerdRootDir:  filepath.Dir(ic.Root),
		ContainerdEndpoint: ic.Address,
		RootDir:            ic.Root,
		StateDir:           ic.State,
	}
	log.G(ctx).Infof("Start cri plugin with config %+v", c)

	if err := setGLogLevel(); err != nil {
		return nil, fmt.Errorf("failed to set glog level: %w", err)
	}

	log.G(ctx).Info("Connect containerd service")
	client, err := containerd.New(
		"",
		containerd.WithDefaultNamespace(constants.K8sContainerdNamespace),
		containerd.WithDefaultPlatform(over_platforms.Default()),
		containerd.WithInMemoryServices(ic),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create containerd client: %w", err)
	}

	var s server.CRIService
	if os.Getenv("ENABLE_CRI_SANDBOXES") != "" {
		log.G(ctx).Info("using experimental CRI Sandbox server - unset ENABLE_CRI_SANDBOXES to disable")
		s, err = sbserver.NewCRIService(c, client, getNRIAPI(ic), warn)
	} else {
		log.G(ctx).Info("using legacy CRI server")
		s, err = server.NewCRIService(c, client, getNRIAPI(ic), warn)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create CRI service: %w", err)
	}

	// RegisterReadiness() must be called after NewCRIService(): https://github.com/containerd/issues/9163
	ready := ic.RegisterReadiness()
	go func() {
		if err := s.Run(ready); err != nil {
			log.G(ctx).WithError(err).Fatal("Failed to run CRI service")
		}
		// TODO(random-liu): Whether and how we can stop containerd.
	}()

	return s, nil
}

// Set glog level.
func setGLogLevel() error {
	l := log.GetLevel()
	fs := flag.NewFlagSet("klog", flag.PanicOnError)
	klog.InitFlags(fs)
	if err := fs.Set("logtostderr", "true"); err != nil {
		return err
	}
	switch l {
	case log.TraceLevel:
		return fs.Set("v", "5")
	case log.DebugLevel:
		return fs.Set("v", "4")
	case log.InfoLevel:
		return fs.Set("v", "2")
	default:
		// glog doesn't support other filters. Defaults to v=0.
	}
	return nil
}

// Get the NRI plugin, and set up our NRI API for it.
func getNRIAPI(ic *over_plugin2.InitContext) *nri.API {
	const (
		pluginType = over_plugin2.NRIApiPlugin
		pluginName = "nri"
	)

	ctx := ic.Context

	p, err := ic.GetByID(pluginType, pluginName)
	if err != nil {
		log.G(ctx).Info("NRI service not found, NRI support disabled")
		return nil
	}

	api, ok := p.(nriservice.API)
	if !ok {
		log.G(ctx).Infof("NRI plugin (%s, %q) has incorrect type %T, NRI support disabled",
			pluginType, pluginName, api)
		return nil
	}

	log.G(ctx).Info("using experimental NRI integration - disable nri plugin to prevent this")

	return nri.NewAPI(api)
}
