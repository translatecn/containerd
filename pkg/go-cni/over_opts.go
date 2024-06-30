package cni

import (
	"fmt"
	"os"
	"sort"
	"strings"

	cnilibrary "demo/pkg/cni/libcni"
	"demo/pkg/cni/pkg/invoke"
	"demo/pkg/cni/pkg/version"
)

// Opt sets options for a CNI instance
type Opt func(c *libcni) error

// WithInterfacePrefix sets the prefix for network interfaces
// e.g. eth or wlan

// WithPluginDir can be used to set the locations of
// the cni plugin binaries
func WithPluginDir(dirs []string) Opt {
	return func(c *libcni) error {
		c.pluginDirs = dirs
		c.cniConfig = cnilibrary.NewCNIConfig(
			dirs,
			&invoke.DefaultExec{
				RawExec:       &invoke.RawExec{Stderr: os.Stderr},
				PluginDecoder: version.PluginDecoder{},
			},
		)
		return nil
	}
}

// WithPluginConfDir can be used to configure the
// cni configuration directory.
func WithPluginConfDir(dir string) Opt {
	return func(c *libcni) error {
		c.pluginConfDir = dir
		return nil
	}
}

// WithPluginMaxConfNum can be used to configure the
// max cni plugin config file num.
func WithPluginMaxConfNum(max int) Opt {
	return func(c *libcni) error {
		c.pluginMaxConfNum = max
		return nil
	}
}

// WithMinNetworkCount can be used to configure the
// minimum networks to be configured and initialized
// for the status to report success. By default its 1.
func WithMinNetworkCount(count int) Opt {
	return func(c *libcni) error {
		c.networkCount = count
		return nil
	}
}

// WithLoNetwork can be used to load the loopback
// network config.
func WithLoNetwork(c *libcni) error {
	loConfig, _ := cnilibrary.ConfListFromBytes([]byte(`{
"cniVersion": "0.3.1",
"name": "cni-loopback",
"plugins": [{
  "type": "loopback"
}]
}`))

	c.networks = append(c.networks, &Network{
		cni:    c.cniConfig,
		config: loConfig,
		ifName: "lo",
	})
	return nil
}

func WithDefaultConf(c *libcni) error {
	return loadFromConfDir(c, c.pluginMaxConfNum)
}

// WithAllConf can be used to detect all network config
// files from the configured cni config directory and load
// them.

// loadFromConfDir detects network config files from the
// configured cni config directory and load them. max is
// the maximum network config to load (max i<= 0 means no limit).
func loadFromConfDir(c *libcni, max int) error {
	files, err := cnilibrary.ConfFiles(c.pluginConfDir, []string{".conf", ".conflist", ".json"})
	switch {
	case err != nil:
		return fmt.Errorf("failed to read config file: %v: %w", err, ErrRead)
	case len(files) == 0:
		return fmt.Errorf("no network config found in %s: %w", c.pluginConfDir, ErrCNINotInitialized)
	}

	// files contains the network config files associated with cni network.
	// Use lexicographical way as a defined order for network config files.
	sort.Strings(files)
	// Since the CNI spec does not specify a way to detect default networks,
	// the convention chosen is - the first network configuration in the sorted
	// list of network conf files as the default network and choose the default
	// interface provided during init as the network interface for this default
	// network. For every other network use a generated interface id.
	i := 0
	var networks []*Network
	for _, confFile := range files {
		var confList *cnilibrary.NetworkConfigList
		if strings.HasSuffix(confFile, ".conflist") {
			confList, err = cnilibrary.ConfListFromFile(confFile)
			if err != nil {
				return fmt.Errorf("failed to load CNI config list file %s: %v: %w", confFile, err, ErrInvalidConfig)
			}
		} else {
			conf, err := cnilibrary.ConfFromFile(confFile)
			if err != nil {
				return fmt.Errorf("failed to load CNI config file %s: %v: %w", confFile, err, ErrInvalidConfig)
			}
			// Ensure the config has a "type" so we know what plugin to run.
			// Also catches the case where somebody put a conflist into a conf file.
			if conf.Network.Type == "" {
				return fmt.Errorf("network type not found in %s: %w", confFile, ErrInvalidConfig)
			}

			confList, err = cnilibrary.ConfListFromConf(conf)
			if err != nil {
				return fmt.Errorf("failed to convert CNI config file %s to CNI config list: %v: %w", confFile, err, ErrInvalidConfig)
			}
		}
		if len(confList.Plugins) == 0 {
			return fmt.Errorf("CNI config list in config file %s has no networks, skipping: %w", confFile, ErrInvalidConfig)

		}
		networks = append(networks, &Network{
			cni:    c.cniConfig,
			config: confList,
			ifName: getIfName(c.prefix, i),
		})
		i++
		if i == max {
			break
		}
	}
	if len(networks) == 0 {
		return fmt.Errorf("no valid networks found in %s: %w", c.pluginDirs, ErrCNINotInitialized)
	}
	c.networks = append(c.networks, networks...)
	return nil
}
