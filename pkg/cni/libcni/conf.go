// Copyright 2015 CNI authors
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

package libcni

import (
	"demo/pkg/cni/pkg/types"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type NotFoundError struct {
	Dir  string
	Name string
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf(`no net configuration with name "%s" in %s`, e.Name, e.Dir)
}

type NoConfigsFoundError struct {
	Dir string
}

func (e NoConfigsFoundError) Error() string {
	return fmt.Sprintf(`no net configurations found in %s`, e.Dir)
}

func ConfListFromBytes(bytes []byte) (*NetworkConfigList, error) {
	rawList := make(map[string]interface{})
	if err := json.Unmarshal(bytes, &rawList); err != nil {
		return nil, fmt.Errorf("error parsing configuration list: %w", err)
	}

	rawName, ok := rawList["name"]
	if !ok {
		return nil, fmt.Errorf("error parsing configuration list: no name")
	}
	name, ok := rawName.(string)
	if !ok {
		return nil, fmt.Errorf("error parsing configuration list: invalid name type %T", rawName)
	}

	var cniVersion string
	rawVersion, ok := rawList["cniVersion"]
	if ok {
		cniVersion, ok = rawVersion.(string)
		if !ok {
			return nil, fmt.Errorf("error parsing configuration list: invalid cniVersion type %T", rawVersion)
		}
	}

	disableCheck := false
	if rawDisableCheck, ok := rawList["disableCheck"]; ok {
		disableCheck, ok = rawDisableCheck.(bool)
		if !ok {
			return nil, fmt.Errorf("error parsing configuration list: invalid disableCheck type %T", rawDisableCheck)
		}
	}

	list := &NetworkConfigList{
		Name:         name,
		DisableCheck: disableCheck,
		CNIVersion:   cniVersion,
		Bytes:        bytes,
	}

	var plugins []interface{}
	plug, ok := rawList["plugins"]
	if !ok {
		return nil, fmt.Errorf("error parsing configuration list: no 'plugins' key")
	}
	plugins, ok = plug.([]interface{})
	if !ok {
		return nil, fmt.Errorf("error parsing configuration list: invalid 'plugins' type %T", plug)
	}
	if len(plugins) == 0 {
		return nil, fmt.Errorf("error parsing configuration list: no plugins in list")
	}

	for i, conf := range plugins {
		newBytes, err := json.Marshal(conf)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal plugin config %d: %w", i, err)
		}
		netConf, err := ConfFromBytes(newBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse plugin config %d: %w", i, err)
		}
		list.Plugins = append(list.Plugins, netConf)
	}

	return list, nil
}

func ConfListFromFile(filename string) (*NetworkConfigList, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %w", filename, err)
	}
	return ConfListFromBytes(bytes)
}

func ConfFiles(dir string, extensions []string) ([]string, error) {
	// In part, adapted from rkt/networking/podenv.go#listFiles
	files, err := ioutil.ReadDir(dir)
	switch {
	case err == nil: // break
	case os.IsNotExist(err):
		return nil, nil
	default:
		return nil, err
	}

	confFiles := []string{}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		fileExt := filepath.Ext(f.Name())
		for _, ext := range extensions {
			if fileExt == ext {
				confFiles = append(confFiles, filepath.Join(dir, f.Name()))
			}
		}
	}
	return confFiles, nil
}

func InjectConf(original *NetworkConfig, newValues map[string]interface{}) (*NetworkConfig, error) {
	config := make(map[string]interface{})
	err := json.Unmarshal(original.Bytes, &config)
	if err != nil {
		return nil, fmt.Errorf("unmarshal existing network bytes: %w", err)
	}

	for key, value := range newValues {
		if key == "" {
			return nil, fmt.Errorf("keys cannot be empty")
		}

		if value == nil {
			return nil, fmt.Errorf("key '%s' value must not be nil", key)
		}

		config[key] = value
	}

	newBytes, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	return ConfFromBytes(newBytes)
}

// ConfListFromConf "upconverts" a network config in to a NetworkConfigList,
// with the single network as the only entry in the list.
func ConfListFromConf(original *NetworkConfig) (*NetworkConfigList, error) {
	// Re-deserialize the config's json, then make a raw map configlist.
	// This may seem a bit strange, but it's to make the Bytes fields
	// actually make sense. Otherwise, the generated json is littered with
	// golang default values.

	rawConfig := make(map[string]interface{})
	if err := json.Unmarshal(original.Bytes, &rawConfig); err != nil {
		return nil, err
	}

	rawConfigList := map[string]interface{}{
		"name":       original.Network.Name,
		"cniVersion": original.Network.CNIVersion,
		"plugins":    []interface{}{rawConfig},
	}

	b, err := json.Marshal(rawConfigList)
	if err != nil {
		return nil, err
	}
	return ConfListFromBytes(b)
}
func ConfFromBytes(bytes []byte) (*NetworkConfig, error) {
	conf := &NetworkConfig{Bytes: bytes, Network: &types.NetConf{}}
	if err := json.Unmarshal(bytes, conf.Network); err != nil {
		return nil, fmt.Errorf("error parsing configuration: %w", err)
	}
	if conf.Network.Type == "" {
		return nil, fmt.Errorf("error parsing configuration: missing 'type'")
	}
	return conf, nil
}

func ConfFromFile(filename string) (*NetworkConfig, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %w", filename, err)
	}
	return ConfFromBytes(bytes)
}
