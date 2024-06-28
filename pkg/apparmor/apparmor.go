package apparmor

// HostSupports returns true if apparmor is enabled for the host:
//   - On Linux returns true if apparmor is enabled, apparmor_parser is
//     present, and if we are not running docker-in-docker.
//   - On non-Linux returns false.
//
// This is derived from libcontainer/apparmor.IsEnabled(), with the addition
// of checks for apparmor_parser to be present and docker-in-docker.
func HostSupports() bool {
	return hostSupports()
}
