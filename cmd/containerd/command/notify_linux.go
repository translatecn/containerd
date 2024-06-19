package command

import (
	"context"
	"demo/over/log"
	sd "github.com/coreos/go-systemd/v22/daemon"
)

// notifyReady notifies systemd that the daemon is ready to serve requests
func notifyReady(ctx context.Context) error {
	return sdNotify(ctx, sd.SdNotifyReady)
}

// notifyStopping notifies systemd that the daemon is about to be stopped
func notifyStopping(ctx context.Context) error {
	return sdNotify(ctx, sd.SdNotifyStopping)
}

func sdNotify(ctx context.Context, state string) error {
	notified, err := sd.SdNotify(false, state)
	entry := log.G(ctx)
	if err != nil {
		entry = entry.WithError(err)
	}
	entry.WithField("notified", notified).
		WithField("state", state).
		Debug("sd notification")

	return err
}
