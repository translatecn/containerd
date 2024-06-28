// Package providing utilies to help cleanup
package cleanup

import (
	"context"
	"time"
)

type clearCancel struct {
	context.Context
}

func (cc clearCancel) Deadline() (deadline time.Time, ok bool) {
	return
}

func (cc clearCancel) Done() <-chan struct{} {
	return nil
}

func (cc clearCancel) Err() error {
	return nil
}

// Background creates a new context which clears out the parent errors
func Background(ctx context.Context) context.Context {
	return clearCancel{ctx}
}

// Do runs the provided function with a context in which the
// errors are cleared out and will timeout after 10 seconds.
func Do(ctx context.Context, do func(context.Context)) {
	ctx, cancel := context.WithTimeout(clearCancel{ctx}, 10*time.Second)
	do(ctx)
	cancel()
}
