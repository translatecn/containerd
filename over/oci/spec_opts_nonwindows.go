package oci

import (
	"context"

	"demo/over/containers"
)

// WithDefaultPathEnv sets the $PATH environment variable to the
// default PATH defined in this package.
func WithDefaultPathEnv(_ context.Context, _ Client, _ *containers.Container, s *Spec) error {
	s.Process.Env = replaceOrAppendEnvValues(s.Process.Env, defaultUnixEnv)
	return nil
}
