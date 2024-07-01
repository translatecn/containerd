package shim

import (
	"context"
	"demo/pkg/defaults"
	"demo/pkg/my_mk"
	"demo/pkg/sys"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

const (
	shimBinaryFormat = "containerd-shim-%s-%s"
	socketPathLimit  = 106
)

func getSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid: true,
	}
}

// AdjustOOMScore sets the OOM score for the process to the parents OOM score +1
// to ensure that they parent has a lower* score than the shim
// if not already at the maximum OOM Score
func AdjustOOMScore(pid int) error {
	parent := os.Getppid()
	score, err := sys.GetOOMScoreAdj(parent)
	if err != nil {
		return fmt.Errorf("get parent OOM score: %w", err)
	}
	shimScore := score + 1
	if err := sys.AdjustOOMScore(pid, shimScore); err != nil {
		return fmt.Errorf("set shim OOM score: %w", err)
	}
	return nil
}

const socketRoot = defaults.DefaultStateDir

// SocketAddress returns a socket address
func SocketAddress(ctx context.Context, socketPath, id string) (string, error) {
	//ns, err := namespaces.NamespaceRequired(ctx)
	//if err != nil {
	//	return "", err
	//}
	//d := sha256.Sum256([]byte(filepath.Join(socketPath, ns, id)))
	// //run/containerd/s/1c2bf84c6529ba17d8234a68f92557fb5c1e4214eb17b724580e60b640e4f68a
	return fmt.Sprintf("unix://%s/%s", filepath.Join(socketRoot, "s"), id), nil
}

// AnonDialer returns a dialer for a socket
func AnonDialer(address string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("unix", socket(address).path(), timeout)
}

// AnonReconnectDialer returns a dialer for an existing socket on reconnection
func AnonReconnectDialer(address string, timeout time.Duration) (net.Conn, error) {
	return AnonDialer(address, timeout)
}

// NewSocket returns a new socket
func NewSocket(address string) (*net.UnixListener, error) {
	var (
		sock       = socket(address)
		path       = sock.path()
		isAbstract = sock.isAbstract()
		perm       = os.FileMode(0600)
	)

	// Darwin needs +x to access socket, otherwise it'll fail with "bind: permission denied" when running as non-root.
	if runtime.GOOS == "darwin" {
		perm = 0700
	}

	if !isAbstract {
		if err := my_mk.MkdirAll(filepath.Dir(path), perm); err != nil {
			return nil, fmt.Errorf("mkdir failed for %s: %w", path, err)
		}
	}
	l, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}

	if !isAbstract {
		if err := os.Chmod(path, perm); err != nil {
			os.Remove(sock.path())
			l.Close()
			return nil, fmt.Errorf("chmod failed for %s: %w", path, err)
		}
	}

	return l.(*net.UnixListener), nil
}

const abstractSocketPrefix = "\x00"

type socket string

func (s socket) isAbstract() bool {
	return !strings.HasPrefix(string(s), "unix://")
}

func (s socket) path() string {
	path := strings.TrimPrefix(string(s), "unix://")
	// if there was no trim performed, we assume an abstract socket
	if len(path) == len(s) {
		path = abstractSocketPrefix + path
	}
	return path
}

// RemoveSocket removes the socket at the specified address if
// it exists on the filesystem
func RemoveSocket(address string) error {
	sock := socket(address)
	if !sock.isAbstract() {
		return os.Remove(sock.path())
	}
	return nil
}

// SocketEaddrinuse returns true if the provided error is caused by the
// EADDRINUSE error number
func SocketEaddrinuse(err error) bool {
	netErr, ok := err.(*net.OpError)
	if !ok {
		return false
	}
	if netErr.Op != "listen" {
		return false
	}
	syscallErr, ok := netErr.Err.(*os.SyscallError)
	if !ok {
		return false
	}
	errno, ok := syscallErr.Err.(syscall.Errno)
	if !ok {
		return false
	}
	return errno == syscall.EADDRINUSE
}

// CanConnect returns true if the socket provided at the address
// is accepting new connections
func CanConnect(address string) bool {
	conn, err := AnonDialer(address, 100*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}