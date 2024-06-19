package cgroup2

import (
	"math"
	"strconv"
	"strings"
)

type CPUMax string

type CPU struct {
	Weight *uint64
	Max    CPUMax
	Cpus   string
	Mems   string
}

func (c CPUMax) extractQuotaAndPeriod() (int64, uint64) {
	var (
		quota  int64
		period uint64
	)
	values := strings.Split(string(c), " ")
	if values[0] == "max" {
		quota = math.MaxInt64
	} else {
		quota, _ = strconv.ParseInt(values[0], 10, 64)
	}
	period, _ = strconv.ParseUint(values[1], 10, 64)
	return quota, period
}

func (r *CPU) Values() (o []Value) {
	if r.Weight != nil {
		o = append(o, Value{
			filename: "cpu.weight",
			value:    *r.Weight,
		})
	}
	if r.Max != "" {
		o = append(o, Value{
			filename: "cpu.max",
			value:    r.Max,
		})
	}
	if r.Cpus != "" {
		o = append(o, Value{
			filename: "cpuset.cpus",
			value:    r.Cpus,
		})
	}
	if r.Mems != "" {
		o = append(o, Value{
			filename: "cpuset.mems",
			value:    r.Mems,
		})
	}
	return o
}
