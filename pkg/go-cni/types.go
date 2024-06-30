package cni

const (
	CNIPluginName     = "cni"
	DefaultMaxConfNum = 1
	DefaultPrefix     = "eth"
)

type config struct {
	pluginDirs       []string
	pluginConfDir    string
	pluginMaxConfNum int
	prefix           string
}

type PortMapping struct {
	HostPort      int32
	ContainerPort int32
	Protocol      string
	HostIP        string
}

type IPRanges struct {
	Subnet     string
	RangeStart string
	RangeEnd   string
	Gateway    string
}

// BandWidth defines the ingress/egress rate and burst limits
type BandWidth struct {
	IngressRate  uint64
	IngressBurst uint64
	EgressRate   uint64
	EgressBurst  uint64
}

// DNS defines the dns config
type DNS struct {
	// List of DNS servers of the cluster.
	Servers []string
	// List of DNS search domains of the cluster.
	Searches []string
	// List of DNS options.
	Options []string
}
