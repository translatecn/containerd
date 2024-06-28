package logging

import (
	"context"
	"io"
)

// Config of the container logs
type Config struct {
	ID        string
	Namespace string
	Stdout    io.Reader
	Stderr    io.Reader
}

// LoggerFunc is implemented by custom v2 logging binaries.
//
// ready should be called when the logging binary finishes its setup and the container can be started.
//
// An example implementation of LoggerFunc: https://github.com/containerd/tree/main/runtime/v2#logging
type LoggerFunc func(ctx context.Context, cfg *Config, ready func() error) error
