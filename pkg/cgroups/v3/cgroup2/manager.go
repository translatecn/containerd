package cgroup2

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"demo/pkg/cgroups/v3/cgroup2/stats"

	systemdDbus "github.com/coreos/go-systemd/v22/dbus"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	subtreeControl     = "cgroup.subtree_control"
	controllersFile    = "cgroup.controllers"
	killFile           = "cgroup.kill"
	typeFile           = "cgroup.type"
	defaultCgroup2Path = "/sys/fs/cgroup"
	defaultSlice       = "system.slice"
)

var _ bool

type Event struct {
	Low     uint64
	High    uint64
	Max     uint64
	OOM     uint64
	OOMKill uint64
}

// Resources for a cgroups v2 unified hierarchy
type Resources struct {
	CPU     *CPU
	Memory  *Memory
	Pids    *Pids
	IO      *IO
	RDMA    *RDMA
	HugeTlb *HugeTlb
	// When len(Devices) is zero, devices are not controlled
	Devices []specs.LinuxDeviceCgroup
}

// Values returns the raw filenames and values that
// can be written to the unified hierarchy
func (r *Resources) Values() (o []Value) {
	if r.CPU != nil {
		o = append(o, r.CPU.Values()...)
	}
	if r.Memory != nil {
		o = append(o, r.Memory.Values()...)
	}
	if r.Pids != nil {
		o = append(o, r.Pids.Values()...)
	}
	if r.IO != nil {
		o = append(o, r.IO.Values()...)
	}
	if r.RDMA != nil {
		o = append(o, r.RDMA.Values()...)
	}
	if r.HugeTlb != nil {
		o = append(o, r.HugeTlb.Values()...)
	}
	return o
}

// EnabledControllers returns the list of all not nil resource controllers
func (r *Resources) EnabledControllers() (c []string) {
	if r.CPU != nil {
		c = append(c, "cpu")
		if r.CPU.Cpus != "" || r.CPU.Mems != "" {
			c = append(c, "cpuset")
		}
	}
	if r.Memory != nil {
		c = append(c, "memory")
	}
	if r.Pids != nil {
		c = append(c, "pids")
	}
	if r.IO != nil {
		c = append(c, "io")
	}
	if r.RDMA != nil {
		c = append(c, "rdma")
	}
	if r.HugeTlb != nil {
		c = append(c, "hugetlb")
	}
	return
}

// Value of a cgroup setting
type Value struct {
	filename string
	value    interface{}
}

// write the value to the full, absolute path, of a unified hierarchy
func (c *Value) write(path string, perm os.FileMode) error {
	var data []byte
	switch t := c.value.(type) {
	case uint64:
		data = []byte(strconv.FormatUint(t, 10))
	case uint16:
		data = []byte(strconv.FormatUint(uint64(t), 10))
	case int64:
		data = []byte(strconv.FormatInt(t, 10))
	case []byte:
		data = t
	case string:
		data = []byte(t)
	case CPUMax:
		data = []byte(t)
	default:
		return ErrInvalidFormat
	}

	return os.WriteFile(
		filepath.Join(path, c.filename),
		data,
		perm,
	)
}

func writeValues(path string, values []Value) error {
	for _, o := range values {
		if err := o.write(path, defaultFilePerm); err != nil {
			return err
		}
	}
	return nil
}

type InitConfig struct {
	mountpoint string
}

type InitOpts func(c *InitConfig) error

// WithMountpoint sets the unified mountpoint. The default path is /sys/fs/cgroup.

// Load a cgroup.
func Load(group string, opts ...InitOpts) (*Manager, error) {
	c := InitConfig{mountpoint: defaultCgroup2Path}
	for _, opt := range opts {
		if err := opt(&c); err != nil {
			return nil, err
		}
	}

	if err := VerifyGroupPath(group); err != nil {
		return nil, err
	}
	path := filepath.Join(c.mountpoint, group)
	return &Manager{
		unifiedMountpoint: c.mountpoint,
		path:              path,
	}, nil
}

type Manager struct {
	unifiedMountpoint string
	path              string
}

func setResources(path string, resources *Resources) error {
	if resources != nil {
		if err := writeValues(path, resources.Values()); err != nil {
			return err
		}
		if err := setDevices(path, resources.Devices); err != nil {
			return err
		}
	}
	return nil
}

// CgroupType represents the types a cgroup can be.
type CgroupType string

const (
	Domain   CgroupType = "domain"
	Threaded CgroupType = "threaded"
)

func (c *Manager) GetType() (CgroupType, error) {
	val, err := os.ReadFile(filepath.Join(c.path, typeFile))
	if err != nil {
		return "", err
	}
	trimmed := strings.TrimSpace(string(val))
	return CgroupType(trimmed), nil
}

func (c *Manager) SetType(cgType CgroupType) error {
	// NOTE: We could abort if cgType != Threaded here as currently
	// it's not possible to revert back to domain, but not sure
	// it's worth being that opinionated, especially if that may
	// ever change.
	v := Value{
		filename: typeFile,
		value:    string(cgType),
	}
	return writeValues(c.path, []Value{v})
}

func (c *Manager) RootControllers() ([]string, error) {
	b, err := os.ReadFile(filepath.Join(c.unifiedMountpoint, controllersFile))
	if err != nil {
		return nil, err
	}
	return strings.Fields(string(b)), nil
}

func (c *Manager) Controllers() ([]string, error) {
	b, err := os.ReadFile(filepath.Join(c.path, controllersFile))
	if err != nil {
		return nil, err
	}
	return strings.Fields(string(b)), nil
}

func (c *Manager) Update(resources *Resources) error {
	return setResources(c.path, resources)
}

type ControllerToggle int

const (
	Enable ControllerToggle = iota + 1
	Disable
)

func toggleFunc(controllers []string, prefix string) []string {
	out := make([]string, len(controllers))
	for i, c := range controllers {
		out[i] = prefix + c
	}
	return out
}

func (c *Manager) ToggleControllers(controllers []string, t ControllerToggle) error {
	// when c.path is like /foo/bar/baz, the following files need to be written:
	// * /sys/fs/cgroup/cgroup.subtree_control
	// * /sys/fs/cgroup/foo/cgroup.subtree_control
	// * /sys/fs/cgroup/foo/bar/cgroup.subtree_control
	// Note that /sys/fs/cgroup/foo/bar/baz/cgroup.subtree_control does not need to be written.
	split := strings.Split(c.path, "/")
	var lastErr error
	for i := range split {
		f := strings.Join(split[:i], "/")
		if !strings.HasPrefix(f, c.unifiedMountpoint) || f == c.path {
			continue
		}
		filePath := filepath.Join(f, subtreeControl)
		if err := c.writeSubtreeControl(filePath, controllers, t); err != nil {
			// When running as rootless, the user may face EPERM on parent groups, but it is neglible when the
			// controller is already written.
			// So we only return the last error.
			lastErr = fmt.Errorf("failed to write subtree controllers %+v to %q: %w", controllers, filePath, err)
		} else {
			lastErr = nil
		}
	}
	return lastErr
}

func (c *Manager) writeSubtreeControl(filePath string, controllers []string, t ControllerToggle) error {
	f, err := os.OpenFile(filePath, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	switch t {
	case Enable:
		controllers = toggleFunc(controllers, "+")
	case Disable:
		controllers = toggleFunc(controllers, "-")
	}
	_, err = f.WriteString(strings.Join(controllers, " "))
	return err
}

func (c *Manager) NewChild(name string, resources *Resources) (*Manager, error) {
	if strings.HasPrefix(name, "/") {
		return nil, errors.New("name must be relative")
	}
	path := filepath.Join(c.path, name)
	if err := os.MkdirAll(path, defaultDirPerm); err != nil {
		return nil, err
	}
	m := Manager{
		unifiedMountpoint: c.unifiedMountpoint,
		path:              path,
	}
	if resources != nil {
		if err := m.ToggleControllers(resources.EnabledControllers(), Enable); err != nil {
			// clean up cgroup dir on failure
			os.Remove(path)
			return nil, err
		}
	}
	if err := setResources(path, resources); err != nil {
		// clean up cgroup dir on failure
		os.Remove(path)
		return nil, err
	}
	return &m, nil
}

func (c *Manager) AddProc(pid uint64) error {
	v := Value{
		filename: cgroupProcs,
		value:    pid,
	}
	return writeValues(c.path, []Value{v})
}

func (c *Manager) AddThread(tid uint64) error {
	v := Value{
		filename: cgroupThreads,
		value:    tid,
	}
	return writeValues(c.path, []Value{v})
}

// Kill will try to forcibly exit all of the processes in the cgroup. This is
// equivalent to sending a SIGKILL to every process. On kernels 5.14 and greater
// this will use the cgroup.kill file, on anything that doesn't have the cgroup.kill
// file, a manual process of freezing -> sending a SIGKILL to every process -> thawing
// will be used.
func (c *Manager) Kill() error {
	v := Value{
		filename: killFile,
		value:    "1",
	}
	err := writeValues(c.path, []Value{v})
	if err == nil {
		return nil
	}
	logrus.Warnf("falling back to slower kill implementation: %s", err)
	// Fallback to slow method.
	return c.fallbackKill()
}

// fallbackKill is a slower fallback to the more modern (kernels 5.14+)
// approach of writing to the cgroup.kill file. This is heavily pulled
// from runc's same approach (in signalAllProcesses), with the only differences
// being this is just tailored to the API exposed in this library, and we don't
// need to care about signals other than SIGKILL.
//
// https://demo/3rd_party/runc/blob/8da0a0b5675764feaaaaad466f6567a9983fcd08/libcontainer/init_linux.go#L523-L529
func (c *Manager) fallbackKill() error {
	if err := c.Freeze(); err != nil {
		logrus.Warn(err)
	}
	pids, err := c.Procs(true)
	if err != nil {
		if err := c.Thaw(); err != nil {
			logrus.Warn(err)
		}
		return err
	}
	var procs []*os.Process
	for _, pid := range pids {
		p, err := os.FindProcess(int(pid))
		if err != nil {
			logrus.Warn(err)
			continue
		}
		procs = append(procs, p)
		if err := p.Signal(unix.SIGKILL); err != nil {
			logrus.Warn(err)
		}
	}
	if err := c.Thaw(); err != nil {
		logrus.Warn(err)
	}

	subreaper, err := getSubreaper()
	if err != nil {
		// The error here means that PR_GET_CHILD_SUBREAPER is not
		// supported because this code might run on a kernel older
		// than 3.4. We don't want to throw an error in that case,
		// and we simplify things, considering there is no subreaper
		// set.
		subreaper = 0
	}

	for _, p := range procs {
		// In case a subreaper has been setup, this code must not
		// wait for the process. Otherwise, we cannot be sure the
		// current process will be reaped by the subreaper, while
		// the subreaper might be waiting for this process in order
		// to retrieve its exit code.
		if subreaper == 0 {
			if _, err := p.Wait(); err != nil {
				if !errors.Is(err, unix.ECHILD) {
					logrus.Warnf("wait on pid %d failed: %s", p.Pid, err)
				}
			}
		}
	}
	return nil
}

func (c *Manager) Delete() error {
	// kernel prevents cgroups with running process from being removed, check the tree is empty
	processes, err := c.Procs(true)
	if err != nil {
		return err
	}
	if len(processes) > 0 {
		return fmt.Errorf("cgroups: unable to remove path %q: still contains running processes", c.path)
	}
	return remove(c.path)
}

func (c *Manager) Procs(recursive bool) ([]uint64, error) {
	var processes []uint64
	err := filepath.Walk(c.path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !recursive && info.IsDir() {
			if p == c.path {
				return nil
			}
			return filepath.SkipDir
		}
		_, name := filepath.Split(p)
		if name != cgroupProcs {
			return nil
		}
		procs, err := parseCgroupProcsFile(p)
		if err != nil {
			return err
		}
		processes = append(processes, procs...)
		return nil
	})
	return processes, err
}

func (c *Manager) MoveTo(destination *Manager) error {
	processes, err := c.Procs(true)
	if err != nil {
		return err
	}
	for _, p := range processes {
		if err := destination.AddProc(p); err != nil {
			if strings.Contains(err.Error(), "no such process") {
				continue
			}
			return err
		}
	}
	return nil
}

func (c *Manager) Stat() (*stats.Metrics, error) {
	controllers, err := c.Controllers()
	if err != nil {
		return nil, err
	}
	// Sizing this avoids an allocation to increase the map at runtime;
	// currently the default bucket size is 8 and we put 40+ elements
	// in it so we'd always end up allocating.
	out := make(map[string]uint64, 50)
	for _, controller := range controllers {
		switch controller {
		case "cpu", "memory":
			if err := readKVStatsFile(c.path, controller+".stat", out); err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, err
			}
		}
	}
	memoryEvents := make(map[string]uint64)
	if err := readKVStatsFile(c.path, "memory.events", memoryEvents); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	var metrics stats.Metrics
	metrics.Pids = &stats.PidsStat{
		Current: getStatFileContentUint64(filepath.Join(c.path, "pids.current")),
		Limit:   getStatFileContentUint64(filepath.Join(c.path, "pids.max")),
	}
	metrics.CPU = &stats.CPUStat{
		UsageUsec:     out["usage_usec"],
		UserUsec:      out["user_usec"],
		SystemUsec:    out["system_usec"],
		NrPeriods:     out["nr_periods"],
		NrThrottled:   out["nr_throttled"],
		ThrottledUsec: out["throttled_usec"],
	}
	metrics.Memory = &stats.MemoryStat{
		Anon:                  out["anon"],
		File:                  out["file"],
		KernelStack:           out["kernel_stack"],
		Slab:                  out["slab"],
		Sock:                  out["sock"],
		Shmem:                 out["shmem"],
		FileMapped:            out["file_mapped"],
		FileDirty:             out["file_dirty"],
		FileWriteback:         out["file_writeback"],
		AnonThp:               out["anon_thp"],
		InactiveAnon:          out["inactive_anon"],
		ActiveAnon:            out["active_anon"],
		InactiveFile:          out["inactive_file"],
		ActiveFile:            out["active_file"],
		Unevictable:           out["unevictable"],
		SlabReclaimable:       out["slab_reclaimable"],
		SlabUnreclaimable:     out["slab_unreclaimable"],
		Pgfault:               out["pgfault"],
		Pgmajfault:            out["pgmajfault"],
		WorkingsetRefault:     out["workingset_refault"],
		WorkingsetActivate:    out["workingset_activate"],
		WorkingsetNodereclaim: out["workingset_nodereclaim"],
		Pgrefill:              out["pgrefill"],
		Pgscan:                out["pgscan"],
		Pgsteal:               out["pgsteal"],
		Pgactivate:            out["pgactivate"],
		Pgdeactivate:          out["pgdeactivate"],
		Pglazyfree:            out["pglazyfree"],
		Pglazyfreed:           out["pglazyfreed"],
		ThpFaultAlloc:         out["thp_fault_alloc"],
		ThpCollapseAlloc:      out["thp_collapse_alloc"],
		Usage:                 getStatFileContentUint64(filepath.Join(c.path, "memory.current")),
		UsageLimit:            getStatFileContentUint64(filepath.Join(c.path, "memory.max")),
		SwapUsage:             getStatFileContentUint64(filepath.Join(c.path, "memory.swap.current")),
		SwapLimit:             getStatFileContentUint64(filepath.Join(c.path, "memory.swap.max")),
	}
	if len(memoryEvents) > 0 {
		metrics.MemoryEvents = &stats.MemoryEvents{
			Low:     memoryEvents["low"],
			High:    memoryEvents["high"],
			Max:     memoryEvents["max"],
			Oom:     memoryEvents["oom"],
			OomKill: memoryEvents["oom_kill"],
		}
	}
	metrics.Io = &stats.IOStat{Usage: readIoStats(c.path)}
	metrics.Rdma = &stats.RdmaStat{
		Current: rdmaStats(filepath.Join(c.path, "rdma.current")),
		Limit:   rdmaStats(filepath.Join(c.path, "rdma.max")),
	}
	metrics.Hugetlb = readHugeTlbStats(c.path)

	return &metrics, nil
}

func readKVStatsFile(path string, file string, out map[string]uint64) error {
	f, err := os.Open(filepath.Join(path, file))
	if err != nil {
		return err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		name, value, err := parseKV(s.Text())
		if err != nil {
			return fmt.Errorf("error while parsing %s (line=%q): %w", filepath.Join(path, file), s.Text(), err)
		}
		out[name] = value
	}
	return s.Err()
}

func (c *Manager) Freeze() error {
	return c.freeze(c.path, Frozen)
}

func (c *Manager) Thaw() error {
	return c.freeze(c.path, Thawed)
}

func (c *Manager) freeze(path string, state State) error {
	values := state.Values()
	for {
		if err := writeValues(path, values); err != nil {
			return err
		}
		current, err := fetchState(path)
		if err != nil {
			return err
		}
		if current == state {
			return nil
		}
		time.Sleep(1 * time.Millisecond)
	}
}

func (c *Manager) isCgroupEmpty() bool {
	// In case of any error we return true so that we exit and don't leak resources
	out := make(map[string]uint64)
	if err := readKVStatsFile(c.path, "cgroup.events", out); err != nil {
		return true
	}
	if v, ok := out["populated"]; ok {
		return v == 0
	}
	return true
}

// MemoryEventFD returns inotify file descriptor and 'memory.events' inotify watch descriptor
func (c *Manager) MemoryEventFD() (int, uint32, error) {
	fpath := filepath.Join(c.path, "memory.events")
	fd, err := unix.InotifyInit()
	if err != nil {
		return 0, 0, errors.New("failed to create inotify fd")
	}
	wd, err := unix.InotifyAddWatch(fd, fpath, unix.IN_MODIFY)
	if err != nil {
		unix.Close(fd)
		return 0, 0, fmt.Errorf("failed to add inotify watch for %q: %w", fpath, err)
	}
	// monitor to detect process exit/cgroup deletion
	evpath := filepath.Join(c.path, "cgroup.events")
	if _, err = unix.InotifyAddWatch(fd, evpath, unix.IN_MODIFY); err != nil {
		unix.Close(fd)
		return 0, 0, fmt.Errorf("failed to add inotify watch for %q: %w", evpath, err)
	}

	return fd, uint32(wd), nil
}

func (c *Manager) EventChan() (<-chan Event, <-chan error) {
	ec := make(chan Event)
	errCh := make(chan error, 1)
	go c.waitForEvents(ec, errCh)

	return ec, errCh
}

func (c *Manager) waitForEvents(ec chan<- Event, errCh chan<- error) {
	defer close(errCh)

	fd, _, err := c.MemoryEventFD()
	if err != nil {
		errCh <- err
		return
	}
	defer unix.Close(fd)

	for {
		buffer := make([]byte, unix.SizeofInotifyEvent*10)
		bytesRead, err := unix.Read(fd, buffer)
		if err != nil {
			errCh <- err
			return
		}
		if bytesRead >= unix.SizeofInotifyEvent {
			out := make(map[string]uint64)
			if err := readKVStatsFile(c.path, "memory.events", out); err != nil {
				// When cgroup is deleted read may return -ENODEV instead of -ENOENT from open.
				if _, statErr := os.Lstat(filepath.Join(c.path, "memory.events")); !os.IsNotExist(statErr) {
					errCh <- err
				}
				return
			}
			ec <- Event{
				Low:     out["low"],
				High:    out["high"],
				Max:     out["max"],
				OOM:     out["oom"],
				OOMKill: out["oom_kill"],
			}
			if c.isCgroupEmpty() {
				return
			}
		}
	}
}

func setDevices(path string, devices []specs.LinuxDeviceCgroup) error {
	if len(devices) == 0 {
		return nil
	}
	insts, license, err := DeviceFilter(devices)
	if err != nil {
		return err
	}
	dirFD, err := unix.Open(path, unix.O_DIRECTORY|unix.O_RDONLY|unix.O_CLOEXEC, 0o600)
	if err != nil {
		return fmt.Errorf("cannot get dir FD for %s", path)
	}
	defer unix.Close(dirFD)
	if _, err := LoadAttachCgroupDeviceFilter(insts, license, dirFD); err != nil {
		if !canSkipEBPFError(devices) {
			return err
		}
	}
	return nil
}

// getSystemdFullPath returns the full systemd path when creating a systemd slice group.
// the reason this is necessary is because the "-" character has a special meaning in
// systemd slice. For example, when creating a slice called "my-group-112233.slice",
// systemd will create a hierarchy like this:
//
//	/sys/fs/cgroup/my.slice/my-group.slice/my-group-112233.slice

// dashesToPath converts a slice name with dashes to it's corresponding systemd filesystem path.

func (c *Manager) DeleteSystemd() error {
	ctx := context.TODO()
	conn, err := systemdDbus.NewWithContext(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	group := systemdUnitFromPath(c.path)
	ch := make(chan string)
	_, err = conn.StopUnitContext(ctx, group, "replace", ch)
	if err != nil {
		return err
	}
	<-ch
	return nil
}