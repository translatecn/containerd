/*
   Copyright The containerd Authors.

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

package server

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"demo/others/cgroups/v3"
	"demo/over/oci"
	"demo/pkg/contrib/apparmor"
	"demo/pkg/contrib/seccomp"
	"demo/snapshots"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	selinux "github.com/opencontainers/selinux/go-selinux"
	"github.com/opencontainers/selinux/go-selinux/label"
	runtime "k8s.io/cri-api/pkg/apis/runtime/v1"

	"demo/pkg/blockio"
	"demo/pkg/cri/annotations"
	"demo/pkg/cri/config"
	customopts "demo/pkg/cri/opts"
)

const (
	// profileNamePrefix is the prefix for loading profiles on a localhost. Eg. AppArmor localhost/profileName.
	profileNamePrefix = "localhost/" // TODO (mikebrow): get localhost/ & runtime/default from CRI kubernetes/kubernetes#51747
	// runtimeDefault indicates that we should use or create a runtime default profile.
	runtimeDefault = "runtime/default"
	// dockerDefault indicates that we should use or create a docker default profile.
	dockerDefault = "docker/default"
	// appArmorDefaultProfileName is name to use when creating a default apparmor profile.
	appArmorDefaultProfileName = "cri-containerd.apparmor.d"
	// unconfinedProfile is a string indicating one should run a pod/containerd without a security profile
	unconfinedProfile = "unconfined"
	// seccompDefaultProfile is the default seccomp profile.
	seccompDefaultProfile = dockerDefault
)

// containerMounts sets up necessary container system file mounts
// including /dev/shm, /etc/hosts and /etc/resolv.conf.
func (c *criService) containerMounts(sandboxID string, config *runtime.ContainerConfig) []*runtime.Mount {
	var mounts []*runtime.Mount
	securityContext := config.GetLinux().GetSecurityContext()
	if !isInCRIMounts(etcHostname, config.GetMounts()) {
		// /etc/hostname is added since 1.1.6, 1.2.4 and 1.3.
		// For in-place upgrade, the old sandbox doesn't have the hostname file,
		// do not mount this in that case.
		// TODO(random-liu): Remove the check and always mount this when
		// containerd 1.1 and 1.2 are deprecated.
		hostpath := c.getSandboxHostname(sandboxID)
		if _, err := c.os.Stat(hostpath); err == nil {
			mounts = append(mounts, &runtime.Mount{
				ContainerPath:  etcHostname,
				HostPath:       hostpath,
				Readonly:       securityContext.GetReadonlyRootfs(),
				SelinuxRelabel: true,
			})
		}
	}

	if !isInCRIMounts(etcHosts, config.GetMounts()) {
		mounts = append(mounts, &runtime.Mount{
			ContainerPath:  etcHosts,
			HostPath:       c.getSandboxHosts(sandboxID),
			Readonly:       securityContext.GetReadonlyRootfs(),
			SelinuxRelabel: true,
		})
	}

	// Mount sandbox resolv.config.
	// TODO: Need to figure out whether we should always mount it as read-only
	if !isInCRIMounts(resolvConfPath, config.GetMounts()) {
		mounts = append(mounts, &runtime.Mount{
			ContainerPath:  resolvConfPath,
			HostPath:       c.getResolvPath(sandboxID),
			Readonly:       securityContext.GetReadonlyRootfs(),
			SelinuxRelabel: true,
		})
	}

	if !isInCRIMounts(devShm, config.GetMounts()) {
		sandboxDevShm := c.getSandboxDevShm(sandboxID)
		if securityContext.GetNamespaceOptions().GetIpc() == runtime.NamespaceMode_NODE {
			sandboxDevShm = devShm
		}
		mounts = append(mounts, &runtime.Mount{
			ContainerPath:  devShm,
			HostPath:       sandboxDevShm,
			Readonly:       false,
			SelinuxRelabel: sandboxDevShm != devShm,
		})
	}
	return mounts
}

func (c *criService) containerSpec(
	id string,
	sandboxID string,
	sandboxPid uint32,
	netNSPath string,
	containerName string,
	imageName string,
	config *runtime.ContainerConfig,
	sandboxConfig *runtime.PodSandboxConfig,
	imageConfig *imagespec.ImageConfig,
	extraMounts []*runtime.Mount,
	ociRuntime config.Runtime,
) (_ *runtimespec.Spec, retErr error) {
	specOpts := []over_oci.SpecOpts{
		over_oci.WithoutRunMount,
	}
	// only clear the default security settings if the runtime does not have a custom
	// base runtime spec spec.  Admins can use this functionality to define
	// default ulimits, seccomp, or other default settings.
	if ociRuntime.BaseRuntimeSpec == "" {
		specOpts = append(specOpts, customopts.WithoutDefaultSecuritySettings)
	}
	specOpts = append(specOpts,
		customopts.WithRelativeRoot(relativeRootfsPath),
		customopts.WithProcessArgs(config, imageConfig),
		over_oci.WithDefaultPathEnv,
		// this will be set based on the security context below
		over_oci.WithNewPrivileges,
	)
	if config.GetWorkingDir() != "" {
		specOpts = append(specOpts, over_oci.WithProcessCwd(config.GetWorkingDir()))
	} else if imageConfig.WorkingDir != "" {
		specOpts = append(specOpts, over_oci.WithProcessCwd(imageConfig.WorkingDir))
	}

	if config.GetTty() {
		specOpts = append(specOpts, over_oci.WithTTY)
	}

	// Add HOSTNAME env.
	var (
		err      error
		hostname = sandboxConfig.GetHostname()
	)
	if hostname == "" {
		if hostname, err = c.os.Hostname(); err != nil {
			return nil, err
		}
	}
	specOpts = append(specOpts, over_oci.WithEnv([]string{hostnameEnv + "=" + hostname}))

	// Apply envs from image config first, so that envs from container config
	// can override them.
	env := append([]string{}, imageConfig.Env...)
	for _, e := range config.GetEnvs() {
		env = append(env, e.GetKey()+"="+e.GetValue())
	}
	specOpts = append(specOpts, over_oci.WithEnv(env))

	securityContext := config.GetLinux().GetSecurityContext()
	labelOptions, err := toLabel(securityContext.GetSelinuxOptions())
	if err != nil {
		return nil, err
	}
	if len(labelOptions) == 0 {
		// Use pod level SELinux config
		if sandbox, err := c.sandboxStore.Get(sandboxID); err == nil {
			labelOptions, err = selinux.DupSecOpt(sandbox.ProcessLabel)
			if err != nil {
				return nil, err
			}
		}
	}

	processLabel, mountLabel, err := label.InitLabels(labelOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to init selinux options %+v: %w", securityContext.GetSelinuxOptions(), err)
	}
	defer func() {
		if retErr != nil {
			selinux.ReleaseLabel(processLabel)
		}
	}()

	specOpts = append(specOpts, customopts.WithMounts(c.os, config, extraMounts, mountLabel))

	if !c.config.DisableProcMount {
		// Change the default masked/readonly paths to empty slices
		// See https://github.com/containerd/issues/5029
		// TODO: Provide an option to set default paths to the ones in oci.populateDefaultUnixSpec()
		specOpts = append(specOpts, over_oci.WithMaskedPaths([]string{}), over_oci.WithReadonlyPaths([]string{}))

		// Apply masked paths if specified.
		// If the container is privileged, this will be cleared later on.
		if maskedPaths := securityContext.GetMaskedPaths(); maskedPaths != nil {
			specOpts = append(specOpts, over_oci.WithMaskedPaths(maskedPaths))
		}

		// Apply readonly paths if specified.
		// If the container is privileged, this will be cleared later on.
		if readonlyPaths := securityContext.GetReadonlyPaths(); readonlyPaths != nil {
			specOpts = append(specOpts, over_oci.WithReadonlyPaths(readonlyPaths))
		}
	}

	specOpts = append(specOpts, customopts.WithDevices(c.os, config, c.config.DeviceOwnershipFromSecurityContext),
		customopts.WithCapabilities(securityContext, c.allCaps))

	if securityContext.GetPrivileged() {
		if !sandboxConfig.GetLinux().GetSecurityContext().GetPrivileged() {
			return nil, errors.New("no privileged container allowed in sandbox")
		}
		specOpts = append(specOpts, over_oci.WithPrivileged)
		if !ociRuntime.PrivilegedWithoutHostDevices {
			specOpts = append(specOpts, over_oci.WithHostDevices, over_oci.WithAllDevicesAllowed)
		} else if ociRuntime.PrivilegedWithoutHostDevicesAllDevicesAllowed {
			// allow rwm on all devices for the container
			specOpts = append(specOpts, over_oci.WithAllDevicesAllowed)
		}
	}

	// Clear all ambient capabilities. The implication of non-root + caps
	// is not clearly defined in Kubernetes.
	// See https://github.com/kubernetes/kubernetes/issues/56374
	// Keep docker's behavior for now.
	specOpts = append(specOpts,
		customopts.WithoutAmbientCaps,
		customopts.WithSelinuxLabels(processLabel, mountLabel),
	)

	// TODO: Figure out whether we should set no new privilege for sandbox container by default
	if securityContext.GetNoNewPrivs() {
		specOpts = append(specOpts, over_oci.WithNoNewPrivileges)
	}
	// TODO(random-liu): [P1] Set selinux options (privileged or not).
	if securityContext.GetReadonlyRootfs() {
		specOpts = append(specOpts, over_oci.WithRootFSReadonly())
	}

	if c.config.DisableCgroup {
		specOpts = append(specOpts, customopts.WithDisabledCgroups)
	} else {
		specOpts = append(specOpts, customopts.WithResources(config.GetLinux().GetResources(), c.config.TolerateMissingHugetlbController, c.config.DisableHugetlbController))
		if sandboxConfig.GetLinux().GetCgroupParent() != "" {
			cgroupsPath := getCgroupsPath(sandboxConfig.GetLinux().GetCgroupParent(), id)
			specOpts = append(specOpts, over_oci.WithCgroup(cgroupsPath))
		}
	}

	supplementalGroups := securityContext.GetSupplementalGroups()

	// Get blockio class
	blockIOClass, err := c.blockIOClassFromAnnotations(config.GetMetadata().GetName(), config.Annotations, sandboxConfig.Annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to set blockio class: %w", err)
	}
	if blockIOClass != "" {
		if linuxBlockIO, err := blockio.ClassNameToLinuxOCI(blockIOClass); err == nil {
			specOpts = append(specOpts, over_oci.WithBlockIO(linuxBlockIO))
		} else {
			return nil, err
		}
	}

	// Get RDT class
	rdtClass, err := c.rdtClassFromAnnotations(config.GetMetadata().GetName(), config.Annotations, sandboxConfig.Annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to set RDT class: %w", err)
	}
	if rdtClass != "" {
		specOpts = append(specOpts, over_oci.WithRdt(rdtClass, "", ""))
	}

	for pKey, pValue := range getPassthroughAnnotations(sandboxConfig.Annotations,
		ociRuntime.PodAnnotations) {
		specOpts = append(specOpts, customopts.WithAnnotation(pKey, pValue))
	}

	for pKey, pValue := range getPassthroughAnnotations(config.Annotations,
		ociRuntime.ContainerAnnotations) {
		specOpts = append(specOpts, customopts.WithAnnotation(pKey, pValue))
	}

	// Default target PID namespace is the sandbox PID.
	targetPid := sandboxPid
	// If the container targets another container's PID namespace,
	// set targetPid to the PID of that container.
	nsOpts := securityContext.GetNamespaceOptions()
	if nsOpts.GetPid() == runtime.NamespaceMode_TARGET {
		targetContainer, err := c.validateTargetContainer(sandboxID, nsOpts.TargetId)
		if err != nil {
			return nil, fmt.Errorf("invalid target container: %w", err)
		}

		status := targetContainer.Status.Get()
		targetPid = status.Pid
	}

	uids, gids, err := parseUsernsIDs(nsOpts.GetUsernsOptions())
	if err != nil {
		return nil, fmt.Errorf("user namespace configuration: %w", err)
	}

	// Check sandbox userns config is consistent with container config.
	sandboxUsernsOpts := sandboxConfig.GetLinux().GetSecurityContext().GetNamespaceOptions().GetUsernsOptions()
	if !sameUsernsConfig(sandboxUsernsOpts, nsOpts.GetUsernsOptions()) {
		return nil, fmt.Errorf("user namespace config for sandbox is different from container. Sandbox userns config: %v - Container userns config: %v", sandboxUsernsOpts, nsOpts.GetUsernsOptions())
	}

	specOpts = append(specOpts,
		customopts.WithOOMScoreAdj(config, c.config.RestrictOOMScoreAdj),
		customopts.WithPodNamespaces(securityContext, sandboxPid, targetPid, uids, gids),
		customopts.WithSupplementalGroups(supplementalGroups),
	)
	specOpts = append(specOpts,
		annotations.DefaultCRIAnnotations(sandboxID, containerName, imageName, sandboxConfig, false)...,
	)
	// cgroupns is used for hiding /sys/fs/cgroup from containers.
	// For compatibility, cgroupns is not used when running in cgroup v1 mode or in privileged.
	// https://github.com/containers/libpod/issues/4363
	// https://github.com/kubernetes/enhancements/blob/0e409b47497e398b369c281074485c8de129694f/keps/sig-node/20191118-cgroups-v2.md#cgroup-namespace
	if cgroups.Mode() == cgroups.Unified && !securityContext.GetPrivileged() {
		specOpts = append(specOpts, over_oci.WithLinuxNamespace(
			runtimespec.LinuxNamespace{
				Type: runtimespec.CgroupNamespace,
			}))
	}
	return c.runtimeSpec(id, ociRuntime.BaseRuntimeSpec, specOpts...)
}

func (c *criService) containerSpecOpts(config *runtime.ContainerConfig, imageConfig *imagespec.ImageConfig) ([]over_oci.SpecOpts, error) {
	var specOpts []over_oci.SpecOpts
	securityContext := config.GetLinux().GetSecurityContext()
	// Set container username. This could only be done by containerd, because it needs
	// access to the container rootfs. Pass user name to containerd, and let it overwrite
	// the spec for us.
	userstr, err := generateUserString(
		securityContext.GetRunAsUsername(),
		securityContext.GetRunAsUser(),
		securityContext.GetRunAsGroup())
	if err != nil {
		return nil, fmt.Errorf("failed to generate user string: %w", err)
	}
	if userstr == "" {
		// Lastly, since no user override was passed via CRI try to set via OCI
		// Image
		userstr = imageConfig.User
	}
	if userstr != "" {
		specOpts = append(specOpts, over_oci.WithUser(userstr))
	}

	userstr = "0" // runtime default
	if securityContext.GetRunAsUsername() != "" {
		userstr = securityContext.GetRunAsUsername()
	} else if securityContext.GetRunAsUser() != nil {
		userstr = strconv.FormatInt(securityContext.GetRunAsUser().GetValue(), 10)
	} else if imageConfig.User != "" {
		userstr, _, _ = strings.Cut(imageConfig.User, ":")
	}
	specOpts = append(specOpts, customopts.WithAdditionalGIDs(userstr),
		customopts.WithSupplementalGroups(securityContext.GetSupplementalGroups()))

	asp := securityContext.GetApparmor()
	if asp == nil {
		asp, err = generateApparmorSecurityProfile(securityContext.GetApparmorProfile()) //nolint:staticcheck // Deprecated but we don't want to remove yet
		if err != nil {
			return nil, fmt.Errorf("failed to generate apparmor spec opts: %w", err)
		}
	}
	apparmorSpecOpts, err := generateApparmorSpecOpts(
		asp,
		securityContext.GetPrivileged(),
		c.apparmorEnabled())
	if err != nil {
		return nil, fmt.Errorf("failed to generate apparmor spec opts: %w", err)
	}
	if apparmorSpecOpts != nil {
		specOpts = append(specOpts, apparmorSpecOpts)
	}

	ssp := securityContext.GetSeccomp()
	if ssp == nil {
		ssp, err = generateSeccompSecurityProfile(
			securityContext.GetSeccompProfilePath(), //nolint:staticcheck // Deprecated but we don't want to remove yet
			c.config.UnsetSeccompProfile)
		if err != nil {
			return nil, fmt.Errorf("failed to generate seccomp spec opts: %w", err)
		}
	}
	seccompSpecOpts, err := c.generateSeccompSpecOpts(
		ssp,
		securityContext.GetPrivileged(),
		c.seccompEnabled())
	if err != nil {
		return nil, fmt.Errorf("failed to generate seccomp spec opts: %w", err)
	}
	if seccompSpecOpts != nil {
		specOpts = append(specOpts, seccompSpecOpts)
	}
	if c.config.EnableCDI {
		specOpts = append(specOpts, customopts.WithCDI(config.Annotations, config.CDIDevices))
	}
	return specOpts, nil
}

func generateSeccompSecurityProfile(profilePath string, unsetProfilePath string) (*runtime.SecurityProfile, error) {
	if profilePath != "" {
		return generateSecurityProfile(profilePath)
	}
	if unsetProfilePath != "" {
		return generateSecurityProfile(unsetProfilePath)
	}
	return nil, nil
}
func generateApparmorSecurityProfile(profilePath string) (*runtime.SecurityProfile, error) {
	if profilePath != "" {
		return generateSecurityProfile(profilePath)
	}
	return nil, nil
}

func generateSecurityProfile(profilePath string) (*runtime.SecurityProfile, error) {
	switch profilePath {
	case runtimeDefault, dockerDefault, "":
		return &runtime.SecurityProfile{
			ProfileType: runtime.SecurityProfile_RuntimeDefault,
		}, nil
	case unconfinedProfile:
		return &runtime.SecurityProfile{
			ProfileType: runtime.SecurityProfile_Unconfined,
		}, nil
	default:
		// Require and Trim default profile name prefix
		if !strings.HasPrefix(profilePath, profileNamePrefix) {
			return nil, fmt.Errorf("invalid profile %q", profilePath)
		}
		return &runtime.SecurityProfile{
			ProfileType:  runtime.SecurityProfile_Localhost,
			LocalhostRef: strings.TrimPrefix(profilePath, profileNamePrefix),
		}, nil
	}
}

// generateSeccompSpecOpts generates containerd SpecOpts for seccomp.
func (c *criService) generateSeccompSpecOpts(sp *runtime.SecurityProfile, privileged, seccompEnabled bool) (over_oci.SpecOpts, error) {
	if privileged {
		// Do not set seccomp profile when container is privileged
		return nil, nil
	}
	if !seccompEnabled {
		if sp != nil {
			if sp.ProfileType != runtime.SecurityProfile_Unconfined {
				return nil, errors.New("seccomp is not supported")
			}
		}
		return nil, nil
	}

	if sp == nil {
		return nil, nil
	}

	if sp.ProfileType != runtime.SecurityProfile_Localhost && sp.LocalhostRef != "" {
		return nil, errors.New("seccomp config invalid LocalhostRef must only be set if ProfileType is Localhost")
	}
	switch sp.ProfileType {
	case runtime.SecurityProfile_Unconfined:
		// Do not set seccomp profile.
		return nil, nil
	case runtime.SecurityProfile_RuntimeDefault:
		return seccomp.WithDefaultProfile(), nil
	case runtime.SecurityProfile_Localhost:
		// trimming the localhost/ prefix just in case even though it should not
		// be necessary with the new SecurityProfile struct
		return seccomp.WithProfile(strings.TrimPrefix(sp.LocalhostRef, profileNamePrefix)), nil
	default:
		return nil, errors.New("seccomp unknown ProfileType")
	}
}

// generateApparmorSpecOpts generates containerd SpecOpts for apparmor.
func generateApparmorSpecOpts(sp *runtime.SecurityProfile, privileged, apparmorEnabled bool) (over_oci.SpecOpts, error) {
	if !apparmorEnabled {
		// Should fail loudly if user try to specify apparmor profile
		// but we don't support it.
		if sp != nil {
			if sp.ProfileType != runtime.SecurityProfile_Unconfined {
				return nil, errors.New("apparmor is not supported")
			}
		}
		return nil, nil
	}

	if sp == nil {
		// Based on kubernetes#51746, default apparmor profile should be applied
		// for when apparmor is not specified.
		sp, _ = generateSecurityProfile("")
	}

	if sp.ProfileType != runtime.SecurityProfile_Localhost && sp.LocalhostRef != "" {
		return nil, errors.New("apparmor config invalid LocalhostRef must only be set if ProfileType is Localhost")
	}

	switch sp.ProfileType {
	case runtime.SecurityProfile_Unconfined:
		// Do not set apparmor profile.
		return nil, nil
	case runtime.SecurityProfile_RuntimeDefault:
		if privileged {
			// Do not set apparmor profile when container is privileged
			return nil, nil
		}
		// TODO (mikebrow): delete created apparmor default profile
		return apparmor.WithDefaultProfile(appArmorDefaultProfileName), nil
	case runtime.SecurityProfile_Localhost:
		// trimming the localhost/ prefix just in case even through it should not
		// be necessary with the new SecurityProfile struct
		appArmorProfile := strings.TrimPrefix(sp.LocalhostRef, profileNamePrefix)
		if profileExists, err := appArmorProfileExists(appArmorProfile); !profileExists {
			if err != nil {
				return nil, fmt.Errorf("failed to generate apparmor spec opts: %w", err)
			}
			return nil, fmt.Errorf("apparmor profile not found %s", appArmorProfile)
		}
		return apparmor.WithProfile(appArmorProfile), nil
	default:
		return nil, errors.New("apparmor unknown ProfileType")
	}
}

// appArmorProfileExists scans apparmor/profiles for the requested profile
func appArmorProfileExists(profile string) (bool, error) {
	if profile == "" {
		return false, errors.New("nil apparmor profile is not supported")
	}
	profiles, err := os.Open("/sys/kernel/security/apparmor/profiles")
	if err != nil {
		return false, err
	}
	defer profiles.Close()

	rbuff := bufio.NewReader(profiles)
	for {
		line, err := rbuff.ReadString('\n')
		switch err {
		case nil:
			if strings.HasPrefix(line, profile+" (") {
				return true, nil
			}
		case io.EOF:
			return false, nil
		default:
			return false, err
		}
	}
}

// generateUserString generates valid user string based on OCI Image Spec
// v1.0.0.
//
// CRI defines that the following combinations are valid:
//
// (none) -> ""
// username -> username
// username, uid -> username
// username, uid, gid -> username:gid
// username, gid -> username:gid
// uid -> uid
// uid, gid -> uid:gid
// gid -> error
//
// TODO(random-liu): Add group name support in CRI.
func generateUserString(username string, uid, gid *runtime.Int64Value) (string, error) {
	var userstr, groupstr string
	if uid != nil {
		userstr = strconv.FormatInt(uid.GetValue(), 10)
	}
	if username != "" {
		userstr = username
	}
	if gid != nil {
		groupstr = strconv.FormatInt(gid.GetValue(), 10)
	}
	if userstr == "" {
		if groupstr != "" {
			return "", fmt.Errorf("user group %q is specified without user", groupstr)
		}
		return "", nil
	}
	if groupstr != "" {
		userstr = userstr + ":" + groupstr
	}
	return userstr, nil
}

// snapshotterOpts returns any Linux specific snapshotter options for the rootfs snapshot
func snapshotterOpts(snapshotterName string, config *runtime.ContainerConfig) ([]snapshots.Opt, error) {
	nsOpts := config.GetLinux().GetSecurityContext().GetNamespaceOptions()
	return snapshotterRemapOpts(nsOpts)
}
