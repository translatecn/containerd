package task

import (
	"context"
	runcconfig "demo/config/runc"
	"demo/pkg/cgroups/v3"
	"demo/pkg/namespaces"
	"demo/pkg/oom"
	oomv1 "demo/pkg/oom/v1"
	oomv2 "demo/pkg/oom/v2"
	"demo/pkg/plugins/shim/shim"
	"demo/pkg/process"
	"demo/pkg/protobuf"
	ptypes "demo/pkg/protobuf/types"
	"demo/pkg/shutdown"
	"demo/pkg/stdio"
	"demo/pkg/sys/reaper"
	"demo/pkg/typeurl/v2"
	"demo/pkg/userns"
	"fmt"
	"os"
	"sync"

	eventstypes "demo/pkg/api/events"
	taskAPI "demo/pkg/api/runtime/task/v2"
	"demo/pkg/api/types/task"
	"demo/pkg/cgroups/v3/cgroup1"
	cgroupsv2 "demo/pkg/cgroups/v3/cgroup2"
	"demo/pkg/errdefs"
	"demo/pkg/runtime/v2/runc"
	"demo/pkg/ttrpc"
	"github.com/sirupsen/logrus"
)

var (
	_     = (taskAPI.TaskService)(&Service{})
	empty = &ptypes.Empty{}
)

// Start a process
func (s *Service) Start(ctx context.Context, r *taskAPI.StartRequest) (*taskAPI.StartResponse, error) {
	container, err := s.getContainer(r.ID)
	if err != nil {
		return nil, err
	}

	var cinit *runc.Container
	s.lifecycleMu.Lock()
	if r.ExecID == "" {
		cinit = container
	} else {
		s.pendingExecs[container]++
	}
	handleStarted, cleanup := s.preStart(cinit)
	s.lifecycleMu.Unlock()
	defer cleanup()

	p, err := container.Start(ctx, r) // runc init 没了
	if err != nil {
		handleStarted(container, p)
		return nil, errdefs.ToGRPC(err)
	}

	switch r.ExecID {
	case "":
		switch cg := container.Cgroup().(type) {
		case cgroup1.Cgroup:
			if err := s.ep.Add(container.ID, cg); err != nil {
				logrus.WithError(err).Error("add cg to OOM monitor")
			}
		case *cgroupsv2.Manager:
			allControllers, err := cg.RootControllers()
			if err != nil {
				logrus.WithError(err).Error("failed to get root controllers")
			} else {
				if err := cg.ToggleControllers(allControllers, cgroupsv2.Enable); err != nil {
					if userns.RunningInUserNS() {
						logrus.WithError(err).Debugf("failed to enable controllers (%v)", allControllers)
					} else {
						logrus.WithError(err).Errorf("failed to enable controllers (%v)", allControllers)
					}
				}
			}
			if err := s.ep.Add(container.ID, cg); err != nil {
				logrus.WithError(err).Error("add cg to OOM monitor")
			}
		}

		s.send(&eventstypes.TaskStart{
			ContainerID: container.ID,
			Pid:         uint32(p.Pid()),
		})
	default:
		s.send(&eventstypes.TaskExecStarted{
			ContainerID: container.ID,
			ExecID:      r.ExecID,
			Pid:         uint32(p.Pid()),
		})
	}
	handleStarted(container, p)
	return &taskAPI.StartResponse{
		Pid: uint32(p.Pid()),
	}, nil
}

// Exec an additional process inside the container
func (s *Service) Exec(ctx context.Context, r *taskAPI.ExecProcessRequest) (*ptypes.Empty, error) {
	container, err := s.getContainer(r.ID)
	if err != nil {
		return nil, err
	}
	ok, cancel := container.ReserveProcess(r.ExecID)
	if !ok {
		return nil, errdefs.ToGRPCf(errdefs.ErrAlreadyExists, "id %s", r.ExecID)
	}
	process, err := container.Exec(ctx, r)
	if err != nil {
		cancel()
		return nil, errdefs.ToGRPC(err)
	}

	s.send(&eventstypes.TaskExecAdded{
		ContainerID: container.ID,
		ExecID:      process.ID(),
	})
	return empty, nil
}

// State returns runtime state information for a process
func (s *Service) State(ctx context.Context, r *taskAPI.StateRequest) (*taskAPI.StateResponse, error) {
	container, err := s.getContainer(r.ID)
	if err != nil {
		return nil, err
	}
	p, err := container.Process(r.ExecID)
	if err != nil {
		return nil, errdefs.ToGRPC(err)
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
	return &taskAPI.StateResponse{
		ID:         p.ID(),
		Bundle:     container.Bundle,
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

// Kill a process with the provided signal
func (s *Service) Kill(ctx context.Context, r *taskAPI.KillRequest) (*ptypes.Empty, error) {
	container, err := s.getContainer(r.ID)
	if err != nil {
		return nil, err
	}
	if err := container.Kill(ctx, r); err != nil {
		return nil, errdefs.ToGRPC(err)
	}
	return empty, nil
}

// Pids returns all pids inside the container
func (s *Service) Pids(ctx context.Context, r *taskAPI.PidsRequest) (*taskAPI.PidsResponse, error) {
	container, err := s.getContainer(r.ID)
	if err != nil {
		return nil, err
	}
	pids, err := s.getContainerPids(ctx, container)
	if err != nil {
		return nil, errdefs.ToGRPC(err)
	}
	var processes []*task.ProcessInfo
	for _, pid := range pids {
		pInfo := task.ProcessInfo{
			Pid: pid,
		}
		for _, p := range container.ExecdProcesses() {
			if p.Pid() == int(pid) {
				d := &runcconfig.ProcessDetails{
					ExecID: p.ID(),
				}
				a, err := protobuf.MarshalAnyToProto(d)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal process %d info: %w", pid, err)
				}
				pInfo.Info = a
				break
			}
		}
		processes = append(processes, &pInfo)
	}
	return &taskAPI.PidsResponse{
		Processes: processes,
	}, nil
}

// CloseIO of a process
func (s *Service) CloseIO(ctx context.Context, r *taskAPI.CloseIORequest) (*ptypes.Empty, error) {
	container, err := s.getContainer(r.ID)
	if err != nil {
		return nil, err
	}
	if err := container.CloseIO(ctx, r); err != nil {
		return nil, err
	}
	return empty, nil
}

// Checkpoint the container
func (s *Service) Checkpoint(ctx context.Context, r *taskAPI.CheckpointTaskRequest) (*ptypes.Empty, error) {
	container, err := s.getContainer(r.ID)
	if err != nil {
		return nil, err
	}
	if err := container.Checkpoint(ctx, r); err != nil {
		return nil, errdefs.ToGRPC(err)
	}
	return empty, nil
}

// Update a running container
func (s *Service) Update(ctx context.Context, r *taskAPI.UpdateTaskRequest) (*ptypes.Empty, error) {
	container, err := s.getContainer(r.ID)
	if err != nil {
		return nil, err
	}
	if err := container.Update(ctx, r); err != nil {
		return nil, errdefs.ToGRPC(err)
	}
	return empty, nil
}

// Wait for a process to exit
func (s *Service) Wait(ctx context.Context, r *taskAPI.WaitRequest) (*taskAPI.WaitResponse, error) {
	container, err := s.getContainer(r.ID)
	if err != nil {
		return nil, err
	}
	p, err := container.Process(r.ExecID)
	if err != nil {
		return nil, errdefs.ToGRPC(err)
	}
	p.Wait()

	return &taskAPI.WaitResponse{
		ExitStatus: uint32(p.ExitStatus()),
		ExitedAt:   protobuf.ToTimestamp(p.ExitedAt()),
	}, nil
}

// Connect returns shim information such as the shim's pid
func (s *Service) Connect(ctx context.Context, r *taskAPI.ConnectRequest) (*taskAPI.ConnectResponse, error) {
	var pid int
	if container, err := s.getContainer(r.ID); err == nil {
		pid = container.Pid()
	}
	return &taskAPI.ConnectResponse{
		ShimPid: uint32(os.Getpid()),
		TaskPid: uint32(pid),
	}, nil
}

func (s *Service) Shutdown(ctx context.Context, r *taskAPI.ShutdownRequest) (*ptypes.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// return out if the shim is still servicing containers
	if len(s.containers) > 0 {
		return empty, nil
	}

	// please make sure that temporary resource has been cleanup or registered
	// for cleanup before calling shutdown
	s.shutdown.Shutdown()

	return empty, nil
}

func (s *Service) forward(ctx context.Context, publisher shim.Publisher) {
	ns, _ := namespaces.Namespace(ctx)
	ctx = namespaces.WithNamespace(context.Background(), ns)
	for e := range s.events {
		err := publisher.Publish(ctx, runc.GetTopic(e), e)
		if err != nil {
			logrus.WithError(err).Error("post event")
		}
	}
	publisher.Close()
}

func (s *Service) getContainer(id string) (*runc.Container, error) {
	s.mu.Lock()
	container := s.containers[id]
	s.mu.Unlock()
	if container == nil {
		return nil, errdefs.ToGRPCf(errdefs.ErrNotFound, "container not created")
	}
	return container, nil
}

// initialize a single epoll fd to manage our consoles. `initPlatform` should
// only be called once.
func (s *Service) initPlatform() error {
	if s.platform != nil {
		return nil
	}
	p, err := runc.NewPlatform()
	if err != nil {
		return err
	}
	s.platform = p
	s.shutdown.RegisterCallback(func(context.Context) error { return s.platform.Close() })
	return nil
}
func (s *Service) send(evt interface{}) {
	s.events <- evt
}

// ResizePty of a process
func (s *Service) ResizePty(ctx context.Context, r *taskAPI.ResizePtyRequest) (*ptypes.Empty, error) {
	container, err := s.getContainer(r.ID)
	if err != nil {
		return nil, err
	}
	if err := container.ResizePty(ctx, r); err != nil {
		return nil, errdefs.ToGRPC(err)
	}
	return empty, nil
}

func NewTaskService(ctx context.Context, publisher shim.Publisher, sd shutdown.Service) (taskAPI.TaskService, error) {
	var (
		ep  oom.Watcher
		err error
	)
	if cgroups.Mode() == cgroups.Unified { // v2
		ep, err = oomv2.New(publisher)
	} else {
		ep, err = oomv1.New(publisher) // v1
	}
	if err != nil {
		return nil, err
	}
	go ep.Run(ctx)
	s := &Service{
		context:         ctx,
		events:          make(chan interface{}, 128),
		ec:              reaper.Default.Subscribe(),
		ep:              ep,
		shutdown:        sd,
		containers:      make(map[string]*runc.Container),
		running:         make(map[int][]containerProcess),
		pendingExecs:    make(map[*runc.Container]int),
		exitSubscribers: make(map[*map[int][]reaper.Exit]struct{}),
	}
	go s.processExits() // runc 的退出信号
	if err := s.initPlatform(); err != nil {
		return nil, fmt.Errorf("failed to initialized platform behavior: %w", err)
	}
	go s.forward(ctx, publisher)
	sd.RegisterCallback(func(context.Context) error {
		close(s.events)
		return nil
	})
	// ///run/containerd/s/1c2bf84c6529ba17d8234a68f92557fb5c1e4214eb17b724580e60b640e4f68a
	if address, err := shim.ReadAddress("address"); err == nil {
		sd.RegisterCallback(func(context.Context) error {
			return shim.RemoveSocket(address)
		})
	}
	return s, nil
}

// Service is the shim implementation of a remote shim over GRPC
type Service struct {
	mu sync.Mutex

	context  context.Context
	events   chan interface{}
	platform stdio.Platform
	ec       chan reaper.Exit
	ep       oom.Watcher

	containers map[string]*runc.Container

	lifecycleMu  sync.Mutex
	running      map[int][]containerProcess // pid -> running process, guarded by lifecycleMu
	pendingExecs map[*runc.Container]int    // container -> num pending execs, guarded by lifecycleMu
	// Subscriptions to exits for PIDs. Adding/deleting subscriptions and
	// dereferencing the subscription pointers must only be done while holding
	// lifecycleMu.
	exitSubscribers map[*map[int][]reaper.Exit]struct{}

	shutdown shutdown.Service
}

type containerProcess struct {
	Container *runc.Container
	Process   process.Process
}

// preStart准备启动一个容器进程并处理它的退出。
// 当启动已经创建的容器的容器初始化进程时，正在启动的容器应该作为c传入。在创建容器或启动exec时，C应该为nil。
//
// 返回的handleStarted闭包记录进程已经启动，以便有效地处理其退出。如果进程已经退出，它将立即处理退出。另外，如果这个进程是exec，并且它的容器的init进程已经退出，那么这个退出也会被处理。
// handleStarted应该在宣布进程开始的事件被发布后调用。注意，在调用handleStarted时不能持有s.r ecycyclemu。
//
// 返回的清理闭包释放用于处理早期退出的资源。
// 它必须在preStart的调用者返回之前被调用，否则会发生严重的内存泄漏。
func (s *Service) preStart(c *runc.Container) (handleStarted func(*runc.Container, process.Process), cleanup func()) {
	exits := make(map[int][]reaper.Exit)
	s.exitSubscribers[&exits] = struct{}{}

	if c != nil {
		// Remove container init process from s.running so it will once again be
		// treated as an early exit if it exits before handleStarted is called.
		pid := c.Pid()
		var newRunning []containerProcess
		for _, cp := range s.running[pid] {
			if cp.Container != c {
				newRunning = append(newRunning, cp)
			}
		}
		if len(newRunning) > 0 {
			s.running[pid] = newRunning
		} else {
			delete(s.running, pid)
		}
	}

	handleStarted = func(c *runc.Container, p process.Process) {
		var pid int
		if p != nil {
			pid = p.Pid()
		}

		_, init := p.(*process.Init)
		s.lifecycleMu.Lock()

		var initExits []reaper.Exit
		var initCps []containerProcess
		if !init {
			s.pendingExecs[c]--

			initPid := c.Pid()
			iExits, initExited := exits[initPid]
			if initExited && s.pendingExecs[c] == 0 {
				// c's init process has exited before handleStarted was called and
				// this is the last pending exec process start - we need to process
				// the exit for the init process after processing this exec, so:
				// - delete c from the s.pendingExecs map
				// - keep the exits for the init pid to process later (after we process
				// this exec's exits)
				// - get the necessary containerProcesses for the init process (that we
				// need to process the exits), and remove them from s.running (which we skipped
				// doing in processExits).
				delete(s.pendingExecs, c)
				initExits = iExits
				var skipped []containerProcess
				for _, initPidCp := range s.running[initPid] {
					if initPidCp.Container == c {
						initCps = append(initCps, initPidCp)
					} else {
						skipped = append(skipped, initPidCp)
					}
				}
				if len(skipped) == 0 {
					delete(s.running, initPid)
				} else {
					s.running[initPid] = skipped
				}
			}
		}

		ees, exited := exits[pid]
		delete(s.exitSubscribers, &exits)
		exits = nil
		if pid == 0 || exited {
			s.lifecycleMu.Unlock()
			for _, ee := range ees {
				s.handleProcessExit(ee, c, p)
			}
			for _, eee := range initExits {
				for _, cp := range initCps {
					s.handleProcessExit(eee, cp.Container, cp.Process)
				}
			}
		} else {
			// Process start was successful, add to `s.running`.
			s.running[pid] = append(s.running[pid], containerProcess{
				Container: c,
				Process:   p,
			})
			s.lifecycleMu.Unlock()
		}
	}

	cleanup = func() {
		if exits != nil {
			s.lifecycleMu.Lock()
			defer s.lifecycleMu.Unlock()
			delete(s.exitSubscribers, &exits)
		}
	}

	return handleStarted, cleanup
}

// Create a new initial process and container with the underlying OCI runtime
func (s *Service) Create(ctx context.Context, r *taskAPI.CreateTaskRequest) (_ *taskAPI.CreateTaskResponse, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lifecycleMu.Lock()
	handleStarted, cleanup := s.preStart(nil)
	s.lifecycleMu.Unlock()
	defer cleanup()

	container, err := runc.NewContainer(ctx, s.platform, r)
	if err != nil {
		return nil, err
	}

	s.containers[r.ID] = container

	s.send(&eventstypes.TaskCreate{
		ContainerID: r.ID,
		Bundle:      r.Bundle,
		Rootfs:      r.Rootfs,
		IO: &eventstypes.TaskIO{
			Stdin:    r.Stdin,
			Stdout:   r.Stdout,
			Stderr:   r.Stderr,
			Terminal: r.Terminal,
		},
		Checkpoint: r.Checkpoint,
		Pid:        uint32(container.Pid()),
	})

	// The following line cannot return an error as the only state in which that
	// could happen would also cause the container.Pid() call above to
	// nil-deference panic.
	proc, _ := container.Process("")
	handleStarted(container, proc)

	return &taskAPI.CreateTaskResponse{
		Pid: uint32(container.Pid()),
	}, nil
}

func (s *Service) RegisterTTRPC(server *ttrpc.Server) error {
	taskAPI.RegisterTaskService(server, s)
	return nil
}

func (s *Service) Stats(ctx context.Context, r *taskAPI.StatsRequest) (*taskAPI.StatsResponse, error) {
	container, err := s.getContainer(r.ID)
	if err != nil {
		return nil, err
	}
	cgx := container.Cgroup()
	if cgx == nil {
		return nil, errdefs.ToGRPCf(errdefs.ErrNotFound, "cgroup does not exist")
	}
	var statsx interface{}
	switch cg := cgx.(type) {
	case cgroup1.Cgroup:
		stats, err := cg.Stat(cgroup1.IgnoreNotExist)
		if err != nil {
			return nil, err
		}
		statsx = stats
	case *cgroupsv2.Manager:
		stats, err := cg.Stat()
		if err != nil {
			return nil, err
		}
		statsx = stats
	default:
		return nil, errdefs.ToGRPCf(errdefs.ErrNotImplemented, "unsupported cgroup type %T", cg)
	}
	data, err := typeurl.MarshalAny(statsx)
	if err != nil {
		return nil, err
	}
	return &taskAPI.StatsResponse{
		Stats: protobuf.FromAny(data),
	}, nil
}

func (s *Service) processExits() {
	for e := range s.ec {
		// While unlikely, it is not impossible for a container process to exit
		// and have its PID be recycled for a new container process before we
		// have a chance to process the first exit. As we have no way to tell
		// for sure which of the processes the exit event corresponds to (until
		// pidfd support is implemented) there is no way for us to handle the
		// exit correctly in that case.

		s.lifecycleMu.Lock()
		// Inform any concurrent s.Start() calls so they can handle the exit
		// if the PID belongs to them.
		for subscriber := range s.exitSubscribers {
			(*subscriber)[e.Pid] = append((*subscriber)[e.Pid], e)
		}
		// Handle the exit for a created/started process. If there's more than
		// one, assume they've all exited. One of them will be the correct
		// process.
		var cps, skipped []containerProcess
		for _, cp := range s.running[e.Pid] {
			_, init := cp.Process.(*process.Init)
			if init && s.pendingExecs[cp.Container] != 0 {
				// This exit relates to a container for which we have pending execs. In
				// order to ensure order between execs and the init process for a given
				// container, skip processing this exit here and let the `handleStarted`
				// closure for the pending exec publish it.
				skipped = append(skipped, cp)
			} else {
				cps = append(cps, cp)
			}
		}
		if len(skipped) > 0 {
			s.running[e.Pid] = skipped
		} else {
			delete(s.running, e.Pid)
		}
		s.lifecycleMu.Unlock()

		for _, cp := range cps {
			s.handleProcessExit(e, cp.Container, cp.Process)
		}
	}
}

// s.mu must be locked when calling handleProcessExit
func (s *Service) handleProcessExit(e reaper.Exit, c *runc.Container, p process.Process) {
	if ip, ok := p.(*process.Init); ok {
		// Ensure all children are killed
		if runc.ShouldKillAllOnExit(s.context, c.Bundle) {
			if err := ip.KillAll(s.context); err != nil {
				logrus.WithError(err).WithField("id", ip.ID()).
					Error("failed to kill init's children")
			}
		}
	}

	p.SetExited(e.Status)
	s.send(&eventstypes.TaskExit{
		ContainerID: c.ID,
		ID:          p.ID(),
		Pid:         uint32(e.Pid),
		ExitStatus:  uint32(e.Status),
		ExitedAt:    protobuf.ToTimestamp(p.ExitedAt()),
	})
}

func (s *Service) getContainerPids(ctx context.Context, container *runc.Container) ([]uint32, error) {
	p, err := container.Process("")
	if err != nil {
		return nil, errdefs.ToGRPC(err)
	}
	ps, err := p.(*process.Init).Runtime().Ps(ctx, container.ID)
	if err != nil {
		return nil, err
	}
	pids := make([]uint32, 0, len(ps))
	for _, pid := range ps {
		pids = append(pids, uint32(pid))
	}
	return pids, nil
}

// Pause the container
func (s *Service) Pause(ctx context.Context, r *taskAPI.PauseRequest) (*ptypes.Empty, error) {
	container, err := s.getContainer(r.ID)
	if err != nil {
		return nil, err
	}
	if err := container.Pause(ctx); err != nil {
		return nil, errdefs.ToGRPC(err)
	}
	s.send(&eventstypes.TaskPaused{
		ContainerID: container.ID,
	})
	return empty, nil
}

// Resume the container
func (s *Service) Resume(ctx context.Context, r *taskAPI.ResumeRequest) (*ptypes.Empty, error) {
	container, err := s.getContainer(r.ID)
	if err != nil {
		return nil, err
	}
	if err := container.Resume(ctx); err != nil {
		return nil, errdefs.ToGRPC(err)
	}
	s.send(&eventstypes.TaskResumed{
		ContainerID: container.ID,
	})
	return empty, nil
}

// Delete the initial process and container
func (s *Service) Delete(ctx context.Context, r *taskAPI.DeleteRequest) (*taskAPI.DeleteResponse, error) {
	container, err := s.getContainer(r.ID)
	if err != nil {
		return nil, err
	}
	p, err := container.Delete(ctx, r)
	if err != nil {
		return nil, errdefs.ToGRPC(err)
	}
	// if we deleted an init task, send the task delete event
	if r.ExecID == "" {
		s.mu.Lock()
		delete(s.containers, r.ID)
		s.mu.Unlock()
		s.send(&eventstypes.TaskDelete{
			ContainerID: container.ID,
			Pid:         uint32(p.Pid()),
			ExitStatus:  uint32(p.ExitStatus()),
			ExitedAt:    protobuf.ToTimestamp(p.ExitedAt()),
		})
	}
	return &taskAPI.DeleteResponse{
		ExitStatus: uint32(p.ExitStatus()),
		ExitedAt:   protobuf.ToTimestamp(p.ExitedAt()),
		Pid:        uint32(p.Pid()),
	}, nil
}
