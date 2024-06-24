package nri

import (
	"time"

	nri "demo/others/over/nri_extend/pkg/adaptation"
)

// Config data for NRI.
type Config struct {
	// Disable this NRI plugin and containerd NRI functionality altogether.
	Disable bool `toml:"disable" json:"disable"`
	// SocketPath is the path to the NRI socket to create for NRI plugins to connect to.
	SocketPath string `toml:"socket_path" json:"socketPath"`
	// PluginPath is the path to search for NRI plugins to launch on startup.
	PluginPath string `toml:"plugin_path" json:"pluginPath"`
	// PluginConfigPath is the path to search for plugin-specific configuration.
	PluginConfigPath string `toml:"plugin_config_path" json:"pluginConfigPath"`
	// PluginRegistrationTimeout is the timeout for plugin registration.
	PluginRegistrationTimeout time.Duration `toml:"plugin_registration_timeout" json:"pluginRegistrationTimeout"`
	// PluginRequestTimeout is the timeout for a plugin to handle a request.
	PluginRequestTimeout time.Duration `toml:"plugin_request_timeout" json:"pluginRequestTimeout"`
	// DisableConnections disables connections from externally launched plugins.
	DisableConnections bool `toml:"disable_connections" json:"disableConnections"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Disable:                   true,
		SocketPath:                nri.DefaultSocketPath,                // /var/run/nri/nri.sock
		PluginPath:                nri.DefaultPluginPath,                // /opt/nri/plugins
		PluginConfigPath:          nri.DefaultPluginConfigPath,          // /etc/nri/conf.d
		PluginRegistrationTimeout: nri.DefaultPluginRegistrationTimeout, // 5s
		PluginRequestTimeout:      nri.DefaultPluginRequestTimeout,      // 2s
	}
}

// toOptions returns NRI options for this configuration.
func (c *Config) toOptions() []nri.Option {
	var opts []nri.Option
	if c.SocketPath != "" {
		opts = append(opts, nri.WithSocketPath(c.SocketPath))
	}
	if c.PluginPath != "" {
		opts = append(opts, nri.WithPluginPath(c.PluginPath))
	}
	if c.PluginConfigPath != "" {
		opts = append(opts, nri.WithPluginConfigPath(c.PluginConfigPath))
	}
	if c.DisableConnections {
		opts = append(opts, nri.WithDisabledExternalConnections())
	}
	return opts
}

// ConfigureTimeouts sets timeout options for NRI.
func (c *Config) ConfigureTimeouts() {
	if c.PluginRegistrationTimeout != 0 {
		nri.SetPluginRegistrationTimeout(c.PluginRegistrationTimeout)
	}
	if c.PluginRequestTimeout != 0 {
		nri.SetPluginRequestTimeout(c.PluginRequestTimeout)
	}
}
