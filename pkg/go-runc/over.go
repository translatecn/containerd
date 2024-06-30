package runc

import (
	"context"
	"demo/pkg/sys/reaper"
	"fmt"
	"strconv"
)

func (r *Runc) args() (out []string) {
	if r.Root != "" {
		out = append(out, "--root", r.Root)
	}
	if r.Debug {
		out = append(out, "--debug")
	}
	if r.Log != "" {
		out = append(out, "--log", r.Log)
	}
	if r.LogFormat != none {
		out = append(out, "--log-format", string(r.LogFormat))
	}
	if r.Criu != "" {
		out = append(out, "--criu", r.Criu)
	}
	if r.SystemdCgroup {
		out = append(out, "--systemd-cgroup")
	}
	if r.Rootless != nil {
		// nil stands for "auto" (differs from explicit "false")
		out = append(out, "--rootless="+strconv.FormatBool(*r.Rootless))
	}
	return out
}

func (r *Runc) Create(context context.Context, id, bundle string, opts *CreateOpts) error {
	args := []string{"create", "--bundle", bundle}
	if opts != nil {
		oargs, err := opts.args()
		if err != nil {
			return err
		}
		args = append(args, oargs...)
	}
	cmd := r.command(context, append(args, id)...)

	if opts != nil && opts.IO != nil {
		opts.Set(cmd)
	}
	cmd.ExtraFiles = opts.ExtraFiles

	if cmd.Stdout == nil && cmd.Stderr == nil {
		data, err := cmdOutput(cmd, true, nil)
		defer putBuf(data)
		if err != nil {
			return fmt.Errorf("%s: %s", err, data.String())
		}
		return nil
	}
	ec, err := reaper.Default.Start(cmd)
	if err != nil {
		return err
	}
	if opts != nil && opts.IO != nil {
		if c, ok := opts.IO.(StartCloser); ok {
			if err := c.CloseAfterStart(); err != nil {
				return err
			}
		}
	}
	status, err := reaper.Default.Wait(cmd, ec)
	if err == nil && status != 0 {
		err = fmt.Errorf("%s did not terminate successfully: %w", cmd.Args[0], &ExitError{status})
	}
	return err
}
