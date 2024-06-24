package multiplex

const (
	// PluginServiceConn is the mux connection ID for NRI plugin services.
	PluginServiceConn ConnID = iota + 1
	// RuntimeServiceConn is the mux connection ID for NRI runtime services.
	RuntimeServiceConn
)
