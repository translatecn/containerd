package v2

import (
	"bytes"
	"context"
	"demo/pkg/api/runtime/task/v2"
	"demo/pkg/log"
	"demo/pkg/namespaces"
	shimx2 "demo/pkg/plugins/shim/shim"
	"demo/pkg/protobuf"
	"demo/pkg/protobuf/proto"
	"demo/pkg/protobuf/types"
	"demo/pkg/runtime"
	"fmt"
	"io"
	"os"
	"path/filepath"
	gruntime "runtime"
)

type ShimBinaryConfig struct {
	Runtime      string
	Address      string
	TtrpcAddress string
	SchedCore    bool
}

func ShimBinary(bundle *Bundle, config ShimBinaryConfig) *binary {
	return &binary{
		bundle:                 bundle,
		runtime:                config.Runtime,
		containerdAddress:      config.Address,
		containerdTTRPCAddress: config.TtrpcAddress,
		schedCore:              config.SchedCore,
	}
}

type binary struct {
	runtime                string
	containerdAddress      string
	containerdTTRPCAddress string
	schedCore              bool
	bundle                 *Bundle
}

func (b *binary) Delete(ctx context.Context) (*runtime.Exit, error) {
	log.G(ctx).Info("cleaning up dead shim")

	// On Windows and FreeBSD, the current working directory of the shim should
	// not be the bundle path during the delete operation. Instead, we invoke
	// with the default work dir and forward the bundle path on the cmdline.
	// Windows cannot delete the current working directory while an executable
	// is in use with it. On FreeBSD, fork/exec can fail.
	var bundlePath string
	if gruntime.GOOS != "windows" && gruntime.GOOS != "freebsd" {
		bundlePath = b.bundle.Path
	}
	args := []string{
		"-id", b.bundle.ID,
		"-bundle", b.bundle.Path,
	}
	switch log.GetLevel() {
	case log.DebugLevel, log.TraceLevel:
		args = append(args, "-debug")
	}
	args = append(args, "delete")

	cmd, err := shimx2.Command(ctx,
		&shimx2.CommandConfig{
			Runtime:      b.runtime,
			Address:      b.containerdAddress,
			TTRPCAddress: b.containerdTTRPCAddress,
			Path:         bundlePath,
			Opts:         nil,
			Args:         args,
		})

	if err != nil {
		return nil, err
	}
	var (
		out  = bytes.NewBuffer(nil)
		errb = bytes.NewBuffer(nil)
	)
	cmd.Stdout = out
	cmd.Stderr = errb
	if err := cmd.Run(); err != nil {
		log.G(ctx).WithField("cmd", cmd).WithError(err).Error("failed to delete")
		return nil, fmt.Errorf("%s: %w", errb.String(), err)
	}
	s := errb.String()
	if s != "" {
		log.G(ctx).Warnf("cleanup warnings %s", s)
	}
	var response task.DeleteResponse
	if err := proto.Unmarshal(out.Bytes(), &response); err != nil {
		return nil, err
	}
	if err := b.bundle.Delete(); err != nil {
		return nil, err
	}
	return &runtime.Exit{
		Status:    response.ExitStatus,
		Timestamp: protobuf.FromTimestamp(response.ExitedAt),
		Pid:       response.Pid,
	}, nil
}

func (b *binary) Start(ctx context.Context, opts *types.Any, onClose func()) (_ *Shim, err error) {
	args := []string{"-id", b.bundle.ID}
	switch log.GetLevel() {
	case log.DebugLevel, log.TraceLevel:
		args = append(args, "-debug")
	}
	args = append(args, "start")

	cmd, err := shimx2.Command(
		ctx,
		&shimx2.CommandConfig{
			Runtime:      b.runtime,
			Address:      b.containerdAddress,
			TTRPCAddress: b.containerdTTRPCAddress,
			Path:         b.bundle.Path,
			Opts:         opts,
			Args:         args, // -id nginx_1 start
			SchedCore:    b.schedCore,
		})
	if err != nil {
		return nil, err
	}
	// Windows needs a namespace when openShimLog
	ns, _ := namespaces.Namespace(ctx)
	shimCtx, cancelShimLog := context.WithCancel(namespaces.WithNamespace(context.Background(), ns))
	defer func() {
		if err != nil {
			cancelShimLog()
		}
	}()
	f, err := openShimLog(shimCtx, b.bundle, shimx2.AnonDialer)
	if err != nil {
		return nil, fmt.Errorf("open shim log pipe: %w", err)
	}
	defer func() {
		if err != nil {
			f.Close()
		}
	}()
	// open the log pipe and block until the writer is ready
	// this helps with synchronization of the shim
	// copy the shim's logs to containerd's output
	go func() {
		defer f.Close()
		_, err := io.Copy(os.Stderr, f) // log-> Stderr
		// To prevent flood of error messages, the expected error
		// should be reset, like os.ErrClosed or os.ErrNotExist, which
		// depends on platform.
		err = checkCopyShimLogError(ctx, err)
		if err != nil {
			log.G(ctx).WithError(err).Error("copy shim log")
		}
	}()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", out, err)
	}
	response := bytes.TrimSpace(out)

	onCloseWithShimLog := func() {
		onClose()
		cancelShimLog()
		f.Close()
	}
	// Save runtime binary path for restore.
	if err := os.WriteFile(filepath.Join(b.bundle.Path, "shim-binary-path"), []byte(b.runtime), 0600); err != nil {
		return nil, err
	}

	params, err := parseStartResponse(ctx, response)
	if err != nil {
		return nil, err
	}
	// unix:///run/containerd/s/1c2bf84c6529ba17d8234a68f92557fb5c1e4214eb17b724580e60b640e4f68a
	conn, err := makeConnection(ctx, params, onCloseWithShimLog)
	if err != nil {
		return nil, err
	}

	return &Shim{
		bundle: b.bundle,
		client: conn,
	}, nil
}
