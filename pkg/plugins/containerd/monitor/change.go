package monitor

import (
	"context"
	"demo/pkg/runtime/restart"
	"fmt"
	"net/url"
	"strconv"
	"syscall"

	"github.com/sirupsen/logrus"

	"demo/containerd"
	"demo/pkg/cio"
)

type stopChange struct {
	container containerd.Container
}

func (s *stopChange) apply(ctx context.Context, client *containerd.Client) error {
	return killTask(ctx, s.container)
}

type startChange struct {
	container containerd.Container
	logURI    string
	count     int

	// Deprecated(in release 1.5): but recognized now, prefer to use logURI
	logPath string
	// logPathCallback is a func invoked if logPath is defined, used for emitting deprecation warnings
	logPathCallback func()
}

func (s *startChange) apply(ctx context.Context, client *containerd.Client) error {
	log := cio.NullIO

	if s.logURI != "" {
		uri, err := url.Parse(s.logURI)
		if err != nil {
			return fmt.Errorf("failed to parse %v into url: %w", s.logURI, err)
		}
		log = cio.LogURI(uri)
	} else if s.logPath != "" {
		log = cio.LogFile(s.logPath)
	}
	if s.logPath != "" && s.logPathCallback != nil {
		logrus.WithField("container", s.container.ID()).WithField(restart.LogPathLabel, s.logPath).
			Warnf("%q label is deprecated in containerd v1.5 and will be removed in containerd v2.0. Use %q instead.", restart.LogPathLabel, restart.LogURILabel)
		s.logPathCallback()
	}

	if s.logURI != "" && s.logPath != "" {
		logrus.Warnf("LogPathLabel=%v has been deprecated, using LogURILabel=%v",
			s.logPath, s.logURI)
	}

	if s.count > 0 {
		labels := map[string]string{
			restart.CountLabel: strconv.Itoa(s.count),
		}
		opt := containerd.WithAdditionalContainerLabels(labels)
		if err := s.container.Update(ctx, containerd.UpdateContainerOpts(opt)); err != nil {
			return err
		}
	}
	killTask(ctx, s.container)
	task, err := s.container.NewTask(ctx, log)
	if err != nil {
		return err
	}
	return task.Start(ctx)
}

func killTask(ctx context.Context, container containerd.Container) error {
	task, err := container.Task(ctx, nil)
	if err == nil {
		wait, err := task.Wait(ctx)
		if err != nil {
			if _, derr := task.Delete(ctx); derr == nil {
				return nil
			}
			return err
		}
		if err := task.Kill(ctx, syscall.SIGKILL, containerd.WithKillAll); err != nil {
			if _, derr := task.Delete(ctx); derr == nil {
				return nil
			}
			return err
		}
		<-wait
		if _, err := task.Delete(ctx); err != nil {
			return err
		}
	}
	return nil
}
