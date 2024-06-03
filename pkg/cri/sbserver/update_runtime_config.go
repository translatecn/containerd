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

package sbserver

import (
	"context"
	"demo/others/log"
	"demo/over/my_mk"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	runtime "k8s.io/cri-api/pkg/apis/runtime/v1"

	"demo/pkg/atomicfile"
)

// cniConfigTemplate contains the values containerd will overwrite
// in the cni config template.
type cniConfigTemplate struct {
	// PodCIDR is the cidr for pods on the node.
	PodCIDR string
	// PodCIDRRanges is the cidr ranges for pods on the node.
	PodCIDRRanges []string
	// Routes is a list of routes configured.
	Routes []string
}

const (
	// cniConfigFileName is the name of cni config file generated by containerd.
	cniConfigFileName = "10-containerd-net.conflist"
	// zeroCIDRv6 is the null route for IPv6.
	zeroCIDRv6 = "::/0"
	// zeroCIDRv4 is the null route for IPv4.
	zeroCIDRv4 = "0.0.0.0/0"
)

// UpdateRuntimeConfig updates the runtime config. Currently only handles podCIDR updates.
func (c *criService) UpdateRuntimeConfig(ctx context.Context, r *runtime.UpdateRuntimeConfigRequest) (*runtime.UpdateRuntimeConfigResponse, error) {
	podCIDRs := r.GetRuntimeConfig().GetNetworkConfig().GetPodCidr()
	if podCIDRs == "" {
		return &runtime.UpdateRuntimeConfigResponse{}, nil
	}
	cidrs := strings.Split(podCIDRs, ",")
	for i := range cidrs {
		cidrs[i] = strings.TrimSpace(cidrs[i])
	}
	routes, err := getRoutes(cidrs)
	if err != nil {
		return nil, fmt.Errorf("get routes: %w", err)
	}

	confTemplate := c.config.NetworkPluginConfTemplate
	if confTemplate == "" {
		log.G(ctx).Info("No cni config template is specified, wait for other system components to drop the config.")
		return &runtime.UpdateRuntimeConfigResponse{}, nil
	}
	netPlugin := c.netPlugin[defaultNetworkPlugin]
	if netPlugin == nil {
		log.G(ctx).Infof("Network plugin is ready, skip generating cni config from template %q", confTemplate)
		return &runtime.UpdateRuntimeConfigResponse{}, nil
	}
	netStart := time.Now()
	err = netPlugin.Status()
	networkPluginOperations.WithValues(networkStatusOp).Inc()
	networkPluginOperationsLatency.WithValues(networkStatusOp).UpdateSince(netStart)
	if err == nil {
		log.G(ctx).Infof("Network plugin is ready, skip generating cni config from template %q", confTemplate)
		return &runtime.UpdateRuntimeConfigResponse{}, nil
	}
	networkPluginOperationsErrors.WithValues(networkStatusOp).Inc()
	if err := netPlugin.Load(c.cniLoadOptions()...); err == nil {
		log.G(ctx).Infof("CNI config is successfully loaded, skip generating cni config from template %q", confTemplate)
		return &runtime.UpdateRuntimeConfigResponse{}, nil
	}
	if err := writeCNIConfigFile(ctx, c.config.NetworkPluginConfDir, confTemplate, cidrs[0], cidrs, routes); err != nil {
		return nil, err
	}
	return &runtime.UpdateRuntimeConfigResponse{}, nil
}

// getRoutes generates required routes for the passed in cidrs.
func getRoutes(cidrs []string) ([]string, error) {
	var (
		routes       []string
		hasV4, hasV6 bool
	)
	for _, c := range cidrs {
		_, cidr, err := net.ParseCIDR(c)
		if err != nil {
			return nil, err
		}
		if cidr.IP.To4() != nil {
			hasV4 = true
		} else {
			hasV6 = true
		}
	}
	if hasV4 {
		routes = append(routes, zeroCIDRv4)
	}
	if hasV6 {
		routes = append(routes, zeroCIDRv6)
	}
	return routes, nil
}

func writeCNIConfigFile(ctx context.Context, confDir string, confTemplate string, podCIDR string, podCIDRRanges []string, routes []string) error {
	log.G(ctx).Infof("Generating cni config from template %q", confTemplate)
	// generate cni config file from the template with updated pod cidr.
	t, err := template.ParseFiles(confTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse cni config template %q: %w", confTemplate, err)
	}
	if err := my_mk.MkdirAll(confDir, 0755); err != nil {
		return fmt.Errorf("failed to create cni config directory: %q: %w", confDir, err)
	}
	confFile := filepath.Join(confDir, cniConfigFileName)
	f, err := atomicfile.New(confFile, 0o644)
	defer func() {
		err = f.Close()
	}()
	if err := t.Execute(f, cniConfigTemplate{
		PodCIDR:       podCIDR,
		PodCIDRRanges: podCIDRRanges,
		Routes:        routes,
	}); err != nil {
		return fmt.Errorf("failed to generate cni config file %q: %w", confFile, err)
	}
	return err
}
