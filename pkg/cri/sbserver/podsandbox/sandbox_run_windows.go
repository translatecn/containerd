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

package podsandbox

import (
	"fmt"
	"strconv"

	"demo/containerd"
	"demo/over/oci"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	runtime "k8s.io/cri-api/pkg/apis/runtime/v1"

	"demo/pkg/cri/annotations"
	customopts "demo/pkg/cri/opts"
)

func (c *Controller) sandboxContainerSpec(id string, config *runtime.PodSandboxConfig,
	imageConfig *imagespec.ImageConfig, nsPath string, runtimePodAnnotations []string) (*runtimespec.Spec, error) {
	// Creates a spec Generator with the default spec.
	specOpts := []over_oci.SpecOpts{
		over_oci.WithEnv(imageConfig.Env),
		over_oci.WithHostname(config.GetHostname()),
	}
	if imageConfig.WorkingDir != "" {
		specOpts = append(specOpts, over_oci.WithProcessCwd(imageConfig.WorkingDir))
	}

	if len(imageConfig.Entrypoint) == 0 && len(imageConfig.Cmd) == 0 {
		// Pause image must have entrypoint or cmd.
		return nil, fmt.Errorf("invalid empty entrypoint and cmd in image config %+v", imageConfig)
	}
	specOpts = append(specOpts, over_oci.WithProcessArgs(append(imageConfig.Entrypoint, imageConfig.Cmd...)...))

	specOpts = append(specOpts,
		// Clear the root location since hcsshim expects it.
		// NOTE: readonly rootfs doesn't work on windows.
		customopts.WithoutRoot,
		over_oci.WithWindowsNetworkNamespace(nsPath),
	)

	specOpts = append(specOpts, customopts.WithWindowsDefaultSandboxShares)

	// Start with the image config user and override below if RunAsUsername is not "".
	username := imageConfig.User

	runAsUser := config.GetWindows().GetSecurityContext().GetRunAsUsername()
	if runAsUser != "" {
		username = runAsUser
	}

	cs := config.GetWindows().GetSecurityContext().GetCredentialSpec()
	if cs != "" {
		specOpts = append(specOpts, customopts.WithWindowsCredentialSpec(cs))
	}

	// There really isn't a good Windows way to verify that the username is available in the
	// image as early as here like there is for Linux. Later on in the stack hcsshim
	// will handle the behavior of erroring out if the user isn't available in the image
	// when trying to run the init process.
	specOpts = append(specOpts, over_oci.WithUser(username))

	for pKey, pValue := range getPassthroughAnnotations(config.Annotations,
		runtimePodAnnotations) {
		specOpts = append(specOpts, customopts.WithAnnotation(pKey, pValue))
	}

	specOpts = append(specOpts, customopts.WithAnnotation(annotations.WindowsHostProcess, strconv.FormatBool(config.GetWindows().GetSecurityContext().GetHostProcess())))
	specOpts = append(specOpts,
		annotations.DefaultCRIAnnotations(id, "", "", config, true)...,
	)

	return c.runtimeSpec(id, "", specOpts...)
}

// No sandbox container spec options for windows yet.
func (c *Controller) sandboxContainerSpecOpts(config *runtime.PodSandboxConfig, imageConfig *imagespec.ImageConfig) ([]over_oci.SpecOpts, error) {
	return nil, nil
}

// No sandbox files needed for windows.
func (c *Controller) setupSandboxFiles(id string, config *runtime.PodSandboxConfig) error {
	return nil
}

// No sandbox files needed for windows.
func (c *Controller) cleanupSandboxFiles(id string, config *runtime.PodSandboxConfig) error {
	return nil
}

// No task options needed for windows.
func (c *Controller) taskOpts(runtimeType string) []containerd.NewTaskOpts {
	return nil
}
