package containerd

import (
	"context"
	"demo/config/runc"
	"demo/over/api/services/tasks/v1"
	"demo/over/api/types"
	tasktypes "demo/over/api/types/task"
	"demo/over/cio"
	"demo/over/containers"
	"demo/over/errdefs"
	"demo/over/fifo"
	"demo/over/images"
	"demo/over/oci"
	"demo/over/protobuf"
	"demo/over/typeurl/v2"
	"encoding/json"
	"fmt"
	ver "github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/opencontainers/selinux/go-selinux/label"
	"os"
	"path/filepath"
	"strings"
)

const (
	checkpointImageNameLabel       = "org.opencontainers.image.ref.name"
	checkpointRuntimeNameLabel     = "io.containerd.checkpoint.runtime"
	checkpointSnapshotterNameLabel = "io.containerd.checkpoint.snapshotter"
)

// Container is a metadata object for container resources and task creation
type Container interface {
	// ID identifies the container
	ID() string
	// Info returns the underlying container record type
	Info(context.Context, ...InfoOpts) (containers.Container, error)
	// Delete removes the container
	Delete(context.Context, ...DeleteOpts) error
	NewTask(context.Context, cio.Creator, ...NewTaskOpts) (Task, error)
	// Spec returns the OCI runtime specification
	Spec(context.Context) (*oci.Spec, error)
	// Task 返回容器的当前任务
	// 如果。通过附加选项，客户端将重新连接到正在运行的任务的IO。如果容器不存在任务，则返回NotFound错误
	// 客户端必须确保只有一个读取器连接到任务，并从任务的fifo消费输出
	Task(context.Context, cio.Attach) (Task, error)
	// Image returns the image that the container is based on
	Image(context.Context) (Image, error)
	// Labels returns the labels set on the container
	Labels(context.Context) (map[string]string, error)
	// SetLabels sets the provided labels for the container and returns the final label set
	SetLabels(context.Context, map[string]string) (map[string]string, error)
	Extensions(context.Context) (map[string]typeurl.Any, error)
	// Update a container
	Update(context.Context, ...UpdateContainerOpts) error
	// Checkpoint creates a checkpoint image of the current container
	Checkpoint(context.Context, string, ...CheckpointOpts) (Image, error)
}

var _ = (Container)(&container{})

type container struct {
	client   *Client
	id       string
	metadata containers.Container
}

// ID returns the container's unique randomId
func (c *container) ID() string {
	return c.id
}

func (c *container) Info(ctx context.Context, opts ...InfoOpts) (containers.Container, error) {
	i := &InfoConfig{
		// default to refreshing the container's local metadata
		Refresh: true,
	}
	for _, o := range opts {
		o(i)
	}
	if i.Refresh {
		metadata, err := c.get(ctx)
		if err != nil {
			return c.metadata, err
		}
		c.metadata = metadata
	}
	return c.metadata, nil
}

func (c *container) Extensions(ctx context.Context) (map[string]typeurl.Any, error) {
	r, err := c.get(ctx)
	if err != nil {
		return nil, err
	}
	return r.Extensions, nil
}

func (c *container) Labels(ctx context.Context) (map[string]string, error) {
	r, err := c.get(ctx)
	if err != nil {
		return nil, err
	}
	return r.Labels, nil
}

func (c *container) SetLabels(ctx context.Context, labels map[string]string) (map[string]string, error) {
	container := containers.Container{
		ID:     c.id,
		Labels: labels,
	}

	var paths []string
	// mask off paths so we only muck with the labels encountered in labels.
	// Labels not in the passed in argument will be left alone.
	for k := range labels {
		paths = append(paths, strings.Join([]string{"labels", k}, "."))
	}

	r, err := c.client.ContainerService().Update(ctx, container, paths...)
	if err != nil {
		return nil, err
	}
	return r.Labels, nil
}

// Spec returns the current OCI specification for the container
func (c *container) Spec(ctx context.Context) (*oci.Spec, error) {
	r, err := c.get(ctx)
	if err != nil {
		return nil, err
	}
	var s oci.Spec
	if err := json.Unmarshal(r.Spec.GetValue(), &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Delete deletes an existing container
// an error is returned if the container has running tasks
func (c *container) Delete(ctx context.Context, opts ...DeleteOpts) error {
	if _, err := c.loadTask(ctx, nil); err == nil {
		return fmt.Errorf("cannot delete running task %v: %w", c.id, errdefs.ErrFailedPrecondition)
	}
	r, err := c.get(ctx)
	if err != nil {
		return err
	}
	for _, o := range opts {
		if err := o(ctx, c.client, r); err != nil {
			return err
		}
	}
	return c.client.ContainerService().Delete(ctx, c.id)
}

func (c *container) Task(ctx context.Context, attach cio.Attach) (Task, error) {
	return c.loadTask(ctx, attach)
}

// Image returns the image that the container is based on
func (c *container) Image(ctx context.Context) (Image, error) {
	r, err := c.get(ctx)
	if err != nil {
		return nil, err
	}
	if r.Image == "" {
		return nil, fmt.Errorf("container not created from an image: %w", errdefs.ErrNotFound)
	}
	i, err := c.client.ImageService().Get(ctx, r.Image)
	if err != nil {
		return nil, fmt.Errorf("failed to get image %s for container: %w", r.Image, err)
	}
	return NewImage(c.client, i), nil
}

func (c *container) Checkpoint(ctx context.Context, ref string, opts ...CheckpointOpts) (Image, error) {
	index := &ocispec.Index{
		Versioned: ver.Versioned{
			SchemaVersion: 2,
		},
		Annotations: make(map[string]string),
	}
	copts := &runc.CheckpointOptions{
		Exit:                false,
		OpenTcp:             false,
		ExternalUnixSockets: false,
		Terminal:            false,
		FileLocks:           true,
		EmptyNamespaces:     nil,
	}
	info, err := c.Info(ctx)
	if err != nil {
		return nil, err
	}

	img, err := c.Image(ctx)
	if err != nil {
		return nil, err
	}

	ctx, done, err := c.client.WithLease(ctx)
	if err != nil {
		return nil, err
	}
	defer done(ctx)

	// add image name to manifest
	index.Annotations[checkpointImageNameLabel] = img.Name()
	// add runtime info to index
	index.Annotations[checkpointRuntimeNameLabel] = info.Runtime.Name
	// add snapshotter info to index
	index.Annotations[checkpointSnapshotterNameLabel] = info.Snapshotter

	// process remaining opts
	for _, o := range opts {
		if err := o(ctx, c.client, &info, index, copts); err != nil {
			err = errdefs.FromGRPC(err)
			if !errdefs.IsAlreadyExists(err) {
				return nil, err
			}
		}
	}

	desc, err := writeIndex(ctx, index, c.client, c.ID()+"index")
	if err != nil {
		return nil, err
	}
	i := images.Image{
		Name:   ref,
		Target: desc,
	}
	checkpoint, err := c.client.ImageService().Create(ctx, i)
	if err != nil {
		return nil, err
	}

	return NewImage(c.client, checkpoint), nil
}

func (c *container) loadTask(ctx context.Context, ioAttach cio.Attach) (Task, error) {
	response, err := c.client.TaskService().Get(ctx, &tasks.GetRequest{
		ContainerID: c.id,
	})
	if err != nil {
		err = errdefs.FromGRPC(err)
		if errdefs.IsNotFound(err) {
			return nil, fmt.Errorf("no running task found: %w", err)
		}
		return nil, err
	}
	var i cio.IO
	if ioAttach != nil && response.Process.Status != tasktypes.Status_UNKNOWN {
		// 不要为未知状态的任务附加IO，因为无论如何也没有fifo路径。
		if i, err = attachExistingIO(response, ioAttach); err != nil {
			return nil, err
		}
	}
	t := &task{
		client: c.client,
		io:     i,
		id:     response.Process.ID,
		pid:    response.Process.Pid,
		c:      c,
	}
	return t, nil
}

func (c *container) get(ctx context.Context) (containers.Container, error) {
	return c.client.ContainerService().Get(ctx, c.id)
}

// get the existing fifo paths from the task information stored by the daemon
func attachExistingIO(response *tasks.GetResponse, ioAttach cio.Attach) (cio.IO, error) {
	fifoSet := loadFifos(response)
	return ioAttach(fifoSet)
}

// loadFifos loads the containers fifos
func loadFifos(response *tasks.GetResponse) *cio.FIFOSet {
	fifos := []string{
		response.Process.Stdin,
		response.Process.Stdout,
		response.Process.Stderr,
	}
	closer := func() error {
		var (
			err  error
			dirs = map[string]struct{}{}
		)
		for _, f := range fifos {
			if isFifo, _ := fifo.IsFifo(f); isFifo {
				if rerr := os.Remove(f); err == nil {
					err = rerr
				}
				dirs[filepath.Dir(f)] = struct{}{}
			}
		}
		for dir := range dirs {
			// we ignore errors here because we don't
			// want to remove the directory if it isn't
			// empty
			os.Remove(dir)
		}
		return err
	}

	return cio.NewFIFOSet(cio.Config{
		Stdin:    response.Process.Stdin,
		Stdout:   response.Process.Stdout,
		Stderr:   response.Process.Stderr,
		Terminal: response.Process.Terminal,
	}, closer)
}

func (c *container) Update(ctx context.Context, opts ...UpdateContainerOpts) error {
	// fetch the current container config before updating it
	r, err := c.get(ctx)
	if err != nil {
		return err
	}
	for _, o := range opts {
		if err := o(ctx, c.client, &r); err != nil {
			return err
		}
	}
	if _, err := c.client.ContainerService().Update(ctx, r); err != nil {
		return errdefs.FromGRPC(err)
	}
	return nil
}
func (c *container) NewTask(ctx context.Context, ioCreate cio.Creator, opts ...NewTaskOpts) (_ Task, err error) {
	i, err := ioCreate(c.id)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil && i != nil {
			i.Cancel()
			i.Close()
		}
	}()
	cfg := i.Config()
	request := &tasks.CreateTaskRequest{
		ContainerID: c.id,
		Terminal:    cfg.Terminal,
		Stdin:       cfg.Stdin,  // 进程间通信的特殊文件类型
		Stdout:      cfg.Stdout, // 进程间通信的特殊文件类型
		Stderr:      cfg.Stderr, // 进程间通信的特殊文件类型
	}
	r, err := c.get(ctx)
	if err != nil {
		return nil, err
	}
	if r.SnapshotKey != "" {
		if r.Snapshotter == "" {
			return nil, fmt.Errorf("unable to resolve rootfs mounts without snapshotter on container: %w", errdefs.ErrInvalidArgument)
		}

		// get the rootfs from the snapshotter and add it to the request
		s, err := c.client.getSnapshotter(ctx, r.Snapshotter)
		if err != nil {
			return nil, err
		}
		mounts, err := s.Mounts(ctx, r.SnapshotKey)
		if err != nil {
			return nil, err
		}
		spec, err := c.Spec(ctx)
		if err != nil {
			return nil, err
		}
		for _, m := range mounts {
			if spec.Linux != nil && spec.Linux.MountLabel != "" {
				context := label.FormatMountLabel("", spec.Linux.MountLabel)
				if context != "" {
					m.Options = append(m.Options, context)
				}
			}
			request.Rootfs = append(request.Rootfs, &types.Mount{
				Type:    m.Type,
				Source:  m.Source,
				Target:  m.Target,
				Options: m.Options,
			})
		}
	}
	info := TaskInfo{
		runtime: r.Runtime.Name,
	}
	for _, o := range opts {
		if err := o(ctx, c.client, &info); err != nil {
			return nil, err
		}
	}
	if info.RootFS != nil {
		for _, m := range info.RootFS {
			request.Rootfs = append(request.Rootfs, &types.Mount{
				Type:    m.Type,
				Source:  m.Source,
				Target:  m.Target,
				Options: m.Options,
			})
		}
	}
	request.RuntimePath = info.RuntimePath
	if info.Options != nil {
		any, err := typeurl.MarshalAny(info.Options)
		if err != nil {
			return nil, err
		}
		request.Options = protobuf.FromAny(any)
	}
	t := &task{
		client: c.client,
		io:     i,
		id:     c.id,
		c:      c,
	}
	if info.Checkpoint != nil {
		request.Checkpoint = info.Checkpoint
	}
	response, err := c.client.TaskService().Create(ctx, request)
	if err != nil {
		return nil, errdefs.FromGRPC(err)
	}
	t.pid = response.Pid
	return t, nil
}
func containerFromRecord(client *Client, c containers.Container) *container {
	return &container{
		client:   client,
		id:       c.ID,
		metadata: c,
	}
}
