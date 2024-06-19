package apparmor

import (
	"os"
	"sync"
)

var (
	appArmorSupported bool
	checkAppArmor     sync.Once
)

// hostSupports returns true if apparmor is enabled for the host, if
// apparmor_parser is enabled, and if we are not running docker-in-docker.
//
// This is derived from libcontainer/apparmor.IsEnabled(), with the addition
// of checks for apparmor_parser to be present and docker-in-docker.
func hostSupports() bool {
	checkAppArmor.Do(func() {
		// see https://demo/3rd_party/runc/blob/0d49470392206f40eaab3b2190a57fe7bb3df458/libcontainer/apparmor/apparmor_linux.go
		if _, err := os.Stat("/sys/kernel/security/apparmor"); err == nil && os.Getenv("container") == "" {
			if _, err = os.Stat("/sbin/apparmor_parser"); err == nil {
				buf, err := os.ReadFile("/sys/module/apparmor/parameters/enabled")
				appArmorSupported = err == nil && len(buf) > 1 && buf[0] == 'Y'
			}
		}
	})
	return appArmorSupported
}
