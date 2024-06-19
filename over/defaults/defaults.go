package defaults

const (
	// DefaultMaxRecvMsgSize defines the default maximum message size for
	// receiving protobufs passed over the GRPC API.
	DefaultMaxRecvMsgSize = 16 << 20
	// DefaultMaxSendMsgSize defines the default maximum message size for
	// sending protobufs passed over the GRPC API.
	DefaultMaxSendMsgSize = 16 << 20
	// DefaultRuntimeNSLabel defines the namespace label to check for the
	// default runtime
	DefaultRuntimeNSLabel = "containerd.io/defaults/runtime"
	// DefaultSnapshotterNSLabel defines the namespace label to check for the
	// default snapshotter
	DefaultSnapshotterNSLabel = "containerd.io/defaults/snapshotter"
)
