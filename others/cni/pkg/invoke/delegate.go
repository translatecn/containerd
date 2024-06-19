// Copyright 2016 CNI authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package invoke

import (
	"os"
	"path/filepath"
)

func delegateCommon(delegatePlugin string, exec Exec) (string, Exec, error) {
	if exec == nil {
		exec = defaultExec
	}

	paths := filepath.SplitList(os.Getenv("CNI_PATH"))
	pluginPath, err := exec.FindInPath(delegatePlugin, paths)
	if err != nil {
		return "", nil, err
	}

	return pluginPath, exec, nil
}

// DelegateAdd calls the given delegate plugin with the CNI ADD action and
// JSON configuration

// DelegateCheck calls the given delegate plugin with the CNI CHECK action and
// JSON configuration

// DelegateDel calls the given delegate plugin with the CNI DEL action and
// JSON configuration

// return CNIArgs used by delegation
func delegateArgs(action string) *DelegateArgs {
	return &DelegateArgs{
		Command: action,
	}
}
