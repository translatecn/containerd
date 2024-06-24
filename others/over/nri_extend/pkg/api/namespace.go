package api

import (
	rspec "github.com/opencontainers/runtime-spec/specs-go"
)

// FromOCILinuxNamespaces returns a namespace slice from an OCI runtime Spec.
func FromOCILinuxNamespaces(o []rspec.LinuxNamespace) []*LinuxNamespace {
	var namespaces []*LinuxNamespace
	for _, ns := range o {
		namespaces = append(namespaces, &LinuxNamespace{
			Type: string(ns.Type),
			Path: ns.Path,
		})
	}
	return namespaces
}
