package labels

const (
	// criContainerdPrefix is common prefix for cri-containerd
	criContainerdPrefix = "io.cri-containerd"
	// ImageLabelKey is the label key indicating the image is managed by cri plugin.
	ImageLabelKey = criContainerdPrefix + ".image"
	// ImageLabelValue is the label value indicating the image is managed by cri plugin.
	ImageLabelValue = "managed"
	// PinnedImageLabelKey is the label value indicating the image is pinned.
	PinnedImageLabelKey = criContainerdPrefix + ".pinned"
	// PinnedImageLabelValue is the label value indicating the image is pinned.
	PinnedImageLabelValue = "pinned"
)
