package command

import (
	"demo/pkg/plugins/containerd/content"
	"github.com/urfave/cli"
)

// serviceFlags returns an array of flags for configuring containerd to run
// as a service. Only relevant on Windows.
func serviceFlags() []cli.Flag {
	return nil
}

// applyPlatformFlags applies platform-specific flags.
func applyPlatformFlags(context *cli.Context) {
}

// registerUnregisterService is only relevant on Windows.
func registerUnregisterService(root string) (bool, error) {
	return false, nil
}

// launchService is only relevant on Windows.
func launchService(s *content.Server, done chan struct{}) error {
	return nil
}
