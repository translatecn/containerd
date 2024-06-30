package nri

import (
	nri "demo/pkg/nri_extend/pkg/adaptation"
)

func containerToNRI(ctr Container) *nri.Container {
	nriCtr := commonContainerToNRI(ctr)
	lnxCtr := ctr.GetLinuxContainer()
	nriCtr.Linux = &nri.LinuxContainer{
		Namespaces:  lnxCtr.GetLinuxNamespaces(),
		Devices:     lnxCtr.GetLinuxDevices(),
		Resources:   lnxCtr.GetLinuxResources(),
		OomScoreAdj: nri.Int(lnxCtr.GetOOMScoreAdj()),
		CgroupsPath: lnxCtr.GetCgroupsPath(),
	}
	return nriCtr
}
