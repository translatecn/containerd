package nri

import (
	nri "demo/pkg/nri_extend/pkg/adaptation"
)

func podSandboxToNRI(pod PodSandbox) *nri.PodSandbox {
	nriPod := commonPodSandboxToNRI(pod)
	lnxPod := pod.GetLinuxPodSandbox()
	nriPod.Linux = &nri.LinuxPodSandbox{
		Namespaces:   lnxPod.GetLinuxNamespaces(),
		PodOverhead:  lnxPod.GetPodLinuxOverhead(),
		PodResources: lnxPod.GetPodLinuxResources(),
		CgroupParent: lnxPod.GetCgroupParent(),
		CgroupsPath:  lnxPod.GetCgroupsPath(),
		Resources:    lnxPod.GetLinuxResources(),
	}
	return nriPod
}
