package linux

import (
	"context"
	"demo/pkg/api/types/task"
	"demo/pkg/errdefs"
	"demo/pkg/protobuf"
	"demo/pkg/runtime"
	"demo/pkg/ttrpc"
	"errors"

	eventstypes "demo/pkg/api/events"
	shim "demo/pkg/runtime/v1/shim/v1"
)

// Process implements a linux process
type Process struct {
	id string
	t  *Task
}

// ID of the process
func (p *Process) ID() string {
	return p.id
}

// Kill sends the provided signal to the underlying process
//
// Unable to kill all processes in the task using this method on a process
func (p *Process) Kill(ctx context.Context, signal uint32, _ bool) error {
	_, err := p.t.shim.Kill(ctx, &shim.KillRequest{
		Signal: signal,
		ID:     p.id,
	})
	if err != nil {
		return errdefs.FromGRPC(err)
	}
	return err
}

func statusFromProto(from task.Status) runtime.Status {
	var status runtime.Status
	switch from {
	case task.Status_CREATED:
		status = runtime.CreatedStatus
	case task.Status_RUNNING:
		status = runtime.RunningStatus
	case task.Status_STOPPED:
		status = runtime.StoppedStatus
	case task.Status_PAUSED:
		status = runtime.PausedStatus
	case task.Status_PAUSING:
		status = runtime.PausingStatus
	}
	return status
}

// State of process
func (p *Process) State(ctx context.Context) (runtime.State, error) {
	// use the container status for the status of the process
	response, err := p.t.shim.State(ctx, &shim.StateRequest{
		ID: p.id,
	})
	if err != nil {
		if !errors.Is(err, ttrpc.ErrClosed) {
			return runtime.State{}, errdefs.FromGRPC(err)
		}

		// We treat ttrpc.ErrClosed as the shim being closed, but really this
		// likely means that the process no longer exists. We'll have to plumb
		// the connection differently if this causes problems.
		return runtime.State{}, errdefs.ErrNotFound
	}
	return runtime.State{
		Pid:        response.Pid,
		Status:     statusFromProto(response.Status),
		Stdin:      response.Stdin,
		Stdout:     response.Stdout,
		Stderr:     response.Stderr,
		Terminal:   response.Terminal,
		ExitStatus: response.ExitStatus,
	}, nil
}

// ResizePty changes the side of the process's PTY to the provided width and height
func (p *Process) ResizePty(ctx context.Context, size runtime.ConsoleSize) error {
	_, err := p.t.shim.ResizePty(ctx, &shim.ResizePtyRequest{
		ID:     p.id,
		Width:  size.Width,
		Height: size.Height,
	})
	if err != nil {
		err = errdefs.FromGRPC(err)
	}
	return err
}

// CloseIO closes the provided IO pipe for the process
func (p *Process) CloseIO(ctx context.Context) error {
	_, err := p.t.shim.CloseIO(ctx, &shim.CloseIORequest{
		ID:    p.id,
		Stdin: true,
	})
	if err != nil {
		return errdefs.FromGRPC(err)
	}
	return nil
}

// Start the process
func (p *Process) Start(ctx context.Context) error {
	r, err := p.t.shim.Start(ctx, &shim.StartRequest{
		ID: p.id,
	})
	if err != nil {
		return errdefs.FromGRPC(err)
	}
	p.t.events.Publish(ctx, runtime.TaskExecStartedEventTopic, &eventstypes.TaskExecStarted{
		ContainerID: p.t.id,
		Pid:         r.Pid,
		ExecID:      p.id,
	})
	return nil
}

// Wait on the process to exit and return the exit status and timestamp
func (p *Process) Wait(ctx context.Context) (*runtime.Exit, error) {
	r, err := p.t.shim.Wait(ctx, &shim.WaitRequest{
		ID: p.id,
	})
	if err != nil {
		return nil, err
	}
	return &runtime.Exit{
		Timestamp: protobuf.FromTimestamp(r.ExitedAt),
		Status:    r.ExitStatus,
	}, nil
}

// Delete the process and return the exit status
func (p *Process) Delete(ctx context.Context) (*runtime.Exit, error) {
	r, err := p.t.shim.DeleteProcess(ctx, &shim.DeleteProcessRequest{
		ID: p.id,
	})
	if err != nil {
		return nil, errdefs.FromGRPC(err)
	}
	return &runtime.Exit{
		Status:    r.ExitStatus,
		Timestamp: protobuf.FromTimestamp(r.ExitedAt),
		Pid:       r.Pid,
	}, nil
}