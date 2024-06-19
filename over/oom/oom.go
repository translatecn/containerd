package oom

import (
	"context"
)

// Watcher watches OOM events
type Watcher interface {
	Close() error
	Run(ctx context.Context)
	Add(id string, cg interface{}) error
}
