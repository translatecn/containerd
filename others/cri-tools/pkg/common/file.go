/*
Copyright 2020 The Kubernetes Authors.

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

package common

import (
	"fmt"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config is the internal representation of the yaml that defines
// server configuration
type Config struct {
	RuntimeEndpoint   string
	ImageEndpoint     string
	Timeout           int
	Debug             bool
	PullImageOnCreate bool
	DisablePullOnRun  bool
	yamlData          *yaml.Node //YAML representation of config
}

// ReadConfig reads from a file with the given name and returns a config or
// an error if the file was unable to be parsed.

// WriteConfig writes config to file
// an error if the file was unable to be written to.

// Extracts config options from the yaml data which is loaded from file
func getConfigOptions(yamlData yaml.Node) (*Config, error) {
	config := Config{}
	config.yamlData = &yamlData

	if yamlData.Content == nil || len(yamlData.Content) == 0 ||
		yamlData.Content[0].Content == nil {
		return &config, nil
	}
	contentLen := len(yamlData.Content[0].Content)

	// YAML representation contains 2 yaml ScalarNodes per config option.
	// One is config option name and other is the value of the option
	// These ScalarNodes help preserve comments associated with
	// the YAML entry
	for indx := 0; indx < contentLen-1; {
		configOption := yamlData.Content[0].Content[indx]
		name := configOption.Value
		value := yamlData.Content[0].Content[indx+1].Value
		var err error
		switch name {
		case "runtime-endpoint":
			config.RuntimeEndpoint = value
		case "image-endpoint":
			config.ImageEndpoint = value
		case "timeout":
			config.Timeout, err = strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("parsing config option '%s': %w", name, err)
			}
		case "debug":
			config.Debug, err = strconv.ParseBool(value)
			if err != nil {
				return nil, fmt.Errorf("parsing config option '%s': %w", name, err)
			}
		case "pull-image-on-create":
			config.PullImageOnCreate, err = strconv.ParseBool(value)
			if err != nil {
				return nil, fmt.Errorf("parsing config option '%s': %w", name, err)
			}
		case "disable-pull-on-run":
			config.DisablePullOnRun, err = strconv.ParseBool(value)
			if err != nil {
				return nil, fmt.Errorf("parsing config option '%s': %w", name, err)
			}
		default:
			return nil, fmt.Errorf("Config option '%s' is not valid", name)
		}
		indx += 2
	}

	return &config, nil
}

// Set config options on yaml data for persistece to file

// Set config option on yaml
func setConfigOption(configName, configValue string, yamlData *yaml.Node) {
	if yamlData.Content == nil || len(yamlData.Content) == 0 {
		yamlData.Kind = yaml.DocumentNode
		yamlData.Content = make([]*yaml.Node, 1)
		yamlData.Content[0] = &yaml.Node{
			Kind: yaml.MappingNode,
			Tag:  "!!map",
		}
	}
	var contentLen = 0
	var foundOption = false
	if yamlData.Content[0].Content != nil {
		contentLen = len(yamlData.Content[0].Content)
	}

	// Set value on existing config option
	for indx := 0; indx < contentLen-1; {
		name := yamlData.Content[0].Content[indx].Value
		if name == configName {
			yamlData.Content[0].Content[indx+1].Value = configValue
			foundOption = true
			break
		}
		indx += 2
	}

	// New config option to set
	// YAML representation contains 2 yaml ScalarNodes per config option.
	// One is config option name and other is the value of the option
	// These ScalarNodes help preserve comments associated with
	// the YAML entry
	if !foundOption {
		name := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: configName,
			Tag:   "!!str",
		}
		var tagType string
		switch configName {
		case "timeout":
			tagType = "!!int"
		case "debug":
			tagType = "!!bool"
		case "pull-image-on-create":
			tagType = "!!bool"
		case "disable-pull-on-run":
			tagType = "!!bool"
		default:
			tagType = "!!str"
		}

		value := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: configValue,
			Tag:   tagType,
		}
		yamlData.Content[0].Content = append(yamlData.Content[0].Content, name, value)
	}
}
