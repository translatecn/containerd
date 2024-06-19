package continuity

// Manifest provides the contents of a manifest. Users of this struct should
// not typically modify any fields directly.
type Manifest struct {
	// Resources specifies all the resources for a manifest in order by path.
	Resources []Resource
}

// BuildManifest creates the manifest for the given context
