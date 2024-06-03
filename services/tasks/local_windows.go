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
	over_plugin2 "demo/over/plugin"
	"demo/runtime"
)

var tasksServiceRequires = []over_plugin2.Type{
	over_plugin2.EventPlugin,
	over_plugin2.RuntimePluginV2,
	over_plugin2.MetadataPlugin,
	over_plugin2.TaskMonitorPlugin,
	over_plugin2.WarningPlugin,
}

// loadV1Runtimes on Windows V2 returns an empty map. There are no v1 runtimes
func loadV1Runtimes(ic *over_plugin2.InitContext) (map[string]runtime.PlatformRuntime, error) {
	return make(map[string]runtime.PlatformRuntime), nil
}
