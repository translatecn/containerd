//go:build gofuzz

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

package config

import (
	"demo/over/my_mk"
	"os"

	fuzz "github.com/AdaLogics/go-fuzz-headers"
)

func FuzzParseHostsFile(data []byte) int {
	f := fuzz.NewConsumer(data)
	dir, err := my_mk.MkdirTemp("", "fuzz-")
	if err != nil {
		return 0
	}
	err = f.CreateFiles(dir)
	if err != nil {
		return 0
	}
	defer os.RemoveAll(dir)
	b, err := f.GetBytes()
	if err != nil {
		return 0
	}
	_, _ = parseHostsFile(dir, b)
	return 1
}
