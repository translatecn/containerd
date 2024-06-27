package sbserver

import (
	"context"
	"demo/over/cri/store/sandbox"
	"fmt"
	"time"

	runtime "demo/over/api/cri/v1"
)

// PodSandboxStatus returns the status of the PodSandbox.
func (c *CriService) PodSandboxStatus(ctx context.Context, r *runtime.PodSandboxStatusRequest) (*runtime.PodSandboxStatusResponse, error) {
	sandbox, err := c.sandboxStore.Get(r.GetPodSandboxId())
	if err != nil {
		return nil, fmt.Errorf("an error occurred when try to find sandbox: %w", err)
	}

	ip, additionalIPs, err := c.getIPs(sandbox)
	if err != nil {
		return nil, fmt.Errorf("failed to get sandbox ip: %w", err)
	}

	controller, err := c.getSandboxController(sandbox.Config, sandbox.RuntimeHandler)
	if err != nil {
		return nil, fmt.Errorf("failed to get sandbox controller: %w", err)
	}

	cstatus, err := controller.Status(ctx, sandbox.ID, r.GetVerbose())
	if err != nil {
		return nil, fmt.Errorf("failed to query controller status: %w", err)
	}

	status := toCRISandboxStatus(sandbox.Metadata, cstatus.State, cstatus.CreatedAt, ip, additionalIPs)
	if status.GetCreatedAt() == 0 {
		// CRI doesn't allow CreatedAt == 0.
		sandboxInfo, err := c.client.SandboxStore().Get(ctx, sandbox.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get sandbox %q from metadata store: %w", sandbox.ID, err)
		}
		status.CreatedAt = sandboxInfo.CreatedAt.UnixNano()
	}

	return &runtime.PodSandboxStatusResponse{
		Status: status,
		Info:   cstatus.Info,
	}, nil
}

// toCRISandboxStatus converts sandbox metadata into CRI pod sandbox status.
func toCRISandboxStatus(meta sandbox.Metadata, status string, createdAt time.Time, ip string, additionalIPs []string) *runtime.PodSandboxStatus {
	// Set sandbox state to NOTREADY by default.
	state := runtime.PodSandboxState_SANDBOX_NOTREADY
	if value, ok := runtime.PodSandboxState_value[status]; ok {
		state = runtime.PodSandboxState(value)
	}
	nsOpts := meta.Config.GetLinux().GetSecurityContext().GetNamespaceOptions()
	var ips []*runtime.PodIP
	for _, additionalIP := range additionalIPs {
		ips = append(ips, &runtime.PodIP{Ip: additionalIP})
	}
	return &runtime.PodSandboxStatus{
		Id:        meta.ID,
		Metadata:  meta.Config.GetMetadata(),
		State:     state,
		CreatedAt: createdAt.UnixNano(),
		Network: &runtime.PodSandboxNetworkStatus{
			Ip:            ip,
			AdditionalIps: ips,
		},
		Linux: &runtime.LinuxPodSandboxStatus{
			Namespaces: &runtime.Namespace{
				Options: nsOpts,
			},
		},
		Labels:         meta.Config.GetLabels(),
		Annotations:    meta.Config.GetAnnotations(),
		RuntimeHandler: meta.RuntimeHandler,
	}
}

func (c *CriService) getIPs(sandbox sandbox.Sandbox) (string, []string, error) {
	config := sandbox.Config

	// For sandboxes using the node network we are not
	// responsible for reporting the IP.
	if hostNetwork(config) {
		return "", nil, nil
	}

	if closed, err := sandbox.NetNS.Closed(); err != nil {
		return "", nil, fmt.Errorf("check network namespace closed: %w", err)
	} else if closed {
		return "", nil, nil
	}

	return sandbox.IP, sandbox.AdditionalIPs, nil
}
