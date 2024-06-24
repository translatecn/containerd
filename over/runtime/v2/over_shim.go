package v2

import (
	"context"
	eventstypes "demo/over/api/events"
	"demo/over/dialer"
	"demo/over/events/exchange"
	"demo/over/log"
	"demo/over/plugins/shim/shim"
	"demo/over/protobuf"
	ptypes "demo/over/protobuf/types"
	"demo/over/runtime"
	"demo/over/timeout"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"demo/over/ttrpc"
	"github.com/hashicorp/go-multierror"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"

	"demo/over/api/runtime/task/v2"
	"demo/over/api/types"
	"demo/over/errdefs"
	"demo/over/identifiers"
)

const (
	loadTimeout     = "io.containerd.timeout.shim.load"
	CleanupTimeout  = "io.containerd.timeout.shim.cleanup"
	shutdownTimeout = "io.containerd.timeout.shim.shutdown"
)

func init() {
	timeout.Set(loadTimeout, 5*time.Second)
	timeout.Set(CleanupTimeout, 5*time.Second)
	timeout.Set(shutdownTimeout, 3*time.Second)
}

func loadAddress(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func LoadShim(ctx context.Context, bundle *Bundle, onClose func()) (_ ShimInstance, retErr error) {
	address, err := loadAddress(filepath.Join(bundle.Path, "address"))
	if err != nil {
		return nil, err
	}

	shimCtx, cancelShimLog := context.WithCancel(ctx)
	defer func() {
		if retErr != nil {
			cancelShimLog()
		}
	}()
	f, err := openShimLog(shimCtx, bundle, shim.AnonReconnectDialer)
	if err != nil {
		return nil, fmt.Errorf("open shim log pipe when reload: %w", err)
	}
	defer func() {
		if retErr != nil {
			f.Close()
		}
	}()
	// open the log pipe and block until the writer is ready
	// this helps with synchronization of the shim
	// copy the shim's logs to containerd's output
	go func() {
		defer f.Close()
		_, err := io.Copy(os.Stderr, f)
		// To prevent flood of error messages, the expected error
		// should be reset, like os.ErrClosed or os.ErrNotExist, which
		// depends on platform.
		err = checkCopyShimLogError(ctx, err)
		if err != nil {
			log.G(ctx).WithError(err).Error("copy shim log after reload")
		}
	}()
	onCloseWithShimLog := func() {
		onClose()
		cancelShimLog()
		f.Close()
	}

	params, err := parseStartResponse(ctx, address)
	if err != nil {
		return nil, err
	}

	conn, err := makeConnection(ctx, params, onCloseWithShimLog)
	if err != nil {
		return nil, err
	}

	defer func() {
		if retErr != nil {
			conn.Close()
		}
	}()

	shim := &Shim{
		bundle: bundle,
		client: conn,
	}

	ctx, cancel := timeout.WithContext(ctx, loadTimeout)
	defer cancel()

	// Check connectivity, TaskService is the only required service, so create a temp one to check connection.
	s, err := NewShimTask(shim)
	if err != nil {
		return nil, err
	}

	if _, err := s.PID(ctx); err != nil {
		return nil, err
	}

	return shim, nil
}

// ShimInstance represents running shim process managed by ShimManager.
type ShimInstance interface {
	io.Closer

	// ID of the shim.
	ID() string
	// Namespace of this shim.
	Namespace() string
	// Bundle is a file system path to shim's bundle.
	Bundle() string
	// Client returns the underlying TTRPC or GRPC client object for this shim.
	// The underlying object can be either *ttrpc.Client or grpc.ClientConnInterface.
	Client() any
	// Delete will close the client and remove bundle from disk.
	Delete(ctx context.Context) error
}

func parseStartResponse(ctx context.Context, response []byte) (shim.BootstrapParams, error) {
	var params shim.BootstrapParams
	var rs []byte
	if strings.Contains(string(response), `API server listening at`) {
		x := strings.Split(string(response), "\n")
		rs = []byte(x[len(x)-1])
	} else {
		rs = response
	}
	if err := json.Unmarshal(rs, &params); err != nil || params.Version < 2 {
		// Use TTRPC for legacy shims
		params.Address = string(rs)
		params.Protocol = "ttrpc"
	}

	if params.Version > 2 {
		return shim.BootstrapParams{}, fmt.Errorf("unsupported shim version (%d): %w", params.Version, errdefs.ErrNotImplemented)
	}

	return params, nil
}

// makeConnection creates a new TTRPC or GRPC connection object from address.
// address can be either a socket path for TTRPC or JSON serialized BootstrapParams.
func makeConnection(ctx context.Context, params shim.BootstrapParams, onClose func()) (_ io.Closer, retErr error) {
	log.G(ctx).WithFields(log.Fields{
		"address":  params.Address,
		"protocol": params.Protocol,
	}).Debug("shim bootstrap parameters")

	switch strings.ToLower(params.Protocol) {
	case "ttrpc":
		conn, err := shim.Connect(params.Address, shim.AnonReconnectDialer)
		if err != nil {
			return nil, fmt.Errorf("failed to create TTRPC connection: %w", err)
		}
		defer func() {
			if retErr != nil {
				conn.Close()
			}
		}()

		return ttrpc.NewClient(conn, ttrpc.WithOnClose(onClose)), nil
	case "grpc":
		ctx, cancel := context.WithTimeout(ctx, time.Second*100)
		defer cancel()

		gopts := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		}
		return grpcDialContext(ctx, params.Address, onClose, gopts...)
	default:
		return nil, fmt.Errorf("unexpected protocol: %q", params.Protocol)
	}
}

// grpcDialContext and the underlying grpcConn type exist solely
// so we can have something similar to ttrpc.WithOnClose to have
// a callback run when the connection is severed or explicitly closed.
func grpcDialContext(
	ctx context.Context,
	address string,
	onClose func(),
	gopts ...grpc.DialOption,
) (*grpcConn, error) {
	// If grpc.WithBlock is specified in gopts this causes the connection to block waiting for
	// a connection regardless of if the socket exists or has a listener when Dial begins. This
	// specific behavior of WithBlock is mostly undesirable for shims, as if the socket isn't
	// there when we go to load/connect there's likely an issue. However, getting rid of WithBlock is
	// also undesirable as we don't want the background connection behavior, we want to ensure
	// a connection before moving on. To bring this in line with the ttrpc connection behavior
	// lets do an initial dial to ensure the shims socket is actually available. stat wouldn't suffice
	// here as if the shim exited unexpectedly its socket may still be on the filesystem, but it'd return
	// ECONNREFUSED which grpc.DialContext will happily trudge along through for the full timeout.
	//
	// This is especially helpful on restart of containerd as if the shim died while containerd
	// was down, we end up waiting the full timeout.
	conn, err := net.DialTimeout("unix", address, time.Second*10)
	if err != nil {
		return nil, err
	}
	conn.Close()

	target := dialer.DialAddress(address)
	client, err := grpc.DialContext(ctx, target, gopts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create GRPC connection: %w", err)
	}

	done := make(chan struct{})
	go func() {
		gctx := context.Background()
		sourceState := connectivity.Ready
		for {
			if client.WaitForStateChange(gctx, sourceState) {
				state := client.GetState()
				if state == connectivity.Idle || state == connectivity.Shutdown {
					break
				}
				// Could be transient failure. Lets see if we can get back to a working
				// state.
				log.G(gctx).WithFields(log.Fields{
					"state": state,
					"addr":  target,
				}).Warn("shim grpc connection unexpected state")
				sourceState = state
			}
		}
		onClose()
		close(done)
	}()

	return &grpcConn{
		ClientConn:  client,
		onCloseDone: done,
	}, nil
}

type grpcConn struct {
	*grpc.ClientConn
	onCloseDone chan struct{}
}

func (gc *grpcConn) UserOnCloseWait(ctx context.Context) error {
	select {
	case <-gc.onCloseDone:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type Shim struct {
	bundle *Bundle
	client any // unix:///run/containerd/s/1c2bf84c6529ba17d8234a68f92557fb5c1e4214eb17b724580e60b640e4f68a
}

var _ ShimInstance = (*Shim)(nil)

// ID of the shim/task
func (s *Shim) ID() string {
	return s.bundle.ID
}

func (s *Shim) Namespace() string {
	return s.bundle.Namespace
}

func (s *Shim) Bundle() string {
	return s.bundle.Path
}

func (s *Shim) Client() any {
	return s.client
}

// Close closes the underlying client connection.
func (s *Shim) Close() error {
	if ttrpcClient, ok := s.client.(*ttrpc.Client); ok {
		return ttrpcClient.Close()
	}

	if grpcClient, ok := s.client.(*grpcConn); ok {
		return grpcClient.Close()
	}

	return nil
}

func (s *Shim) Delete(ctx context.Context) error {
	var (
		result *multierror.Error
	)

	if ttrpcClient, ok := s.client.(*ttrpc.Client); ok {
		if err := ttrpcClient.Close(); err != nil {
			result = multierror.Append(result, fmt.Errorf("failed to close ttrpc client: %w", err))
		}

		if err := ttrpcClient.UserOnCloseWait(ctx); err != nil {
			result = multierror.Append(result, fmt.Errorf("close wait error: %w", err))
		}
	}

	if grpcClient, ok := s.client.(*grpcConn); ok {
		if err := grpcClient.Close(); err != nil {
			result = multierror.Append(result, fmt.Errorf("failed to close grpc client: %w", err))
		}

		if err := grpcClient.UserOnCloseWait(ctx); err != nil {
			result = multierror.Append(result, fmt.Errorf("close wait error: %w", err))
		}
	}

	if err := s.bundle.Delete(); err != nil {
		log.G(ctx).WithField("id", s.ID()).WithError(err).Error("failed to delete bundle")
		result = multierror.Append(result, fmt.Errorf("failed to delete bundle: %w", err))
	}

	return result.ErrorOrNil()
}

var _ runtime.Task = &ShimTask{}

// ShimTask wraps Shim process and adds task service client for compatibility with existing Shim manager.
type ShimTask struct {
	ShimInstance
	task task.TaskService
}

func NewShimTask(shim ShimInstance) (*ShimTask, error) {
	taskClient, err := NewTaskClient(shim.Client())
	if err != nil {
		return nil, err
	}

	return &ShimTask{
		ShimInstance: shim,
		task:         taskClient,
	}, nil
}

func (s *ShimTask) Shutdown(ctx context.Context) error {
	_, err := s.task.Shutdown(ctx, &task.ShutdownRequest{
		ID: s.ID(),
	})
	if err != nil && !errors.Is(err, ttrpc.ErrClosed) {
		return errdefs.FromGRPC(err)
	}
	return nil
}

func (s *ShimTask) waitShutdown(ctx context.Context) error {
	ctx, cancel := timeout.WithContext(ctx, shutdownTimeout)
	defer cancel()
	return s.Shutdown(ctx)
}

// PID of the task
func (s *ShimTask) PID(ctx context.Context) (uint32, error) {
	response, err := s.task.Connect(ctx, &task.ConnectRequest{
		ID: s.ID(),
	})
	if err != nil {
		return 0, errdefs.FromGRPC(err)
	}

	return response.TaskPid, nil
}

func (s *ShimTask) Delete(ctx context.Context, sandboxed bool, removeTask func(ctx context.Context, id string)) (*runtime.Exit, error) {
	response, shimErr := s.task.Delete(ctx, &task.DeleteRequest{
		ID: s.ID(),
	})
	if shimErr != nil {
		log.G(ctx).WithField("id", s.ID()).WithError(shimErr).Debug("failed to delete task")
		if !errors.Is(shimErr, ttrpc.ErrClosed) {
			shimErr = errdefs.FromGRPC(shimErr)
			if !errdefs.IsNotFound(shimErr) {
				return nil, shimErr
			}
		}
	}

	// NOTE: If the shim has been killed and ttrpc connection has been
	// closed, the shimErr will not be nil. For this case, the event
	// subscriber, like moby/moby, might have received the exit or delete
	// events. Just in case, we should allow ttrpc-callback-on-close to
	// send the exit and delete events again. And the exit status will
	// depend on result of shimV2.Delete.
	//
	// If not, the shim has been delivered the exit and delete events.
	// So we should remove the record and prevent duplicate events from
	// ttrpc-callback-on-close.
	//
	// TODO: It's hard to guarantee that the event is unique and sent only
	// once. The moby/moby should not rely on that assumption that there is
	// only one exit event. The moby/moby should handle the duplicate events.
	//
	// REF: https://github.com/containerd/issues/4769
	if shimErr == nil {
		removeTask(ctx, s.ID())
	}

	// Don't shutdown sandbox as there may be other containers running.
	// Let controller decide when to shutdown.
	if !sandboxed {
		if err := s.waitShutdown(ctx); err != nil {
			// FIXME(fuweid):
			//
			// If the error is context canceled, should we use context.TODO()
			// to wait for it?
			log.G(ctx).WithField("id", s.ID()).WithError(err).Error("failed to shutdown shim task and the shim might be leaked")
		}
	}

	if err := s.ShimInstance.Delete(ctx); err != nil {
		log.G(ctx).WithField("id", s.ID()).WithError(err).Error("failed to delete shim")
	}

	// remove self from the runtime task list
	// this seems dirty but it cleans up the API across runtimes, tasks, and the service
	removeTask(ctx, s.ID())

	if shimErr != nil {
		return nil, shimErr
	}

	return &runtime.Exit{
		Status:    response.ExitStatus,
		Timestamp: protobuf.FromTimestamp(response.ExitedAt),
		Pid:       response.Pid,
	}, nil
}

func (s *ShimTask) Create(ctx context.Context, opts runtime.CreateOpts) (runtime.Task, error) {
	topts := opts.TaskOptions
	if topts == nil || topts.GetValue() == nil {
		topts = opts.RuntimeOptions
	}
	request := &task.CreateTaskRequest{
		ID:         s.ID(),
		Bundle:     s.Bundle(),
		Stdin:      opts.IO.Stdin,
		Stdout:     opts.IO.Stdout,
		Stderr:     opts.IO.Stderr,
		Terminal:   opts.IO.Terminal,
		Checkpoint: opts.Checkpoint,
		Options:    protobuf.FromAny(topts),
	}
	for _, m := range opts.Rootfs {
		request.Rootfs = append(request.Rootfs, &types.Mount{
			Type:    m.Type,
			Source:  m.Source,
			Target:  m.Target,
			Options: m.Options,
		})
	}

	_, err := s.task.Create(ctx, request)
	if err != nil {
		return nil, errdefs.FromGRPC(err)
	}

	return s, nil
}

func (s *ShimTask) Pause(ctx context.Context) error {
	if _, err := s.task.Pause(ctx, &task.PauseRequest{
		ID: s.ID(),
	}); err != nil {
		return errdefs.FromGRPC(err)
	}
	return nil
}

func (s *ShimTask) Resume(ctx context.Context) error {
	if _, err := s.task.Resume(ctx, &task.ResumeRequest{
		ID: s.ID(),
	}); err != nil {
		return errdefs.FromGRPC(err)
	}
	return nil
}

func (s *ShimTask) Start(ctx context.Context) error {
	_, err := s.task.Start(ctx, &task.StartRequest{ // 调用 shim
		ID: s.ID(),
	})
	if err != nil {
		return errdefs.FromGRPC(err)
	}
	return nil
}

func (s *ShimTask) Kill(ctx context.Context, signal uint32, all bool) error {
	if _, err := s.task.Kill(ctx, &task.KillRequest{
		ID:     s.ID(),
		Signal: signal,
		All:    all,
	}); err != nil {
		return errdefs.FromGRPC(err)
	}
	return nil
}

func (s *ShimTask) Exec(ctx context.Context, id string, opts runtime.ExecOpts) (runtime.ExecProcess, error) {
	if err := identifiers.Validate(id); err != nil {
		return nil, fmt.Errorf("invalid exec id %s: %w", id, err)
	}
	request := &task.ExecProcessRequest{
		ID:       s.ID(),
		ExecID:   id,
		Stdin:    opts.IO.Stdin,
		Stdout:   opts.IO.Stdout,
		Stderr:   opts.IO.Stderr,
		Terminal: opts.IO.Terminal,
		Spec:     opts.Spec,
	}
	if _, err := s.task.Exec(ctx, request); err != nil {
		return nil, errdefs.FromGRPC(err)
	}
	return &process{
		id:   id,
		shim: s,
	}, nil
}

func (s *ShimTask) Pids(ctx context.Context) ([]runtime.ProcessInfo, error) {
	resp, err := s.task.Pids(ctx, &task.PidsRequest{
		ID: s.ID(),
	})
	if err != nil {
		return nil, errdefs.FromGRPC(err)
	}
	var processList []runtime.ProcessInfo
	for _, p := range resp.Processes {
		processList = append(processList, runtime.ProcessInfo{
			Pid:  p.Pid,
			Info: p.Info,
		})
	}
	return processList, nil
}

func (s *ShimTask) ResizePty(ctx context.Context, size runtime.ConsoleSize) error {
	_, err := s.task.ResizePty(ctx, &task.ResizePtyRequest{
		ID:     s.ID(),
		Width:  size.Width,
		Height: size.Height,
	})
	if err != nil {
		return errdefs.FromGRPC(err)
	}
	return nil
}

func (s *ShimTask) CloseIO(ctx context.Context) error {
	_, err := s.task.CloseIO(ctx, &task.CloseIORequest{
		ID:    s.ID(),
		Stdin: true,
	})
	if err != nil {
		return errdefs.FromGRPC(err)
	}
	return nil
}

func (s *ShimTask) Wait(ctx context.Context) (*runtime.Exit, error) {
	taskPid, err := s.PID(ctx)
	if err != nil {
		return nil, err
	}
	response, err := s.task.Wait(ctx, &task.WaitRequest{
		ID: s.ID(),
	})
	if err != nil {
		return nil, errdefs.FromGRPC(err)
	}
	return &runtime.Exit{
		Pid:       taskPid,
		Timestamp: protobuf.FromTimestamp(response.ExitedAt),
		Status:    response.ExitStatus,
	}, nil
}

func (s *ShimTask) Checkpoint(ctx context.Context, path string, options *ptypes.Any) error {
	request := &task.CheckpointTaskRequest{
		ID:      s.ID(),
		Path:    path,
		Options: options,
	}
	if _, err := s.task.Checkpoint(ctx, request); err != nil {
		return errdefs.FromGRPC(err)
	}
	return nil
}

func (s *ShimTask) Update(ctx context.Context, resources *ptypes.Any, annotations map[string]string) error {
	if _, err := s.task.Update(ctx, &task.UpdateTaskRequest{
		ID:          s.ID(),
		Resources:   resources,
		Annotations: annotations,
	}); err != nil {
		return errdefs.FromGRPC(err)
	}
	return nil
}
func CleanupAfterDeadShim(ctx context.Context, id string, rt *runtime.NSMap[ShimInstance], events *exchange.Exchange, binaryCall *binary) {
	ctx, cancel := timeout.WithContext(ctx, CleanupTimeout)
	defer cancel()

	log.G(ctx).WithField("id", id).Warn("cleaning up after shim disconnected")
	response, err := binaryCall.Delete(ctx)
	if err != nil {
		log.G(ctx).WithError(err).WithField("id", id).Warn("failed to clean up after shim disconnected")
	}

	if _, err := rt.Get(ctx, id); err != nil {
		// Task was never started or was already successfully deleted
		// No need to publish events
		return
	}

	var (
		pid        uint32
		exitStatus uint32
		exitedAt   time.Time
	)
	if response != nil {
		pid = response.Pid
		exitStatus = response.Status
		exitedAt = response.Timestamp
	} else {
		exitStatus = 255
		exitedAt = time.Now()
	}
	events.Publish(ctx, runtime.TaskExitEventTopic, &eventstypes.TaskExit{
		ContainerID: id,
		ID:          id,
		Pid:         pid,
		ExitStatus:  exitStatus,
		ExitedAt:    protobuf.ToTimestamp(exitedAt),
	})

	events.Publish(ctx, runtime.TaskDeleteEventTopic, &eventstypes.TaskDelete{
		ContainerID: id,
		Pid:         pid,
		ExitStatus:  exitStatus,
		ExitedAt:    protobuf.ToTimestamp(exitedAt),
	})
}

func (s *ShimTask) Stats(ctx context.Context) (*ptypes.Any, error) {
	response, err := s.task.Stats(ctx, &task.StatsRequest{
		ID: s.ID(),
	})
	if err != nil {
		return nil, errdefs.FromGRPC(err)
	}
	return response.Stats, nil
}

func (s *ShimTask) Process(ctx context.Context, id string) (runtime.ExecProcess, error) {
	p := &process{
		id:   id,
		shim: s,
	}
	if _, err := p.State(ctx); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *ShimTask) State(ctx context.Context) (runtime.State, error) {
	response, err := s.task.State(ctx, &task.StateRequest{
		ID: s.ID(),
	})
	if err != nil {
		if !errors.Is(err, ttrpc.ErrClosed) {
			return runtime.State{}, errdefs.FromGRPC(err)
		}
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
		ExitedAt:   protobuf.FromTimestamp(response.ExitedAt),
	}, nil
}
