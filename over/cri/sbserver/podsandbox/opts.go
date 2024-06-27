package podsandbox

import (
	"context"
	"demo/containerd"
	"demo/others/over/nri_extend"
	v1 "demo/others/over/nri_extend/types/v1"
	"demo/over/log"
)

// WithNRISandboxDelete calls delete for a sandbox'd task
func WithNRISandboxDelete(sandboxID string) containerd.ProcessDeleteOpts {
	return func(ctx context.Context, p containerd.Process) error {
		task, ok := p.(containerd.Task)
		if !ok {
			return nil
		}
		nric, err := nri_extend.New()
		if err != nil {
			log.G(ctx).WithError(err).Error("unable to create nri client")
			return nil
		}
		if nric == nil {
			return nil
		}
		sb := &nri_extend.Sandbox{
			ID: sandboxID,
		}
		if _, err := nric.InvokeWithSandbox(ctx, task, v1.Delete, sb); err != nil {
			log.G(ctx).WithError(err).Errorf("Failed to delete nri for %q", task.ID())
		}
		return nil
	}
}
