package cgroup2

import "strings"

type HugeTlb []HugeTlbEntry

type HugeTlbEntry struct {
	HugePageSize string
	Limit        uint64
}

func (r *HugeTlb) Values() (o []Value) {
	for _, e := range *r {
		o = append(o, Value{
			filename: strings.Join([]string{"hugetlb", e.HugePageSize, "max"}, "."),
			value:    e.Limit,
		})
	}

	return o
}
