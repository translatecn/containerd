package runc

import (
	"context"
	"demo/over/drop"
	"demo/over/log"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func (r *Runc) command(context context.Context, args ...string) *exec.Cmd {
	command := r.Command
	if command == "" {
		command = DefaultCommand
	}
	cmd := exec.CommandContext(context, command, append(r.args(), args...)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: r.Setpgid,
	}
	cmd.Env = filterEnv(os.Environ(), "NOTIFY_SOCKET") // NOTIFY_SOCKET introduces a special behavior in runc but should only be set if invoked from systemd
	if r.PdeathSignal != 0 {
		cmd.SysProcAttr.Pdeathsig = r.PdeathSignal
	}
	log.G(context).WithFields(log.Fields{"type": "runc"}).Errorln("========> runc  ENV: ", drop.DropEnv(cmd.Env))
	log.G(context).WithFields(log.Fields{"type": "runc"}).Errorln("========> runc  Args: ", cmd.Args)
	log.G(context).WithFields(log.Fields{"type": "runc"}).Errorln("========> runc  Path: ", cmd.Path)
	log.G(context).WithFields(log.Fields{"type": "runc"}).Errorln("========> runc  Process: ", cmd.Process)
	log.G(context).WithFields(log.Fields{"type": "runc"}).Errorln("========> runc  Dir: ", cmd.Dir)

	return cmd
}

func filterEnv(in []string, names ...string) []string {
	out := make([]string, 0, len(in))
loop0:
	for _, v := range in {
		for _, k := range names {
			if strings.HasPrefix(v, k+"=") {
				continue loop0
			}
		}
		out = append(out, v)
	}
	return out
}
