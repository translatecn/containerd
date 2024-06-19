package runc

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"runtime"
)

// NewPipeIO creates pipe pairs to be used with runc
func NewPipeIO(uid, gid int, opts ...IOOpt) (i IO, err error) {
	option := defaultIOOption()
	for _, o := range opts {
		o(option)
	}
	var (
		pipes                 []*pipe
		stdin, stdout, stderr *pipe
	)
	// cleanup in case of an error
	defer func() {
		if err != nil {
			for _, p := range pipes {
				p.Close()
			}
		}
	}()
	if option.OpenStdin {
		if stdin, err = newPipe(); err != nil {
			return nil, err
		}
		pipes = append(pipes, stdin)
		if err = unix.Fchown(int(stdin.r.Fd()), uid, gid); err != nil {
			// TODO: revert with proper darwin solution, skipping for now
			// as darwin chown is returning EINVAL on anonymous pipe
			if runtime.GOOS == "darwin" {
				logrus.WithError(err).Debug("failed to chown stdin, ignored")
			} else {
				return nil, errors.Wrap(err, "failed to chown stdin")
			}
		}
	}
	if option.OpenStdout {
		if stdout, err = newPipe(); err != nil {
			return nil, err
		}
		pipes = append(pipes, stdout)
		if err = unix.Fchown(int(stdout.w.Fd()), uid, gid); err != nil {
			// TODO: revert with proper darwin solution, skipping for now
			// as darwin chown is returning EINVAL on anonymous pipe
			if runtime.GOOS == "darwin" {
				logrus.WithError(err).Debug("failed to chown stdout, ignored")
			} else {
				return nil, errors.Wrap(err, "failed to chown stdout")
			}
		}
	}
	if option.OpenStderr {
		if stderr, err = newPipe(); err != nil {
			return nil, err
		}
		pipes = append(pipes, stderr)
		if err = unix.Fchown(int(stderr.w.Fd()), uid, gid); err != nil {
			// TODO: revert with proper darwin solution, skipping for now
			// as darwin chown is returning EINVAL on anonymous pipe
			if runtime.GOOS == "darwin" {
				logrus.WithError(err).Debug("failed to chown stderr, ignored")
			} else {
				return nil, errors.Wrap(err, "failed to chown stderr")
			}
		}
	}
	return &pipeIO{
		in:  stdin,
		out: stdout,
		err: stderr,
	}, nil
}
