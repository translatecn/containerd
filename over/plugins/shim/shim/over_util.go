package shim

import (
	"bytes"
	"context"
	"demo/over/atomicfile"
	"demo/over/drop"
	"demo/over/namespaces"
	"demo/over/protobuf/proto"
	"demo/over/protobuf/types"
	"demo/over/typeurl/v2"
	"errors"
	"fmt"
	"github.com/fatih/color"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"demo/over/errdefs"
	"demo/over/ttrpc"
)

type CommandConfig struct {
	Runtime      string
	Address      string
	TTRPCAddress string
	Path         string
	SchedCore    bool
	Args         []string
	Opts         *types.Any
}

func Command(ctx context.Context, config *CommandConfig) (*exec.Cmd, error) {
	ns, err := namespaces.NamespaceRequired(ctx)
	if err != nil {
		return nil, err
	}
	self, err := os.Executable() // /Users/acejilam/Desktop/containerd/containerd_bin
	if err != nil {
		return nil, err
	}
	args := []string{
		"-namespace", ns,
		"-address", config.Address,
		"-publish-binary", self,
	}
	args = append(args, config.Args...)
	count := 0
	// 读取目录
	files, _ := ioutil.ReadDir("/run/containerd/io.containerd.runtime.v2.task/k8s.io")
	for _, file := range files {
		if file.IsDir() {
			count += 1
		}
	}

	// /Users/acejilam/Desktop/containerd/containerd-shim-runc-v2 -- -namespace k8s.io
	// -address /run/containerd/containerd.sock -publish-binary /Users/acejilam/Desktop/con/Users/acejilam/Desktop/containerd/containerd-shim-runc-v2
	// -- -namespace k8s.io -address /run/containerd/containerd.sock -publish-binary /Users/acejilam/Desktop/ctainerd/containerd_bin
	// -id f56fc531a7713ebd6a0ecea8024a55e895094f7138cc2344b0fc341ddb43b6cf start
	var cmd *exec.Cmd
	//if count > 1 {
	//x := []string{"--listen=:22345", "--headless=true", "--api-version=2", "--accept-multiclient", "exec", config.Runtime, "--"}
	//cmd = exec.Command("dlv", append(x, args...)...)
	//} else {
	cmd = exec.CommandContext(ctx, config.Runtime, args...) // container-shim-runc-v2
	//}
	cmd.Dir = config.Path
	cmd.Env = append(
		os.Environ(),
		"GOMAXPROCS=2",
		fmt.Sprintf("%s=2", maxVersionEnv),
		fmt.Sprintf("%s=%s", ttrpcAddressEnv, config.TTRPCAddress),
		fmt.Sprintf("%s=%s", grpcAddressEnv, config.Address),
		fmt.Sprintf("%s=%s", namespaceEnv, ns),
	)
	if config.SchedCore {
		cmd.Env = append(cmd.Env, "SCHED_CORE=1")
	}
	cmd.SysProcAttr = getSysProcAttr()
	if config.Opts != nil {
		d, err := proto.Marshal(config.Opts) // 内置了一些输入内容
		if err != nil {
			return nil, err
		}
		cmd.Stdin = bytes.NewReader(d)
	}
	if os.Getenv("DEBUG") != "" {
		color.New(color.FgRed).SetWriter(os.Stderr).Println("---------->ENV: ", drop.DropEnv(cmd.Env))
		color.New(color.FgRed).SetWriter(os.Stderr).Println("---------->Args: ", cmd.Args)
		color.New(color.FgRed).SetWriter(os.Stderr).Println("---------->Path: ", cmd.Path)
		color.New(color.FgRed).SetWriter(os.Stderr).Println("---------->Process: ", cmd.Process)
		color.New(color.FgRed).SetWriter(os.Stderr).Println("---------->Dir: ", cmd.Dir)
	}

	return cmd, nil
}

// BinaryName returns the shim binary name from the runtime name,
// empty string returns means runtime name is invalid
func BinaryName(runtime string) string {
	// runtime name should format like $prefix.name.version
	parts := strings.Split(runtime, ".")
	if len(parts) < 2 || parts[0] == "" {
		return ""
	}

	return fmt.Sprintf(shimBinaryFormat, parts[len(parts)-2], parts[len(parts)-1])
}

// BinaryPath returns the full path for the shim binary from the runtime name,
// empty string returns means runtime name is invalid
func BinaryPath(runtime string) string {
	dir := filepath.Dir(runtime)
	binary := BinaryName(runtime)

	path, err := filepath.Abs(filepath.Join(dir, binary))
	if err != nil {
		return ""
	}

	return path
}

// Connect to the provided address
func Connect(address string, d func(string, time.Duration) (net.Conn, error)) (net.Conn, error) {
	return d(address, 100*time.Second)
}

// WriteAddress writes a address file atomically
func WriteAddress(path, address string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	f, err := atomicfile.New(path, 0o644)
	if err != nil {
		return err
	}
	_, err = f.Write([]byte(address))
	if err != nil {
		f.Cancel()
		return err
	}
	return f.Close()
}

// ErrNoAddress is returned when the address file has no content
var ErrNoAddress = errors.New("no shim address")

// ReadAddress returns the shim's socket address from the path
func ReadAddress(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", ErrNoAddress
	}
	return string(data), nil
}

// ReadRuntimeOptions reads config bytes from io.Reader and unmarshals it into the provided type.
// The type must be registered with typeurl.
//
// The function will return ErrNotFound, if the config is not provided.
// And ErrInvalidArgument, if unable to cast the config to the provided type T.
func ReadRuntimeOptions[T any](reader io.Reader) (T, error) {
	var config T

	data, err := io.ReadAll(reader)
	if err != nil {
		return config, fmt.Errorf("failed to read config bytes from stdin: %w", err)
	}

	if len(data) == 0 {
		return config, errdefs.ErrNotFound
	}

	var any types.Any
	if err := proto.Unmarshal(data, &any); err != nil {
		return config, err
	}

	v, err := typeurl.UnmarshalAny(&any)
	if err != nil {
		return config, err
	}

	config, ok := v.(T)
	if !ok {
		return config, fmt.Errorf("invalid type %T: %w", v, errdefs.ErrInvalidArgument)
	}

	return config, nil
}

// chainUnaryServerInterceptors creates a single ttrpc server interceptor from
// a chain of many interceptors executed from first to last.
func chainUnaryServerInterceptors(interceptors ...ttrpc.UnaryServerInterceptor) ttrpc.UnaryServerInterceptor {
	n := len(interceptors)

	// force to use default interceptor in ttrpc
	if n == 0 {
		return nil
	}

	return func(ctx context.Context, unmarshal ttrpc.Unmarshaler, info *ttrpc.UnaryServerInfo, method ttrpc.Method) (interface{}, error) {
		currentMethod := method

		for i := n - 1; i > 0; i-- {
			interceptor := interceptors[i]
			innerMethod := currentMethod

			currentMethod = func(currentCtx context.Context, currentUnmarshal func(interface{}) error) (interface{}, error) {
				return interceptor(currentCtx, currentUnmarshal, info, innerMethod)
			}
		}
		return interceptors[0](ctx, unmarshal, info, currentMethod)
	}
}
