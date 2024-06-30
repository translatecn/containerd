package runc

import (
	"context"
	"demo/pkg/drop"
	"demo/pkg/write"
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

	write.AppendRunLog("⚛️⚛️⚛️⚛️⚛️⚛️⚛️⚛️⚛️⚛️", "")
	write.AppendRunLog("========> runc  ENV: ", drop.DropEnv(cmd.Env))
	write.AppendRunLog("========> runc  Args: ", cmd.Args)
	write.AppendRunLog("========> runc  Path: ", cmd.Path)
	write.AppendRunLog("========> runc  Process: ", cmd.Process)
	write.AppendRunLog("========> runc  Dir: ", cmd.Dir)

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
