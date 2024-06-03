//go:build !linux

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

package opts

import (
	"context"

	"demo/containers"
	"demo/over/oci"
)

func isHugetlbControllerPresent() bool {
	return false
}

func SwapControllerAvailable() bool {
	return false
}

// WithCDI does nothing on non Linux platforms.
func WithCDI(_ map[string]string) over_oci.SpecOpts {
	return func(ctx context.Context, client over_oci.Client, container *containers.Container, spec *over_oci.Spec) error {
		return nil
	}
}
