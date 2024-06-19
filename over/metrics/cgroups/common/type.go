package common

import (
	"context"
	"demo/over/protobuf/types"
)

// Statable type that returns cgroup metrics
type Statable interface {
	ID() string
	Namespace() string
	Stats(context.Context) (*types.Any, error)
}
