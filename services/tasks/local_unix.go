//go:build !windows && !freebsd && !darwin

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

package tasks

import (
	"demo/others/log"
	over_plugin2 "demo/over/plugin"
	"demo/runtime"
	"errors"
)

var tasksServiceRequires = []over_plugin2.Type{
	over_plugin2.EventPlugin,
	over_plugin2.RuntimePlugin,
	over_plugin2.RuntimePluginV2,
	over_plugin2.MetadataPlugin,
	over_plugin2.TaskMonitorPlugin,
	over_plugin2.WarningPlugin,
}

func loadV1Runtimes(ic *over_plugin2.InitContext) (map[string]runtime.PlatformRuntime, error) {
	rt, err := ic.GetByType(over_plugin2.RuntimePlugin)
	if err != nil {
		return nil, err
	}

	runtimes := make(map[string]runtime.PlatformRuntime)
	for _, rr := range rt {
		ri, err := rr.Instance()
		if err != nil {
			log.G(ic.Context).WithError(err).Warn("could not load runtime instance due to initialization error")
			continue
		}
		r := ri.(runtime.PlatformRuntime)
		runtimes[r.ID()] = r
	}

	if len(runtimes) == 0 {
		return nil, errors.New("no runtimes available to create task service")
	}
	return runtimes, nil
}
