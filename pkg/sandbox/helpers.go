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

package sandbox

import (
	"demo/others/typeurl/v2"
	over_protobuf2 "demo/over/protobuf"
	gogo_types "demo/over/protobuf/types"
	"demo/pkg/api/types"
)

// ToProto will map Sandbox struct to it's protobuf definition
func ToProto(sandbox *Sandbox) *types.Sandbox {
	extensions := make(map[string]*gogo_types.Any)
	for k, v := range sandbox.Extensions {
		extensions[k] = over_protobuf2.FromAny(v)
	}
	return &types.Sandbox{
		SandboxID: sandbox.ID,
		Runtime: &types.Sandbox_Runtime{
			Name:    sandbox.Runtime.Name,
			Options: over_protobuf2.FromAny(sandbox.Runtime.Options),
		},
		Labels:     sandbox.Labels,
		CreatedAt:  over_protobuf2.ToTimestamp(sandbox.CreatedAt),
		UpdatedAt:  over_protobuf2.ToTimestamp(sandbox.UpdatedAt),
		Extensions: extensions,
		Spec:       over_protobuf2.FromAny(sandbox.Spec),
	}
}

// FromProto map protobuf sandbox definition to Sandbox struct
func FromProto(sandboxpb *types.Sandbox) Sandbox {
	runtime := RuntimeOpts{
		Name:    sandboxpb.Runtime.Name,
		Options: sandboxpb.Runtime.Options,
	}

	extensions := make(map[string]typeurl.Any)
	for k, v := range sandboxpb.Extensions {
		v := v
		extensions[k] = v
	}

	return Sandbox{
		ID:         sandboxpb.SandboxID,
		Labels:     sandboxpb.Labels,
		Runtime:    runtime,
		Spec:       sandboxpb.Spec,
		CreatedAt:  over_protobuf2.FromTimestamp(sandboxpb.CreatedAt),
		UpdatedAt:  over_protobuf2.FromTimestamp(sandboxpb.UpdatedAt),
		Extensions: extensions,
	}
}
