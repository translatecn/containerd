package sbserver

import (
	cap2 "demo/pkg/cap"
	"demo/pkg/go-cni"
	"demo/pkg/userns"
	"fmt"

	"github.com/opencontainers/selinux/go-selinux"
	"github.com/sirupsen/logrus"
	"tags.cncf.io/container-device-interface/pkg/cdi"
)

const networkAttachCount = 2

func (c *CriService) initPlatform() (err error) {
	if userns.RunningInUserNS() {
		if c.apparmorEnabled() || !c.config.RestrictOOMScoreAdj {
			logrus.Warn("Running CRI plugin in a user namespace typically requires disable_apparmor and restrict_oom_score_adj to be true")
		}
	}

	if c.config.EnableSelinux {
		if !selinux.GetEnabled() {
			logrus.Warn("Selinux is not supported")
		}
		if r := c.config.SelinuxCategoryRange; r > 0 {
			selinux.CategoryRange = uint32(r)
		}
	} else {
		selinux.SetDisabled()
	}

	pluginDirs := map[string]string{
		defaultNetworkPlugin: c.config.NetworkPluginConfDir,
	}
	for name, conf := range c.config.Runtimes {
		if conf.NetworkPluginConfDir != "" {
			pluginDirs[name] = conf.NetworkPluginConfDir
		}
	}

	c.netPlugin = make(map[string]cni.CNI)
	for name, dir := range pluginDirs {
		max := c.config.NetworkPluginMaxConfNum
		if name != defaultNetworkPlugin {
			if m := c.config.Runtimes[name].NetworkPluginMaxConfNum; m != 0 {
				max = m
			}
		}
		// Pod needs to attach to at least loopback network and a non host network,
		// hence networkAttachCount is 2. If there are more network configs the
		// pod will be attached to all the networks but we will only use the ip
		// of the default network interface as the pod IP.
		i, err := cni.New(cni.WithMinNetworkCount(networkAttachCount),
			cni.WithPluginConfDir(dir),
			cni.WithPluginMaxConfNum(max),
			cni.WithPluginDir([]string{c.config.NetworkPluginBinDir}))
		if err != nil {
			return fmt.Errorf("failed to initialize cni: %w", err)
		}
		c.netPlugin[name] = i
	}

	if c.allCaps == nil {
		c.allCaps, err = cap2.Current()
		if err != nil {
			return fmt.Errorf("failed to get caps: %w", err)
		}
	}

	if c.config.EnableCDI {
		err = cdi.Configure(cdi.WithSpecDirs(c.config.CDISpecDirs...))
		if err != nil {
			return fmt.Errorf("failed to configure CDI registry")
		}
	}

	return nil
}

func (c *CriService) cniLoadOptions() []cni.Opt {
	return []cni.Opt{cni.WithLoNetwork, cni.WithDefaultConf}
}
