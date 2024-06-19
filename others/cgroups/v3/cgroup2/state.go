package cgroup2

import (
	"os"
	"path/filepath"
	"strings"
)

// State is a type that represents the state of the current cgroup
type State string

const (
	Unknown State = ""
	Thawed  State = "thawed"
	Frozen  State = "frozen"
	Deleted State = "deleted"

	cgroupFreeze = "cgroup.freeze"
)

func (s State) Values() []Value {
	v := Value{
		filename: cgroupFreeze,
	}
	switch s {
	case Frozen:
		v.value = "1"
	case Thawed:
		v.value = "0"
	}
	return []Value{
		v,
	}
}

func fetchState(path string) (State, error) {
	current, err := os.ReadFile(filepath.Join(path, cgroupFreeze))
	if err != nil {
		return Unknown, err
	}
	switch strings.TrimSpace(string(current)) {
	case "1":
		return Frozen, nil
	case "0":
		return Thawed, nil
	default:
		return Unknown, nil
	}
}
