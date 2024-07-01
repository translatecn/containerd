package containerd

import (
	diffapi "demo/pkg/api/services/diff/v1"
	"demo/pkg/diff"
	"demo/pkg/diff/proxy"
)

// DiffService handles the computation and application of diffs
type DiffService interface {
	diff.Comparer
	diff.Applier
}

// NewDiffServiceFromClient returns a new diff service which communicates
// over a GRPC connection.
func NewDiffServiceFromClient(client diffapi.DiffClient) DiffService {
	return proxy.NewDiffApplier(client).(DiffService)
}
