package cgroup1

import (
	"os"
	"path/filepath"
	"strconv"

	v1 "demo/pkg/cgroups/v3/cgroup1/stats"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func NewPids(root string) *pidsController {
	return &pidsController{
		root: filepath.Join(root, string(Pids)),
	}
}

type pidsController struct {
	root string
}

func (p *pidsController) Name() Name {
	return Pids
}

func (p *pidsController) Path(path string) string {
	return filepath.Join(p.root, path)
}

func (p *pidsController) Create(path string, resources *specs.LinuxResources) error {
	if err := os.MkdirAll(p.Path(path), defaultDirPerm); err != nil {
		return err
	}
	if resources.Pids != nil && resources.Pids.Limit > 0 {
		return os.WriteFile(
			filepath.Join(p.Path(path), "pids.max"),
			[]byte(strconv.FormatInt(resources.Pids.Limit, 10)),
			defaultFilePerm,
		)
	}
	return nil
}

func (p *pidsController) Update(path string, resources *specs.LinuxResources) error {
	return p.Create(path, resources)
}

func (p *pidsController) Stat(path string, stats *v1.Metrics) error {
	current, err := readUint(filepath.Join(p.Path(path), "pids.current"))
	if err != nil {
		return err
	}
	max, err := readUint(filepath.Join(p.Path(path), "pids.max"))
	if err != nil {
		return err
	}
	stats.Pids = &v1.PidsStat{
		Current: current,
		Limit:   max,
	}
	return nil
}
