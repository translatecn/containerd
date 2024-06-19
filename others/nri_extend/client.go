package nri_extend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"

	types "demo/others/nri_extend/types/v1"

	oci "github.com/opencontainers/runtime-spec/specs-go"
)

const (
	// DefaultBinaryPath for nri plugins
	DefaultBinaryPath = "/opt/nri/bin"
	// DefaultConfPath for the global nri configuration
	DefaultConfPath = "/etc/nri/conf.json"
	// Version of NRI
	Version = "0.1"
)

var appendPathOnce sync.Once

// New nri client
func New() (*Client, error) {
	conf, err := loadConfig(DefaultConfPath)
	if err != nil {
		return nil, err
	}

	appendPathOnce.Do(func() {
		err = os.Setenv("PATH", fmt.Sprintf("%s:%s", os.Getenv("PATH"), DefaultBinaryPath))
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		conf: conf,
	}, nil
}

// Client for calling nri plugins
type Client struct {
	conf *types.ConfigList
}

// Sandbox information
type Sandbox struct {
	// ID of the sandbox
	ID string
	// Labels of the sandbox
	Labels map[string]string
}

// process is a subset of containerd's Process interface.
type process interface {
	// ID of the process
	ID() string
	// Pid is the system specific process id
	Pid() uint32
}

// Task is a subset of containerd's Task interface.
type Task interface {
	process

	// Spec returns the current OCI specification for the task
	Spec(context.Context) (*oci.Spec, error)
}

// Invoke the ConfList of nri plugins
func (c *Client) Invoke(ctx context.Context, task Task, state types.State) ([]*types.Result, error) {
	return c.InvokeWithSandbox(ctx, task, state, nil)
}

// InvokeWithSandbox invokes the ConfList of nri plugins
func (c *Client) InvokeWithSandbox(ctx context.Context, task Task, state types.State, sandbox *Sandbox) ([]*types.Result, error) {
	if len(c.conf.Plugins) == 0 {
		return nil, nil
	}
	spec, err := task.Spec(ctx)
	if err != nil {
		return nil, err
	}
	rs, err := createSpec(spec)
	if err != nil {
		return nil, err
	}
	r := &types.Request{
		Version: c.conf.Version,
		ID:      task.ID(),
		Pid:     int(task.Pid()),
		State:   state,
		Spec:    rs,
	}
	if sandbox != nil {
		r.SandboxID = sandbox.ID
		r.Labels = sandbox.Labels
	}
	for _, p := range c.conf.Plugins {
		r.Conf = p.Conf
		result, err := c.invokePlugin(ctx, p.Type, r)
		if err != nil {
			return nil, fmt.Errorf("plugin: %s: %w", p.Type, err)
		}
		r.Results = append(r.Results, result)
	}
	return r.Results, nil
}

func (c *Client) invokePlugin(ctx context.Context, name string, r *types.Request) (*types.Result, error) {
	payload, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, name, "invoke")
	cmd.Stdin = bytes.NewBuffer(payload)
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	msg := "output:"
	if len(out) > 0 {
		msg = fmt.Sprintf("%s %q", msg, out)
	} else {
		msg = fmt.Sprintf("%s <empty>", msg)
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// ExitError is returned when there is a non-zero exit status
			msg = fmt.Sprintf("%s exit code: %d", msg, exitErr.ExitCode())
		} else {
			// plugin did not get a chance to run, return exec err
			return nil, err
		}
	}
	var result types.Result
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plugin output %s: %w", msg, err)
	}
	if result.Err() != nil {
		return nil, result.Err()
	}
	return &result, nil
}

func loadConfig(path string) (*types.ConfigList, error) {
	f, err := os.Open(path)
	if err != nil {
		// if we don't have a config list on disk, create a new one for use
		if os.IsNotExist(err) {
			return &types.ConfigList{
				Version: Version,
			}, nil
		}
		return nil, err
	}
	var c types.ConfigList
	err = json.NewDecoder(f).Decode(&c)
	f.Close()
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func createSpec(spec *oci.Spec) (*types.Spec, error) {
	s := types.Spec{
		Namespaces:  make(map[string]string),
		Annotations: spec.Annotations,
	}
	switch {
	case spec.Linux != nil:
		s.CgroupsPath = spec.Linux.CgroupsPath
		data, err := json.Marshal(spec.Linux.Resources)
		if err != nil {
			return nil, err
		}
		s.Resources = json.RawMessage(data)
		for _, ns := range spec.Linux.Namespaces {
			s.Namespaces[string(ns.Type)] = ns.Path
		}
	case spec.Windows != nil:
		data, err := json.Marshal(spec.Windows.Resources)
		if err != nil {
			return nil, err
		}
		s.Resources = json.RawMessage(data)
	}
	return &s, nil
}
