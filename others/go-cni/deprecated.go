package cni

import types100 "demo/others/cni/pkg/types/100"

// Deprecated: use cni.Opt instead
type CNIOpt = Opt //revive:disable // type name will be used as cni.CNIOpt by other packages, and that stutters

// Deprecated: use cni.Result instead
type CNIResult = Result //revive:disable // type name will be used as cni.CNIResult by other packages, and that stutters

// GetCNIResultFromResults creates a Result from the given slice of types100.Result,
// adding structured data containing the interface configuration for each of the
// interfaces created in the namespace. It returns an error if validation of
// results fails, or if a network could not be found.
// Deprecated: do not use
func (c *libcni) GetCNIResultFromResults(results []*types100.Result) (*Result, error) {
	return c.createResult(results)
}
