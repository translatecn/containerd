package nri

import (
	nri "demo/others/nri_extend/pkg/adaptation"
)

// Container interface for interacting with NRI.
type Container interface {
	GetDomain() string

	GetPodSandboxID() string
	GetID() string
	GetName() string
	GetState() nri.ContainerState
	GetLabels() map[string]string
	GetAnnotations() map[string]string
	GetArgs() []string
	GetEnv() []string
	GetMounts() []*nri.Mount
	GetHooks() *nri.Hooks
	GetLinuxContainer() LinuxContainer

	GetPid() uint32
}

type LinuxContainer interface {
	GetLinuxNamespaces() []*nri.LinuxNamespace
	GetLinuxDevices() []*nri.LinuxDevice
	GetLinuxResources() *nri.LinuxResources
	GetOOMScoreAdj() *int
	GetCgroupsPath() string
}

func commonContainerToNRI(ctr Container) *nri.Container {
	return &nri.Container{
		Id:           ctr.GetID(),
		PodSandboxId: ctr.GetPodSandboxID(),
		Name:         ctr.GetName(),
		State:        ctr.GetState(),
		Labels:       ctr.GetLabels(),
		Annotations:  ctr.GetAnnotations(),
		Args:         ctr.GetArgs(),
		Env:          ctr.GetEnv(),
		Mounts:       ctr.GetMounts(),
		Hooks:        ctr.GetHooks(),
		Pid:          ctr.GetPid(),
	}
}

func containersToNRI(ctrList []Container) []*nri.Container {
	ctrs := []*nri.Container{}
	for _, ctr := range ctrList {
		ctrs = append(ctrs, containerToNRI(ctr))
	}
	return ctrs
}
