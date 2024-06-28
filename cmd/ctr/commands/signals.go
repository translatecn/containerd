package commands

import (
	gocontext "context"
	"os"
	"os/signal"
	"syscall"

	"demo/containerd"
	"demo/pkg/errdefs"
	"github.com/sirupsen/logrus"
)

type killer interface {
	Kill(gocontext.Context, syscall.Signal, ...containerd.KillOpts) error
}

// ForwardAllSignals forwards signals
func ForwardAllSignals(ctx gocontext.Context, task killer) chan os.Signal {
	sigc := make(chan os.Signal, 128)
	signal.Notify(sigc)
	go func() {
		for s := range sigc {
			if canIgnoreSignal(s) {
				logrus.Debugf("Ignoring signal %s", s)
				continue
			}
			logrus.Debug("forwarding signal ", s)
			if err := task.Kill(ctx, s.(syscall.Signal)); err != nil {
				if errdefs.IsNotFound(err) {
					logrus.WithError(err).Debugf("Not forwarding signal %s", s)
					return
				}
				logrus.WithError(err).Errorf("forward signal %s", s)
			}
		}
	}()
	return sigc
}

// StopCatch stops and closes a channel
func StopCatch(sigc chan os.Signal) {
	signal.Stop(sigc)
	close(sigc)
}
