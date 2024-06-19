package containerd

// InstallOpts configures binary installs
type InstallOpts func(*InstallConfig)

// InstallConfig sets the binary install configuration
type InstallConfig struct {
	// Libs installs libs from the image
	Libs bool
	// Replace will overwrite existing binaries or libs in the opt directory
	Replace bool
	// Path to install libs and binaries to
	Path string
}

// WithInstallLibs installs libs from the image
func WithInstallLibs(c *InstallConfig) {
	c.Libs = true
}

// WithInstallReplace will replace existing files
func WithInstallReplace(c *InstallConfig) {
	c.Replace = true
}

// WithInstallPath sets the optional install path
func WithInstallPath(path string) InstallOpts {
	return func(c *InstallConfig) {
		c.Path = path
	}
}
