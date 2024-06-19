package seccomp

// IsEnabled checks whether seccomp support is enabled. On Linux, it returns
// true if the kernel has been configured to support seccomp (kernel options
// CONFIG_SECCOMP and CONFIG_SECCOMP_FILTER are set). On non-Linux, it always
// returns false.
func IsEnabled() bool {
	return isEnabled()
}
