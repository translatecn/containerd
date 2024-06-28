package manager

import (
	"context"
	runcconfig "demo/config/runc"
	"demo/others/cgroups/v3"
	"demo/others/cgroups/v3/cgroup1"
	cgroupsv2 "demo/others/cgroups/v3/cgroup2"
	runcC "demo/others/go-runc"
	"demo/pkg/drop"
	"demo/pkg/log"
	"demo/pkg/mount"
	"demo/pkg/namespaces"
	"demo/pkg/oci"
	"demo/pkg/plugins/shim/shim"
	"demo/pkg/process"
	"demo/pkg/runtime/v2/runc"
	"demo/pkg/schedcore"
	"demo/pkg/write"
	"encoding/json"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"syscall"
	"time"
)

// NewShimManager returns an implementation of the shim manager
// using runc
func NewShimManager(name string) shim.Manager {
	return &manager{
		name: name,
	}
}

// 分组标签指定shim如何分组服务。
// 当前支持runc.v2专用的.group标签和标准k8s pod标签。这个列表的顺序很重要
var groupLabels = []string{
	"io.containerd.runc.v2.group",
	"io.kubernetes.cri.sandbox-id",
}

// spec is a shallow version of [oci.Spec] containing only the
// fields we need for the hook. We use a shallow struct to reduce
// the overhead of unmarshaling.
type spec struct {
	// Annotations contains arbitrary metadata for the container.
	Annotations map[string]string `json:"annotations,omitempty"`
}

type manager struct {
	name string
}

func readSpec() (*spec, error) {
	f, err := os.Open(oci.ConfigFilename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var s spec
	if err := json.NewDecoder(f).Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (m manager) Name() string {
	return m.name
}

func (manager) Stop(ctx context.Context, id string) (shim.StopStatus, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return shim.StopStatus{}, err
	}

	path := filepath.Join(filepath.Dir(cwd), id)
	ns, err := namespaces.NamespaceRequired(ctx)
	if err != nil {
		return shim.StopStatus{}, err
	}
	runtime, err := runc.ReadRuntime(path)
	if err != nil {
		return shim.StopStatus{}, err
	}
	opts, err := runc.ReadOptions(path)
	if err != nil {
		return shim.StopStatus{}, err
	}
	root := process.RuncRoot
	if opts != nil && opts.Root != "" {
		root = opts.Root
	}

	r := process.NewRunc(root, path, ns, runtime, false)
	if err := r.Delete(ctx, id, &runcC.DeleteOpts{
		Force: true,
	}); err != nil {
		log.G(ctx).WithError(err).Warn("failed to remove runc container")
	}
	if err := mount.UnmountRecursive(filepath.Join(path, "rootfs"), 0); err != nil {
		log.G(ctx).WithError(err).Warn("failed to cleanup rootfs mount")
	}
	pid, err := runcC.ReadPidFile(filepath.Join(path, process.InitPidFile))
	if err != nil {
		log.G(ctx).WithError(err).Warn("failed to read init pid file")
	}
	return shim.StopStatus{
		ExitedAt:   time.Now(),
		ExitStatus: 128 + int(unix.SIGKILL),
		Pid:        pid,
	}, nil
}

func newCommand(ctx context.Context, id, containerdAddress, containerdTTRPCAddress string, debug bool) (*exec.Cmd, error) {
	ns, err := namespaces.NamespaceRequired(ctx)
	if err != nil {
		return nil, err
	}
	self, err := os.Executable()
	if err != nil {
		return nil, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	args := []string{
		"-namespace", ns,
		"-id", id,
		"-address", containerdAddress,
	}
	if debug {
		args = append(args, "-debug")
	}
	// 第二次
	//x := []string{"--listen=:32345", "--headless=true", "--api-version=2", "--accept-multiclient", "exec", self, "--"}
	//  /Users/acejilam/Desktop/containerd/containerd-shim-runc-v2 -namespace k8s.io
	//  -address /run/containerd/containerd.sock -id f56fc531a7713ebd6a0ecea8024a55e895094f7138cc2344b0fc341ddb43b6cf
	//cmd := exec.Command("dlv", append(x, args...)...)
	cmd := exec.Command(self, args...)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), "GOMAXPROCS=4")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // 将一个进程从原来所属的进程组迁移到pgid对应的进程组
	}
	write.AppendRunLog("⚛️⚛️⚛️⚛️⚛️⚛️⚛️⚛️⚛️⚛️", "")
	write.AppendRunLog("shim: ---------->ENV: ", drop.DropEnv(cmd.Env))
	write.AppendRunLog("shim: ---------->Args: ", cmd.Args)
	write.AppendRunLog("shim: ---------->Path: ", cmd.Path)
	write.AppendRunLog("shim: ---------->Process: ", cmd.Process)
	write.AppendRunLog("shim: ---------->Dir: ", cmd.Dir)

	return cmd, nil
}
func (manager) Start(ctx context.Context, id string, opts shim.StartOpts) (_ string, retErr error) {
	cmd, err := newCommand(ctx, id, opts.Address, opts.TTRPCAddress, opts.Debug)
	if err != nil {
		return "", err
	}
	grouping := id // 容器ID
	spec, err := readSpec()
	if err != nil {
		return "", err
	}
	for _, group := range groupLabels {
		if groupID, ok := spec.Annotations[group]; ok {
			grouping = groupID
			break
		}
	}
	address, err := shim.SocketAddress(ctx, opts.Address, grouping)
	if err != nil {
		return "", err
	}

	socket, err := shim.NewSocket(address)
	// /run/containerd/s/1c2bf84c6529ba17d8234a68f92557fb5c1e4214eb17b724580e60b640e4f68a
	if err != nil {
		// the only time where this would happen is if there is a bug and the socket
		// was not cleaned up in the cleanup method of the shim or we are using the
		// grouping functionality where the new process should be run with the same
		// shim as an existing container
		if !shim.SocketEaddrinuse(err) {
			return "", fmt.Errorf("create new shim socket: %w", err)
		}
		if shim.CanConnect(address) {
			if err := shim.WriteAddress("address", address); err != nil {
				return "", fmt.Errorf("write existing socket for shim: %w", err)
			}
			return address, nil
		}
		if err := shim.RemoveSocket(address); err != nil {
			return "", fmt.Errorf("remove pre-existing socket: %w", err)
		}
		if socket, err = shim.NewSocket(address); err != nil {
			return "", fmt.Errorf("try create new shim socket 2x: %w", err)
		}
	}
	defer func() {
		if retErr != nil {
			socket.Close()
			_ = shim.RemoveSocket(address)
		}
	}()

	// make sure that reexec shim-v2 binary use the value if need
	if err := shim.WriteAddress("address", address); err != nil {
		return "", err
	}

	f, err := socket.File()
	if err != nil {
		return "", err
	}

	cmd.ExtraFiles = append(cmd.ExtraFiles, f) // 当前创建的unix  交给子进程

	goruntime.LockOSThread()
	if os.Getenv("SCHED_CORE") != "" {
		if err := schedcore.Create(schedcore.ProcessGroup); err != nil {
			return "", fmt.Errorf("enable sched core support: %w", err)
		}
	}

	if err := cmd.Start(); err != nil {
		f.Close()
		return "", err
	}

	goruntime.UnlockOSThread()

	defer func() {
		if retErr != nil {
			cmd.Process.Kill()
		}
	}()
	// make sure to wait after start
	go cmd.Wait()

	if opts, err := shim.ReadRuntimeOptions[*runcconfig.Options](os.Stdin); err == nil {
		if opts.ShimCgroup != "" {
			if cgroups.Mode() == cgroups.Unified {
				cg, err := cgroupsv2.Load(opts.ShimCgroup)
				if err != nil {
					return "", fmt.Errorf("failed to load cgroup %s: %w", opts.ShimCgroup, err)
				}
				if err := cg.AddProc(uint64(cmd.Process.Pid)); err != nil {
					return "", fmt.Errorf("failed to join cgroup %s: %w", opts.ShimCgroup, err)
				}
			} else {
				cg, err := cgroup1.Load(cgroup1.StaticPath(opts.ShimCgroup))
				if err != nil {
					return "", fmt.Errorf("failed to load cgroup %s: %w", opts.ShimCgroup, err)
				}
				if err := cg.AddProc(uint64(cmd.Process.Pid)); err != nil {
					return "", fmt.Errorf("failed to join cgroup %s: %w", opts.ShimCgroup, err)
				}
			}
		}
	}
	// 优先级分数 +1  ,
	if err := shim.AdjustOOMScore(cmd.Process.Pid); err != nil { // 第二次启动的shim
		return "", fmt.Errorf("failed to adjust OOM score for shim: %w", err)
	}
	return address, nil
}
