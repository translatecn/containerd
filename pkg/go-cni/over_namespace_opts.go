package cni

type NamespaceOpts func(s *Namespace) error

// WithCapabilityDNS adds support for dns
func WithCapabilityDNS(dns DNS) NamespaceOpts {
	return func(c *Namespace) error {
		c.capabilityArgs["dns"] = dns
		return nil
	}
}

// WithCapabilityCgroupPath passes in the cgroup path capability.
func WithCapabilityCgroupPath(cgroupPath string) NamespaceOpts {
	return func(c *Namespace) error {
		c.capabilityArgs["cgroupPath"] = cgroupPath
		return nil
	}
}

// WithCapability support well-known capabilities
// https://www.cni.dev/docs/conventions/#well-known-capabilities
func WithCapability(name string, capability interface{}) NamespaceOpts {
	return func(c *Namespace) error {
		c.capabilityArgs[name] = capability
		return nil
	}
}

// Args
func WithLabels(labels map[string]string) NamespaceOpts {
	return func(c *Namespace) error {
		for k, v := range labels {
			c.args[k] = v
		}
		return nil
	}
}

func WithCapabilityPortMap(portMapping []PortMapping) NamespaceOpts {
	return func(c *Namespace) error {
		c.capabilityArgs["portMappings"] = portMapping
		return nil
	}
}

func WithCapabilityBandWidth(bandWidth BandWidth) NamespaceOpts {
	return func(c *Namespace) error {
		c.capabilityArgs["bandwidth"] = bandWidth
		return nil
	}
}
