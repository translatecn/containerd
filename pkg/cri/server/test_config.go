package server

import (
	criconfig "demo/config/cri"
)

const (
	testRootDir  = "/test/root"
	testStateDir = "/test/state"
	// Use an image id as test sandbox image to avoid image name resolve.
	// TODO(random-liu): Change this to image name after we have complete image
	// management unit test framework.
	testSandboxImage = "sha256:c75bebcdd211f41b3a460c7bf82970ed6c75acaab9cd4c9a4e125b03ca113798"
	testImageFSPath  = "/test/image/fs/path"
)

var testConfig = criconfig.Config{
	RootDir:  testRootDir,
	StateDir: testStateDir,
	PluginConfig: criconfig.PluginConfig{
		SandboxImage:                     testSandboxImage,
		TolerateMissingHugetlbController: true,
	},
}
