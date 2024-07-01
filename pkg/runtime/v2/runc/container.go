package runc

import (
	"context"
	"demo/config/runc"
	"demo/pkg/api/runtime/task/v2"
	"demo/pkg/console"
	"demo/pkg/mount"
	"demo/pkg/my_mk"
	"demo/pkg/namespaces"
	process3 "demo/pkg/process"
	stdio2 "demo/pkg/stdio"
	"demo/pkg/typeurl/v2"
	"demo/pkg/write"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"demo/pkg/cgroups/v3"
	"demo/pkg/cgroups/v3/cgroup1"
	cgroupsv2 "demo/pkg/cgroups/v3/cgroup2"
	"demo/pkg/errdefs"
	"github.com/sirupsen/logrus"
)

const optionsFilename = "options.json"

// ReadOptions reads the option information from the path.
// When the file does not exist, ReadOptions returns nil without an error.
func ReadOptions(path string) (*runc.Options, error) {
	filePath := filepath.Join(path, optionsFilename)
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var opts runc.Options
	if err := json.Unmarshal(data, &opts); err != nil {
		return nil, err
	}
	return &opts, nil
}

// WriteOptions writes the options information into the path
func WriteOptions(path string, opts *runc.Options) error {
	data, err := json.Marshal(opts)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(path, optionsFilename), data, 0600)
}

// ReadRuntime reads the runtime information from the path
func ReadRuntime(path string) (string, error) {
	data, err := os.ReadFile(filepath.Join(path, "runtime"))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteRuntime writes the runtime information into the path
func WriteRuntime(path, runtime string) error {
	return os.WriteFile(filepath.Join(path, "runtime"), []byte(runtime), 0600)
}

// Container for operating on a runc container and its processes
type Container struct {
	mu sync.Mutex

	// ID of the container
	ID string
	// Bundle path
	Bundle string

	// cgroup is either cgroups.Cgroup or *cgroupsv2.Manager
	cgroup          interface{}
	process         process3.Process
	processes       map[string]process3.Process
	reservedProcess map[string]struct{}
}

// All processes in the container
func (c *Container) All() (o []process3.Process) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, p := range c.processes {
		o = append(o, p)
	}
	if c.process != nil {
		o = append(o, c.process)
	}
	return o
}

// ExecdProcesses added to the container
func (c *Container) ExecdProcesses() (o []process3.Process) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, p := range c.processes {
		o = append(o, p)
	}
	return o
}

// Pid of the main process of a container
func (c *Container) Pid() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.process.Pid()
}

// Cgroup of the container
func (c *Container) Cgroup() interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cgroup
}

// CgroupSet sets the cgroup to the container
func (c *Container) CgroupSet(cg interface{}) {
	c.mu.Lock()
	c.cgroup = cg
	c.mu.Unlock()
}

// Process returns the process by id
func (c *Container) Process(id string) (process3.Process, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if id == "" {
		if c.process == nil {
			return nil, fmt.Errorf("container must be created: %w", errdefs.ErrFailedPrecondition)
		}
		return c.process, nil
	}
	p, ok := c.processes[id]
	if !ok {
		return nil, fmt.Errorf("process does not exist %s: %w", id, errdefs.ErrNotFound)
	}
	return p, nil
}

// ReserveProcess checks for the existence of an id and atomically
// reserves the process id if it does not already exist
//
// Returns true if the process id was successfully reserved and a
// cancel func to release the reservation
func (c *Container) ReserveProcess(id string) (bool, func()) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.processes[id]; ok {
		return false, nil
	}
	if _, ok := c.reservedProcess[id]; ok {
		return false, nil
	}
	c.reservedProcess[id] = struct{}{}
	return true, func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		delete(c.reservedProcess, id)
	}
}

// ProcessAdd adds a new process to the container
func (c *Container) ProcessAdd(process process3.Process) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.reservedProcess, process.ID())
	c.processes[process.ID()] = process
}

// ProcessRemove removes the process by id from the container
func (c *Container) ProcessRemove(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.processes, id)
}

// Delete the container or a process by id
func (c *Container) Delete(ctx context.Context, r *task.DeleteRequest) (process3.Process, error) {
	p, err := c.Process(r.ExecID)
	if err != nil {
		return nil, err
	}
	if err := p.Delete(ctx); err != nil {
		return nil, err
	}
	if r.ExecID != "" {
		c.ProcessRemove(r.ExecID)
	}
	return p, nil
}

// Exec an additional process
func (c *Container) Exec(ctx context.Context, r *task.ExecProcessRequest) (process3.Process, error) {
	process, err := c.process.(*process3.Init).Exec(ctx, c.Bundle, &process3.ExecConfig{
		ID:       r.ExecID,
		Terminal: r.Terminal,
		Stdin:    r.Stdin,
		Stdout:   r.Stdout,
		Stderr:   r.Stderr,
		Spec:     r.Spec,
	})
	if err != nil {
		return nil, err
	}
	c.ProcessAdd(process)
	return process, nil
}

// Kill a process
func (c *Container) Kill(ctx context.Context, r *task.KillRequest) error {
	p, err := c.Process(r.ExecID)
	if err != nil {
		return err
	}
	return p.Kill(ctx, r.Signal, r.All)
}

// CloseIO of a process
func (c *Container) CloseIO(ctx context.Context, r *task.CloseIORequest) error {
	p, err := c.Process(r.ExecID)
	if err != nil {
		return err
	}
	if stdin := p.Stdin(); stdin != nil {
		if err := stdin.Close(); err != nil {
			return fmt.Errorf("close stdin: %w", err)
		}
	}
	return nil
}

// Checkpoint the container
func (c *Container) Checkpoint(ctx context.Context, r *task.CheckpointTaskRequest) error {
	p, err := c.Process("")
	if err != nil {
		return err
	}

	var opts runc.CheckpointOptions
	if r.Options != nil {
		if err := typeurl.UnmarshalTo(r.Options, &opts); err != nil {
			return err
		}
	}
	return p.(*process3.Init).Checkpoint(ctx, &process3.CheckpointConfig{
		Path:                     r.Path,
		Exit:                     opts.Exit,
		AllowOpenTCP:             opts.OpenTcp,
		AllowExternalUnixSockets: opts.ExternalUnixSockets,
		AllowTerminal:            opts.Terminal,
		FileLocks:                opts.FileLocks,
		EmptyNamespaces:          opts.EmptyNamespaces,
		WorkDir:                  opts.WorkPath,
	})
}

// Update the resource information of a running container
func (c *Container) Update(ctx context.Context, r *task.UpdateTaskRequest) error {
	p, err := c.Process("")
	if err != nil {
		return err
	}
	return p.(*process3.Init).Update(ctx, r.Resources)
}

// HasPid returns true if the container owns a specific pid
func (c *Container) HasPid(pid int) bool {
	if c.Pid() == pid {
		return true
	}
	for _, p := range c.All() {
		if p.Pid() == pid {
			return true
		}
	}
	return false
}

func newInit(ctx context.Context, path, workDir, namespace string, platform stdio2.Platform,
	r *process3.CreateConfig, options *runc.Options, rootfs string) (*process3.Init, error) {
	runtime := process3.NewRunc(options.Root, path, namespace, options.BinaryName, options.SystemdCgroup)
	write.WriteLock.Lock()
	defer write.WriteLock.Unlock()
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

func NewContainer(ctx context.Context, platform stdio2.Platform, r *task.CreateTaskRequest) (_ *Container, retErr error) {
	ns, err := namespaces.NamespaceRequired(ctx)
	if err != nil {
		return nil, fmt.Errorf("create namespace: %w", err)
	}

	opts := &runc.Options{}
	if r.Options.GetValue() != nil {
		v, err := typeurl.UnmarshalAny(r.Options)
		if err != nil {
			return nil, err
		}
		if v != nil {
			opts = v.(*runc.Options)
		}
	}

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
		if err := my_mk.Mkdir(rootfs, 0711); err != nil && !os.IsExist(err) {
			return nil, err
		}
	}

	config := &process3.CreateConfig{
		ID:               r.ID,
		Bundle:           r.Bundle,
		Runtime:          opts.BinaryName,
		Rootfs:           pmounts,
		Terminal:         r.Terminal,
		Stdin:            r.Stdin,
		Stdout:           r.Stdout,
		Stderr:           r.Stderr,
		Checkpoint:       r.Checkpoint,
		ParentCheckpoint: r.ParentCheckpoint,
		Options:          r.Options,
	}

	if err := WriteOptions(r.Bundle, opts); err != nil {
		return nil, err
	}
	// For historical reason, we write opts.BinaryName as well as the entire opts
	if err := WriteRuntime(r.Bundle, opts.BinaryName); err != nil {
		return nil, err
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
		if retErr != nil {
			if err := mount.UnmountMounts(mounts, rootfs, 0); err != nil {
				logrus.WithError(err).Warn("failed to cleanup rootfs mount")
			}
		}
	}()
	if err := mount.All(mounts, rootfs); err != nil {
		return nil, fmt.Errorf("failed to mount rootfs component: %w", err)
	}

	p, err := newInit(
		ctx,
		r.Bundle,
		filepath.Join(r.Bundle, "work"),
		ns,
		platform,
		config,
		opts,
		rootfs,
	)
	if err != nil {
		return nil, errdefs.ToGRPC(err)
	}
	if err := p.Create(ctx, config); err != nil {
		return nil, errdefs.ToGRPC(err)
	}
	container := &Container{
		ID:              r.ID,
		Bundle:          r.Bundle,
		process:         p,
		processes:       make(map[string]process3.Process),
		reservedProcess: make(map[string]struct{}),
	}
	pid := p.Pid()
	if pid > 0 {
		var cg interface{}
		if cgroups.Mode() == cgroups.Unified { // v1
			g, err := cgroupsv2.PidGroupPath(pid)
			if err != nil {
				logrus.WithError(err).Errorf("loading cgroup2 for %d", pid)
				return container, nil
			}
			cg, err = cgroupsv2.Load(g)
			if err != nil {
				logrus.WithError(err).Errorf("loading cgroup2 for %d", pid)
			}
		} else {
			cg, err = cgroup1.Load(cgroup1.PidPath(pid))
			if err != nil {
				logrus.WithError(err).Errorf("loading cgroup for %d", pid)
			}
		}
		container.cgroup = cg
	}
	return container, nil
}

// ResizePty of a process
func (c *Container) ResizePty(ctx context.Context, r *task.ResizePtyRequest) error {
	p, err := c.Process(r.ExecID)
	if err != nil {
		return err
	}
	ws := console.WinSize{
		Width:  uint16(r.Width),
		Height: uint16(r.Height),
	}
	return p.Resize(ws)
}

// Start a container process
func (c *Container) Start(ctx context.Context, r *task.StartRequest) (process3.Process, error) {
	p, err := c.Process(r.ExecID)
	if err != nil {
		return nil, err
	}
	if err := p.Start(ctx); err != nil {
		return p, err
	}
	if c.Cgroup() == nil && p.Pid() > 0 {
		var cg interface{}
		if cgroups.Mode() == cgroups.Unified {
			g, err := cgroupsv2.PidGroupPath(p.Pid())
			if err != nil {
				logrus.WithError(err).Errorf("loading cgroup2 for %d", p.Pid())
			}
			cg, err = cgroupsv2.Load(g)
			if err != nil {
				logrus.WithError(err).Errorf("loading cgroup2 for %d", p.Pid())
			}
		} else {
			cg, err = cgroup1.Load(cgroup1.PidPath(p.Pid()))
			if err != nil {
				logrus.WithError(err).Errorf("loading cgroup for %d", p.Pid())
			}
		}
		c.cgroup = cg
	}
	return p, nil
}

// Pause the container
func (c *Container) Pause(ctx context.Context) error {
	return c.process.(*process3.Init).Pause(ctx)
}

// Resume the container
func (c *Container) Resume(ctx context.Context) error {
	return c.process.(*process3.Init).Resume(ctx)
}
