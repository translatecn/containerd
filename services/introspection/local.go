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

package introspection

import (
	context "context"
	"demo/over/my_mk"
	over_plugin2 "demo/over/plugin"
	"demo/over/protobuf"
	ptypes "demo/over/protobuf/types"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/rpc/code"
	rpc "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"demo/over/errdefs"
	api "demo/pkg/api/services/introspection/v1"
	"demo/pkg/api/types"
	"demo/pkg/filters"
	"demo/services"
	"demo/services/warning"
)

func init() {
	over_plugin2.Register(&over_plugin2.Registration{
		Type:     over_plugin2.ServicePlugin,
		ID:       services.IntrospectionService,
		Requires: []over_plugin2.Type{over_plugin2.WarningPlugin},
		InitFn: func(ic *over_plugin2.InitContext) (interface{}, error) {
			sps, err := ic.GetByType(over_plugin2.WarningPlugin)
			if err != nil {
				return nil, err
			}
			p, ok := sps[over_plugin2.DeprecationsPlugin]
			if !ok {
				return nil, errors.New("warning service not found")
			}

			i, err := p.Instance()
			if err != nil {
				return nil, err
			}

			warningClient, ok := i.(warning.Service)
			if !ok {
				return nil, errors.New("could not create a local client for warning service")
			}

			// this service fetches all plugins through the plugin set of the plugin context
			return &Local{
				plugins:       ic.Plugins(),
				root:          ic.Root,
				warningClient: warningClient,
			}, nil
		},
	})
}

// Local is a local implementation of the introspection service
type Local struct {
	mu            sync.Mutex
	root          string
	plugins       *over_plugin2.Set
	pluginCache   []*api.Plugin
	warningClient warning.Service
}

var _ = (api.IntrospectionClient)(&Local{})

// UpdateLocal updates the local introspection service
func (l *Local) UpdateLocal(root string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.root = root
}

// Plugins returns the locally defined plugins
func (l *Local) Plugins(ctx context.Context, req *api.PluginsRequest, _ ...grpc.CallOption) (*api.PluginsResponse, error) {
	filter, err := filters.ParseAll(req.Filters...)
	if err != nil {
		return nil, over_errdefs.ToGRPCf(over_errdefs.ErrInvalidArgument, err.Error())
	}

	var plugins []*api.Plugin
	allPlugins := l.getPlugins()
	for _, p := range allPlugins {
		p := p
		if filter.Match(adaptPlugin(p)) {
			plugins = append(plugins, p)
		}
	}

	return &api.PluginsResponse{
		Plugins: plugins,
	}, nil
}

func (l *Local) getPlugins() []*api.Plugin {
	l.mu.Lock()
	defer l.mu.Unlock()
	plugins := l.plugins.GetAll()
	if l.pluginCache == nil || len(plugins) != len(l.pluginCache) {
		l.pluginCache = pluginsToPB(plugins)
	}
	return l.pluginCache
}

// Server returns the local server information
func (l *Local) Server(ctx context.Context, _ *ptypes.Empty, _ ...grpc.CallOption) (*api.ServerResponse, error) {
	u, err := l.getUUID()
	if err != nil {
		return nil, over_errdefs.ToGRPC(err)
	}
	pid := os.Getpid()
	var pidns uint64
	if runtime.GOOS == "linux" {
		pidns, err = statPIDNS(pid)
		if err != nil {
			return nil, over_errdefs.ToGRPC(err)
		}
	}
	return &api.ServerResponse{
		UUID:         u,
		Pid:          uint64(pid),
		Pidns:        pidns,
		Deprecations: l.getWarnings(ctx),
	}, nil
}

func (l *Local) getUUID() (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := os.ReadFile(l.uuidPath())
	if err != nil {
		if os.IsNotExist(err) {
			return l.generateUUID()
		}
		return "", err
	}
	u := string(data)
	if _, err := uuid.Parse(u); err != nil {
		return "", err
	}
	return u, nil
}

func (l *Local) generateUUID() (string, error) {
	u, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	path := l.uuidPath()
	if err := my_mk.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return "", err
	}
	uu := u.String()
	if err := os.WriteFile(path, []byte(uu), 0666); err != nil {
		return "", err
	}
	return uu, nil
}

func (l *Local) uuidPath() string {
	return filepath.Join(l.root, "uuid")
}

func (l *Local) getWarnings(ctx context.Context) []*api.DeprecationWarning {
	return warningsPB(ctx, l.warningClient.Warnings())
}

func adaptPlugin(o interface{}) filters.Adaptor {
	obj := o.(*api.Plugin)
	return filters.AdapterFunc(func(fieldpath []string) (string, bool) {
		if len(fieldpath) == 0 {
			return "", false
		}

		switch fieldpath[0] {
		case "type":
			return obj.Type, len(obj.Type) > 0
		case "id":
			return obj.ID, len(obj.ID) > 0
		case "platforms":
			// TODO(stevvooe): Another case here where have multiple values.
			// May need to refactor the filter system to allow filtering by
			// platform, if this is required.
		case "capabilities":
			// TODO(stevvooe): Need a better way to match against
			// collections. We can only return "the value" but really it
			// would be best if we could return a set of values for the
			// path, any of which could match.
		}

		return "", false
	})
}

func pluginsToPB(plugins []*over_plugin2.Plugin) []*api.Plugin {
	var pluginsPB []*api.Plugin
	for _, p := range plugins {
		var platforms []*types.Platform
		for _, p := range p.Meta.Platforms {
			platforms = append(platforms, &types.Platform{
				OS:           p.OS,
				Architecture: p.Architecture,
				Variant:      p.Variant,
			})
		}

		var requires []string
		for _, r := range p.Registration.Requires {
			requires = append(requires, r.String())
		}

		var initErr *rpc.Status
		if err := p.Err(); err != nil {
			st, ok := status.FromError(over_errdefs.ToGRPC(err))
			if ok {
				var details []*ptypes.Any
				for _, d := range st.Proto().Details {
					details = append(details, &ptypes.Any{
						TypeUrl: d.TypeUrl,
						Value:   d.Value,
					})
				}
				initErr = &rpc.Status{
					Code:    int32(st.Code()),
					Message: st.Message(),
					Details: details,
				}
			} else {
				initErr = &rpc.Status{
					Code:    int32(code.Code_UNKNOWN),
					Message: err.Error(),
				}
			}
		}

		pluginsPB = append(pluginsPB, &api.Plugin{
			Type:         p.Registration.Type.String(),
			ID:           p.Registration.ID,
			Requires:     requires,
			Platforms:    platforms,
			Capabilities: p.Meta.Capabilities,
			Exports:      p.Meta.Exports,
			InitErr:      initErr,
		})
	}

	return pluginsPB
}

func warningsPB(ctx context.Context, warnings []warning.Warning) []*api.DeprecationWarning {
	var pb []*api.DeprecationWarning

	for _, w := range warnings {
		pb = append(pb, &api.DeprecationWarning{
			ID:             string(w.ID),
			Message:        w.Message,
			LastOccurrence: over_protobuf.ToTimestamp(w.LastOccurrence),
		})
	}
	return pb
}
