package command

import (
	"context"
	"demo/pkg/log"
	"demo/pkg/plugins/containerd/content"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

var handledSignals = []os.Signal{
	unix.SIGTERM,
	unix.SIGINT,
	unix.SIGUSR1,
	unix.SIGPIPE,
}

func handleSignals(ctx context.Context, signals chan os.Signal, serverC chan *content.Server, cancel func()) chan struct{} {
	done := make(chan struct{}, 1)
	go func() {
		var server *content.Server
		for {
			select {
			case s := <-serverC:
				server = s
			case s := <-signals:

				// Do not print message when dealing with SIGPIPE, which may cause
				// nested signals and consume lots of cpu bandwidth.
				if s == unix.SIGPIPE {
					continue
				}

				log.G(ctx).WithField("signal", s).Debug("received signal")
				switch s {
				case unix.SIGUSR1:
					dumpStacks(true)
				default:
					if err := notifyStopping(ctx); err != nil {
						log.G(ctx).WithError(err).Error("notify stopping failed")
					}

					cancel()
					if server != nil {
						server.Stop()
					}
					close(done)
					return
				}
			}
		}
	}()
	return done
}

func isLocalAddress(path string) bool {
	return filepath.IsAbs(path)
}
