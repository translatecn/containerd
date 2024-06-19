package labels

// LabelUncompressed is added to compressed layer contents.
// The value is digest of the uncompressed content.
const LabelUncompressed = "containerd.io/uncompressed"

// LabelSharedNamespace is added to a namespace to allow that namespaces
// contents to be shared.
const LabelSharedNamespace = "containerd.io/namespace.shareable"

// LabelDistributionSource is added to content to indicate its origin.
// e.g., "containerd.io/distribution.source.docker.io=library/redis"
const LabelDistributionSource = "containerd.io/distribution.source"
