package shim

import (
	"context"
	"demo/pkg/api/runctypes"
	"demo/pkg/api/types/task"
	"demo/pkg/console"
	"demo/pkg/errdefs"
	"demo/pkg/events"
	"demo/pkg/log"
	"demo/pkg/mount"
	"demo/pkg/namespaces"
	process3 "demo/pkg/process"
	"demo/pkg/protobuf"
	"demo/pkg/runtime"
	stdio2 "demo/pkg/stdio"
	"demo/pkg/sys/reaper"
	"demo/pkg/typeurl/v2"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	eventstypes "demo/pkg/api/events"
	ptypes "demo/pkg/protobuf/types"
	shimapi "demo/pkg/runtime/v1/shim/v1"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	empty   = &ptypes.Empty{}
	bufPool = sync.Pool{
		New: func() interface{} {
			buffer := make([]byte, 4096)
			return &buffer
		},
	}
)

// Config contains shim specific configuration
type Config struct {
	Path      string
	Namespace string
	WorkDir   string
	// Criu is the path to the criu binary used for checkpoint and restore.
	//
	// Deprecated: runc option --criu is now ignored (with a warning), and the
	// option will be removed entirely in a future release. Users who need a non-
	// standard criu binary should rely on the standard way of looking up binaries
	// in $PATH.
	Criu          string
	RuntimeRoot   string
	SystemdCgroup bool
}

// NewService returns a new shim service that can be used via GRPC
func NewService(config Config, publisher events.Publisher) (*Service, error) {
	if config.Namespace == "" {
		return nil, fmt.Errorf("shim namespace cannot be empty")
	}
	ctx := namespaces.WithNamespace(context.Background(), config.Namespace)
	ctx = log.WithLogger(ctx, logrus.WithFields(log.Fields{
		"namespace": config.Namespace,
		"path":      config.Path,
		"pid":       os.Getpid(),
	}))
	s := &Service{
		config:    config,
		context:   ctx,
		processes: make(map[string]process3.Process),
		events:    make(chan interface{}, 128),
		ec:        reaper.Default.Subscribe(),
	}
	go s.processExits()
	if err := s.initPlatform(); err != nil {
		return nil, fmt.Errorf("failed to initialized platform behavior: %w", err)
	}
	go s.forward(publisher)
	return s, nil
}

// Service is the shim implementation of a remote shim over GRPC
type Service struct {
	mu sync.Mutex

	config    Config
	context   context.Context
	processes map[string]process3.Process
	events    chan interface{}
	platform  stdio2.Platform
	ec        chan reaper.Exit

	// Filled by Create()
	id     string
	bundle string
}

// Start a process
func (s *Service) Start(ctx context.Context, r *shimapi.StartRequest) (*shimapi.StartResponse, error) {
	p, err := s.getExecProcess(r.ID)
	if err != nil {
		return nil, err
	}
	if err := p.Start(ctx); err != nil {
		return nil, err
	}
	return &shimapi.StartResponse{
		ID:  p.ID(),
		Pid: uint32(p.Pid()),
	}, nil
}

// Delete the initial process and container
func (s *Service) Delete(ctx context.Context, r *ptypes.Empty) (*shimapi.DeleteResponse, error) {
	p, err := s.getInitProcess()
	if err != nil {
		return nil, err
	}
	if err := p.Delete(ctx); err != nil {
		return nil, errdefs.ToGRPC(err)
	}
	s.mu.Lock()
	delete(s.processes, s.id)
	s.mu.Unlock()
	s.platform.Close()
	return &shimapi.DeleteResponse{
		ExitStatus: uint32(p.ExitStatus()),
		ExitedAt:   protobuf.ToTimestamp(p.ExitedAt()),
		Pid:        uint32(p.Pid()),
	}, nil
}

// DeleteProcess deletes an exec'd process
func (s *Service) DeleteProcess(ctx context.Context, r *shimapi.DeleteProcessRequest) (*shimapi.DeleteResponse, error) {
	if r.ID == s.id {
		return nil, status.Errorf(codes.InvalidArgument, "cannot delete init process with DeleteProcess")
	}
	p, err := s.getExecProcess(r.ID)
	if err != nil {
		return nil, err
	}
	if err := p.Delete(ctx); err != nil {
		return nil, errdefs.ToGRPC(err)
	}
	s.mu.Lock()
	delete(s.processes, r.ID)
	s.mu.Unlock()
	return &shimapi.DeleteResponse{
		ExitStatus: uint32(p.ExitStatus()),
		ExitedAt:   protobuf.ToTimestamp(p.ExitedAt()),
		Pid:        uint32(p.Pid()),
	}, nil
}

// Exec an additional process inside the container
func (s *Service) Exec(ctx context.Context, r *shimapi.ExecProcessRequest) (*ptypes.Empty, error) {
	s.mu.Lock()

	if p := s.processes[r.ID]; p != nil {
		s.mu.Unlock()
		return nil, errdefs.ToGRPCf(errdefs.ErrAlreadyExists, "id %s", r.ID)
	}

	p := s.processes[s.id]
	s.mu.Unlock()
	if p == nil {
		return nil, errdefs.ToGRPCf(errdefs.ErrFailedPrecondition, "container must be created")
	}

	process, err := p.(*process3.Init).Exec(ctx, s.config.Path, &process3.ExecConfig{
		ID:       r.ID,
		Terminal: r.Terminal,
		Stdin:    r.Stdin,
		Stdout:   r.Stdout,
		Stderr:   r.Stderr,
		Spec:     r.Spec,
	})
	if err != nil {
		return nil, errdefs.ToGRPC(err)
	}
	s.mu.Lock()
	s.processes[r.ID] = process
	s.mu.Unlock()
	return empty, nil
}

// ResizePty of a process
func (s *Service) ResizePty(ctx context.Context, r *shimapi.ResizePtyRequest) (*ptypes.Empty, error) {
	if r.ID == "" {
		return nil, errdefs.ToGRPCf(errdefs.ErrInvalidArgument, "id not provided")
	}
	ws := console.WinSize{
		Width:  uint16(r.Width),
		Height: uint16(r.Height),
	}
	s.mu.Lock()
	p := s.processes[r.ID]
	s.mu.Unlock()
	if p == nil {
		return nil, fmt.Errorf("process does not exist %s", r.ID)
	}
	if err := p.Resize(ws); err != nil {
		return nil, errdefs.ToGRPC(err)
	}
	return empty, nil
}

// State returns runtime state information for a process
func (s *Service) State(ctx context.Context, r *shimapi.StateRequest) (*shimapi.StateResponse, error) {
	p, err := s.getExecProcess(r.ID)
	if err != nil {
		return nil, err
	}
	st, err := p.Status(ctx)
	if err != nil {
		return nil, err
	}
	status := task.Status_UNKNOWN
	switch st {
	case "created":
		status = task.Status_CREATED
	case "running":
		status = task.Status_RUNNING
	case "stopped":
		status = task.Status_STOPPED
	case "paused":
		status = task.Status_PAUSED
	case "pausing":
		status = task.Status_PAUSING
	}
	sio := p.Stdio()
	return &shimapi.StateResponse{
		ID:         p.ID(),
		Bundle:     s.bundle,
		Pid:        uint32(p.Pid()),
		Status:     status,
		Stdin:      sio.Stdin,
		Stdout:     sio.Stdout,
		Stderr:     sio.Stderr,
		Terminal:   sio.Terminal,
		ExitStatus: uint32(p.ExitStatus()),
		ExitedAt:   protobuf.ToTimestamp(p.ExitedAt()),
	}, nil
}

// Pause the container
func (s *Service) Pause(ctx context.Context, r *ptypes.Empty) (*ptypes.Empty, error) {
	p, err := s.getInitProcess()
	if err != nil {
		return nil, err
	}
	if err := p.(*process3.Init).Pause(ctx); err != nil {
		return nil, err
	}
	return empty, nil
}

// Resume the container
func (s *Service) Resume(ctx context.Context, r *ptypes.Empty) (*ptypes.Empty, error) {
	p, err := s.getInitProcess()
	if err != nil {
		return nil, err
	}
	if err := p.(*process3.Init).Resume(ctx); err != nil {
		return nil, err
	}
	return empty, nil
}

// Kill a process with the provided signal
func (s *Service) Kill(ctx context.Context, r *shimapi.KillRequest) (*ptypes.Empty, error) {
	if r.ID == "" {
		p, err := s.getInitProcess()
		if err != nil {
			return nil, err
		}
		if err := p.Kill(ctx, r.Signal, r.All); err != nil {
			return nil, errdefs.ToGRPC(err)
		}
		return empty, nil
	}

	p, err := s.getExecProcess(r.ID)
	if err != nil {
		return nil, err
	}
	if err := p.Kill(ctx, r.Signal, r.All); err != nil {
		return nil, errdefs.ToGRPC(err)
	}
	return empty, nil
}

// ListPids returns all pids inside the container
func (s *Service) ListPids(ctx context.Context, r *shimapi.ListPidsRequest) (*shimapi.ListPidsResponse, error) {
	pids, err := s.getContainerPids(ctx, r.ID)
	if err != nil {
		return nil, errdefs.ToGRPC(err)
	}
	var processes []*task.ProcessInfo

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, pid := range pids {
		pInfo := task.ProcessInfo{
			Pid: pid,
		}
		for _, p := range s.processes {
			if p.Pid() == int(pid) {
				d := &runctypes.ProcessDetails{
					ExecID: p.ID(),
				}
				a, err := typeurl.MarshalAny(d)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal process %d info: %w", pid, err)
				}
				pInfo.Info = protobuf.FromAny(a)
				break
			}
		}
		processes = append(processes, &pInfo)
	}
	return &shimapi.ListPidsResponse{
		Processes: processes,
	}, nil
}

// CloseIO of a process
func (s *Service) CloseIO(ctx context.Context, r *shimapi.CloseIORequest) (*ptypes.Empty, error) {
	p, err := s.getExecProcess(r.ID)
	if err != nil {
		return nil, err
	}
	if stdin := p.Stdin(); stdin != nil {
		if err := stdin.Close(); err != nil {
			return nil, fmt.Errorf("close stdin: %w", err)
		}
	}
	return empty, nil
}

// Checkpoint the container
func (s *Service) Checkpoint(ctx context.Context, r *shimapi.CheckpointTaskRequest) (*ptypes.Empty, error) {
	p, err := s.getInitProcess()
	if err != nil {
		return nil, err
	}
	var options *runctypes.CheckpointOptions
	if r.Options != nil {
		v, err := typeurl.UnmarshalAny(r.Options)
		if err != nil {
			return nil, err
		}
		options = v.(*runctypes.CheckpointOptions)
	}
	if err := p.(*process3.Init).Checkpoint(ctx, &process3.CheckpointConfig{
		Path:                     r.Path,
		Exit:                     options.Exit,
		AllowOpenTCP:             options.OpenTcp,
		AllowExternalUnixSockets: options.ExternalUnixSockets,
		AllowTerminal:            options.Terminal,
		FileLocks:                options.FileLocks,
		EmptyNamespaces:          options.EmptyNamespaces,
		WorkDir:                  options.WorkPath,
	}); err != nil {
		return nil, errdefs.ToGRPC(err)
	}
	return empty, nil
}

// ShimInfo returns shim information such as the shim's pid
func (s *Service) ShimInfo(ctx context.Context, r *ptypes.Empty) (*shimapi.ShimInfoResponse, error) {
	return &shimapi.ShimInfoResponse{
		ShimPid: uint32(os.Getpid()),
	}, nil
}

// Update a running container
func (s *Service) Update(ctx context.Context, r *shimapi.UpdateTaskRequest) (*ptypes.Empty, error) {
	p, err := s.getInitProcess()
	if err != nil {
		return nil, err
	}
	if err := p.(*process3.Init).Update(ctx, r.Resources); err != nil {
		return nil, errdefs.ToGRPC(err)
	}
	return empty, nil
}

// Wait for a process to exit
func (s *Service) Wait(ctx context.Context, r *shimapi.WaitRequest) (*shimapi.WaitResponse, error) {
	p, err := s.getExecProcess(r.ID)
	if err != nil {
		return nil, err
	}
	p.Wait()

	return &shimapi.WaitResponse{
		ExitStatus: uint32(p.ExitStatus()),
		ExitedAt:   protobuf.ToTimestamp(p.ExitedAt()),
	}, nil
}

func (s *Service) processExits() {
	for e := range s.ec {
		s.checkProcesses(e)
	}
}

func (s *Service) checkProcesses(e reaper.Exit) {
	var p process3.Process
	s.mu.Lock()
	for _, proc := range s.processes {
		if proc.Pid() == e.Pid {
			p = proc
			break
		}
	}
	s.mu.Unlock()
	if p == nil {
		log.G(s.context).Debugf("process with id:%d wasn't found", e.Pid)
		return
	}
	if ip, ok := p.(*process3.Init); ok {
		// Ensure all children are killed
		if shouldKillAllOnExit(s.context, s.bundle) {
			if err := ip.KillAll(s.context); err != nil {
				log.G(s.context).WithError(err).WithField("id", ip.ID()).
					Error("failed to kill init's children")
			}
		}
	}

	p.SetExited(e.Status)
	s.events <- &eventstypes.TaskExit{
		ContainerID: s.id,
		ID:          p.ID(),
		Pid:         uint32(e.Pid),
		ExitStatus:  uint32(e.Status),
		ExitedAt:    protobuf.ToTimestamp(p.ExitedAt()),
	}
}

func shouldKillAllOnExit(ctx context.Context, bundlePath string) bool {
	var bundleSpec specs.Spec
	bundleConfigContents, err := os.ReadFile(filepath.Join(bundlePath, "config.json"))
	if err != nil {
		log.G(ctx).WithError(err).Error("shouldKillAllOnExit: failed to read config.json")
		return true
	}
	if err := json.Unmarshal(bundleConfigContents, &bundleSpec); err != nil {
		log.G(ctx).WithError(err).Error("shouldKillAllOnExit: failed to unmarshal bundle json")
		return true
	}
	if bundleSpec.Linux != nil {
		for _, ns := range bundleSpec.Linux.Namespaces {
			if ns.Type == specs.PIDNamespace && ns.Path == "" {
				return false
			}
		}
	}
	return true
}

func (s *Service) getContainerPids(ctx context.Context, id string) ([]uint32, error) {
	p, err := s.getInitProcess()
	if err != nil {
		return nil, err
	}

	ps, err := p.(*process3.Init).Runtime().Ps(ctx, id)
	if err != nil {
		return nil, err
	}
	pids := make([]uint32, 0, len(ps))
	for _, pid := range ps {
		pids = append(pids, uint32(pid))
	}
	return pids, nil
}

func (s *Service) forward(publisher events.Publisher) {
	for e := range s.events {
		if err := publisher.Publish(s.context, getTopic(s.context, e), e); err != nil {
			log.G(s.context).WithError(err).Error("post event")
		}
	}
}

// getInitProcess returns initial process
func (s *Service) getInitProcess() (process3.Process, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p := s.processes[s.id]
	if p == nil {
		return nil, errdefs.ToGRPCf(errdefs.ErrFailedPrecondition, "container must be created")
	}
	return p, nil
}

// getExecProcess returns exec process
func (s *Service) getExecProcess(id string) (process3.Process, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p := s.processes[id]
	if p == nil {
		return nil, errdefs.ToGRPCf(errdefs.ErrNotFound, "process %s does not exist", id)
	}
	return p, nil
}

func getTopic(ctx context.Context, e interface{}) string {
	switch e.(type) {
	case *eventstypes.TaskCreate:
		return runtime.TaskCreateEventTopic
	case *eventstypes.TaskStart:
		return runtime.TaskStartEventTopic
	case *eventstypes.TaskOOM:
		return runtime.TaskOOMEventTopic
	case *eventstypes.TaskExit:
		return runtime.TaskExitEventTopic
	case *eventstypes.TaskDelete:
		return runtime.TaskDeleteEventTopic
	case *eventstypes.TaskExecAdded:
		return runtime.TaskExecAddedEventTopic
	case *eventstypes.TaskExecStarted:
		return runtime.TaskExecStartedEventTopic
	case *eventstypes.TaskPaused:
		return runtime.TaskPausedEventTopic
	case *eventstypes.TaskResumed:
		return runtime.TaskResumedEventTopic
	case *eventstypes.TaskCheckpointed:
		return runtime.TaskCheckpointedEventTopic
	default:
		logrus.Warnf("no topic for type %#v", e)
	}
	return runtime.TaskUnknownTopic
}

func newInit(ctx context.Context, path, workDir, runtimeRoot, namespace string, systemdCgroup bool, platform stdio2.Platform, r *process3.CreateConfig, rootfs string) (*process3.Init, error) {
	options := &runctypes.CreateOptions{}
	if r.Options != nil {
		v, err := typeurl.UnmarshalAny(r.Options)
		if err != nil {
			return nil, err
		}
		options = v.(*runctypes.CreateOptions)
	}

	runtime := process3.NewRunc(runtimeRoot, path, namespace, r.Runtime, systemdCgroup)
	p := process3.New(r.ID, runtime, stdio2.Stdio{
		Stdin:    r.Stdin,
		Stdout:   r.Stdout,
		Stderr:   r.Stderr,
		Terminal: r.Terminal,
	})
	p.Bundle = r.Bundle
	p.Platform = platform
	p.Rootfs = rootfs
	p.WorkDir = workDir
	p.IoUID = int(options.IoUid)
	p.IoGID = int(options.IoGid)
	p.NoPivotRoot = options.NoPivotRoot
	p.NoNewKeyring = options.NoNewKeyring
	p.CriuWorkPath = options.CriuWorkPath
	if p.CriuWorkPath == "" {
		// if criu work path not set, use container WorkDir
		p.CriuWorkPath = p.WorkDir
	}

	return p, nil
}
func (s *Service) Create(ctx context.Context, r *shimapi.CreateTaskRequest) (_ *shimapi.CreateTaskResponse, err error) {
	var pmounts []process3.Mount
	for _, m := range r.Rootfs {
		pmounts = append(pmounts, process3.Mount{
			Type:    m.Type,
			Source:  m.Source,
			Target:  m.Target,
			Options: m.Options,
		})
	}

	rootfs := ""
	if len(pmounts) > 0 {
		rootfs = filepath.Join(r.Bundle, "rootfs")
		if err := os.Mkdir(rootfs, 0711); err != nil && !os.IsExist(err) {
			return nil, err
		}
	}

	config := &process3.CreateConfig{
		ID:               r.ID,
		Bundle:           r.Bundle,
		Runtime:          r.Runtime,
		Rootfs:           pmounts,
		Terminal:         r.Terminal,
		Stdin:            r.Stdin,
		Stdout:           r.Stdout,
		Stderr:           r.Stderr,
		Checkpoint:       r.Checkpoint,
		ParentCheckpoint: r.ParentCheckpoint,
		Options:          r.Options,
	}
	var mounts []mount.Mount
	for _, pm := range pmounts {
		mounts = append(mounts, mount.Mount{
			Type:    pm.Type,
			Source:  pm.Source,
			Target:  pm.Target,
			Options: pm.Options,
		})
	}
	defer func() {
		if err != nil {
			if err2 := mount.UnmountMounts(mounts, rootfs, 0); err2 != nil {
				log.G(ctx).WithError(err2).Warn("Failed to cleanup rootfs mount")
			}
		}
	}()
	if err := mount.All(mounts, rootfs); err != nil {
		return nil, fmt.Errorf("failed to mount rootfs component: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	process, err := newInit(
		ctx,
		s.config.Path,
		s.config.WorkDir,
		s.config.RuntimeRoot,
		s.config.Namespace,
		s.config.SystemdCgroup,
		s.platform,
		config,
		rootfs,
	)
	if err != nil {
		return nil, errdefs.ToGRPC(err)
	}
	if err := process.Create(ctx, config); err != nil {
		return nil, errdefs.ToGRPC(err)
	}
	// save the main task id and bundle to the shim for additional requests
	s.id = r.ID
	s.bundle = r.Bundle
	pid := process.Pid()
	s.processes[r.ID] = process
	return &shimapi.CreateTaskResponse{
		Pid: uint32(pid),
	}, nil
}