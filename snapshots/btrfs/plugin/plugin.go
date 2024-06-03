//go:build linux && !no_btrfs && cgo

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

package plugin

import (
	over_plugin2 "demo/over/plugin"
	"errors"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"demo/over/platforms"
	"demo/snapshots/btrfs"
)

// Config represents configuration for the btrfs plugin.
type Config struct {
	// Root directory for the plugin
	RootPath string `toml:"root_path"`
}

func init() {
	over_plugin2.Register(&over_plugin2.Registration{
		ID:     "btrfs",
		Type:   over_plugin2.SnapshotPlugin,
		Config: &Config{},
		InitFn: func(ic *over_plugin2.InitContext) (interface{}, error) {
			ic.Meta.Platforms = []ocispec.Platform{over_platforms.DefaultSpec()}

			config, ok := ic.Config.(*Config)
			if !ok {
				return nil, errors.New("invalid btrfs configuration")
			}

			root := ic.Root
			if len(config.RootPath) != 0 {
				root = config.RootPath
			}

			ic.Meta.Exports[over_plugin2.SnapshotterRootDir] = root
			return btrfs.NewSnapshotter(root)
		},
	})
}
