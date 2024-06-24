package api

import (
	"fmt"
	"strings"
)

const (
	// DefaultSocketPath is the default socket path for external plugins.
	DefaultSocketPath = "/var/run/nri/nri.sock"
	// PluginSocketEnvVar is used to inform plugins about pre-connected sockets.
	PluginSocketEnvVar = "NRI_PLUGIN_SOCKET"
	// PluginNameEnvVar is used to inform NRI-launched plugins about their name.
	PluginNameEnvVar = "NRI_PLUGIN_NAME"
	// PluginIdxEnvVar is used to inform NRI-launched plugins about their ID.
	PluginIdxEnvVar = "NRI_PLUGIN_IDX"
)

// ParsePluginName parses the (file)name of a plugin into an index and a base.
func ParsePluginName(name string) (string, string, error) {
	split := strings.SplitN(name, "-", 2)
	if len(split) < 2 {
		return "", "", fmt.Errorf("invalid plugin name %q, idx-pluginname expected", name)
	}

	if err := CheckPluginIndex(split[0]); err != nil {
		return "", "", err
	}

	return split[0], split[1], nil
}

// CheckPluginIndex checks the validity of a plugin index.
func CheckPluginIndex(idx string) error {
	if len(idx) != 2 {
		return fmt.Errorf("invalid plugin index %q, must be 2 digits", idx)
	}
	if !('0' <= idx[0] && idx[0] <= '9') || !('0' <= idx[1] && idx[1] <= '9') {
		return fmt.Errorf("invalid plugin index %q (not [0-9][0-9])", idx)
	}
	return nil
}
