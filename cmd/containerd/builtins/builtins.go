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

package builtins

// register containerd builtins here
import (
	_ "demo/diff/walking/plugin"
	_ "demo/pkg/events/plugin"
	_ "demo/pkg/gc/scheduler"
	_ "demo/pkg/leases/plugin"
	_ "demo/pkg/metadata/plugin"
	_ "demo/pkg/nri/plugin"
	_ "demo/pkg/plugins/sandbox"
	_ "demo/pkg/plugins/streaming"
	_ "demo/pkg/plugins/transfer"
	_ "demo/runtime/restart/monitor"
	_ "demo/runtime/v2"
	_ "demo/services/containers"
	_ "demo/services/content"
	_ "demo/services/diff"
	_ "demo/services/events"
	_ "demo/services/healthcheck"
	_ "demo/services/images"
	_ "demo/services/introspection"
	_ "demo/services/leases"
	_ "demo/services/namespaces"
	_ "demo/services/opt"
	_ "demo/services/sandbox"
	_ "demo/services/snapshots"
	_ "demo/services/streaming"
	_ "demo/services/tasks"
	_ "demo/services/transfer"
	_ "demo/services/version"
	_ "demo/services/warning"
)
