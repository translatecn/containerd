package cgroup2

import "strconv"

type Pids struct {
	Max int64
}

func (r *Pids) Values() (o []Value) {
	if r.Max != 0 {
		limit := "max"
		if r.Max > 0 {
			limit = strconv.FormatInt(r.Max, 10)
		}
		o = append(o, Value{
			filename: "pids.max",
			value:    limit,
		})
	}
	return o
}
