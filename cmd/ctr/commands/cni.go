package commands

import (
	"context"
	"demo/pkg/namespaces"
	"demo/pkg/typeurl/v2"
	"fmt"

	"demo/containerd"
)

func init() {
	typeurl.Register(&NetworkMetaData{},
		"demo/cmd/ctr/commands", "NetworkMetaData")
}

const (

	// CtrCniMetadataExtension is an extension name that identify metadata of container in CreateContainerRequest
	CtrCniMetadataExtension = "ctr.cni-containerd.metadata"
)

// ctr pass cni network metadata to containerd if ctr run use option of --cni
type NetworkMetaData struct {
	EnableCni bool
}

func FullID(ctx context.Context, c containerd.Container) string {
	id := c.ID()
	ns, ok := namespaces.Namespace(ctx)
	if !ok {
		return id
	}
	return fmt.Sprintf("%s-%s", ns, id)
}
