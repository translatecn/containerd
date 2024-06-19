/*
Copyright 2019 The Kubernetes Authors.

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

package validate

import (
	"runtime"

	"github.com/kubernetes-sigs/cri-tools/pkg/framework"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

// Container test constants

var (
	echoHelloCmd      []string
	sleepCmd          []string
	checkSleepCmd     []string
	shellCmd          []string
	pauseCmd          []string
	logDefaultCmd     []string
	loopLogDefaultCmd []string
	echoHelloOutput   string
	checkPathCmd      func(string) []string

	// Linux defaults
	echoHelloLinuxCmd      = []string{"echo", "-n", "hello"}
	sleepLinuxCmd          = []string{"sleep", "4321"}
	checkSleepLinuxCmd     = []string{"sh", "-c", "pgrep sleep || true"}
	shellLinuxCmd          = []string{"/bin/sh"}
	pauseLinuxCmd          = []string{"sh", "-c", "top"}
	logDefaultLinuxCmd     = []string{"echo", defaultLog}
	loopLogDefaultLinuxCmd = []string{"sh", "-c", "while true; do echo " + defaultLog + "; sleep 1; done"}
	echoHelloLinuxOutput   = "hello"
	checkPathLinuxCmd      = func(path string) []string { return []string{"ls", "-A", path} }
)

var _ = framework.AddBeforeSuiteCallback(func() {
	echoHelloCmd = echoHelloLinuxCmd
	sleepCmd = sleepLinuxCmd
	checkSleepCmd = checkSleepLinuxCmd
	shellCmd = shellLinuxCmd
	pauseCmd = pauseLinuxCmd
	logDefaultCmd = logDefaultLinuxCmd
	loopLogDefaultCmd = loopLogDefaultLinuxCmd
	echoHelloOutput = echoHelloLinuxOutput
	checkPathCmd = checkPathLinuxCmd
})

// Image test constants

const (
	registry = "gcr.io/k8s-staging-cri-tools/"

	testImageUserUID           = registry + "test-image-user-uid"
	imageUserUID               = int64(1002)
	testImageUserUsername      = registry + "test-image-user-username"
	imageUserUsername          = "www-data"
	testImageUserUIDGroup      = registry + "test-image-user-uid-group"
	imageUserUIDGroup          = int64(1003)
	testImageUserUsernameGroup = registry + "test-image-user-username-group"
	imageUserUsernameGroup     = "www-data"
	testImagePreDefinedGroup   = registry + "test-image-predefined-group"
	imagePredefinedGroupUID    = int64(1000)
	imagePredefinedGroupGID    = int64(50000)

	// Linux defaults
	testLinuxImageWithoutTag        = registry + "test-image-latest"
	testLinuxImageWithTag           = registry + "test-image-tag:test"
	testLinuxImageWithDigest        = registry + "test-image-digest@sha256:9700f9a2f5bf2c45f2f605a0bd3bce7cf37420ec9d3ed50ac2758413308766bf"
	testLinuxImageWithAllReferences = registry + "test-image-tag:all"
)

var (
	// image reference without tag
	testImageWithoutTag string

	// name-tagged reference for test image
	testImageWithTag string

	// digested reference for test image
	testImageWithDigest string

	// image used to test all kinds of references.
	testImageWithAllReferences string

	// image list where different tags refer to different images
	testDifferentTagDifferentImageList []string

	// image list where different tags refer to the same image
	testDifferentTagSameImageList []string

	// pod sandbox to use when pulling images
	testImagePodSandbox *runtimeapi.PodSandboxConfig

	// Linux defaults
	testLinuxDifferentTagDifferentImageList = []string{
		registry + "test-image-1:latest",
		registry + "test-image-2:latest",
		registry + "test-image-3:latest",
	}
	testLinuxDifferentTagSameImageList = []string{
		registry + "test-image-tags:1",
		registry + "test-image-tags:2",
		registry + "test-image-tags:3",
	}
)

var _ = framework.AddBeforeSuiteCallback(func() {
	testImageWithoutTag = testLinuxImageWithoutTag
	testImageWithTag = testLinuxImageWithTag
	testImageWithDigest = testLinuxImageWithDigest
	testImageWithAllReferences = testLinuxImageWithAllReferences
	testDifferentTagDifferentImageList = testLinuxDifferentTagDifferentImageList
	testDifferentTagSameImageList = testLinuxDifferentTagSameImageList
	testImagePodSandbox = &runtimeapi.PodSandboxConfig{
		Labels: framework.DefaultPodLabels,
	}
})

// Networking test constants

const (
	resolvConfigPath              = "/etc/resolv.conf"
	defaultDNSServer       string = "10.10.10.10"
	defaultDNSSearch       string = "google.com"
	defaultDNSOption       string = "ndots:8"
	webServerContainerPort int32  = 80
	// The following host ports must not be in-use when running the test.
	webServerHostPortForPortMapping        int32 = 12000
	webServerHostPortForPortForward        int32 = 12001
	webServerHostPortForHostNetPortFroward int32 = 12002
	// The port used in hostNetNginxImage (See images/hostnet-nginx/)
	webServerHostNetContainerPort int32 = 12003

	// Linux defaults
	webServerLinuxImage        = framework.DefaultRegistryE2ETestImagesPrefix + "nginx:1.14-2"
	hostNetWebServerLinuxImage = registry + "hostnet-nginx-" + runtime.GOARCH

	// Windows defaults
	webServerWindowsImage        = webServerLinuxImage
	hostNetWebServerWindowsImage = webServerLinuxImage
)

var (
	webServerImage        string
	hostNetWebServerImage string
	getDNSConfigCmd       []string
	getDNSConfigContent   []string
	getHostnameCmd        []string

	// Linux defaults
	getDNSConfigLinuxCmd     = []string{"cat", resolvConfigPath}
	getDNSConfigLinuxContent = []string{
		"nameserver " + defaultDNSServer,
		"search " + defaultDNSSearch,
		"options " + defaultDNSOption,
	}
	getHostnameLinuxCmd = []string{"hostname"}
)

var _ = framework.AddBeforeSuiteCallback(func() {
	webServerImage = webServerLinuxImage
	hostNetWebServerImage = hostNetWebServerLinuxImage
	getDNSConfigCmd = getDNSConfigLinuxCmd
	getDNSConfigContent = getDNSConfigLinuxContent
	getHostnameCmd = getHostnameLinuxCmd

	// Override the web server test image if an explicit one is provided:
	if framework.TestContext.TestImageList.WebServerTestImage != "" {
		webServerImage = framework.TestContext.TestImageList.WebServerTestImage
	}
})

// Streaming test constants

const (
	defaultStreamServerAddress string = "127.0.0.1:10250"
	defaultStreamServerScheme  string = "http"

	// Linux defaults
	attachEchoHelloLinuxOutput = "hello"
)

var (
	attachEchoHelloOutput string
)

var _ = framework.AddBeforeSuiteCallback(func() {
	attachEchoHelloOutput = attachEchoHelloLinuxOutput
})
