package cgroup2

import (
	"errors"
)

var (
	ErrInvalidFormat    = errors.New("cgroups: parsing file with invalid format failed")
	ErrInvalidGroupPath = errors.New("cgroups: invalid group path")
)
