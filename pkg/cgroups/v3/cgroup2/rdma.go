package cgroup2

import (
	"fmt"
)

type RDMA struct {
	Limit []RDMAEntry
}

type RDMAEntry struct {
	Device     string
	HcaHandles uint32
	HcaObjects uint32
}

func (r RDMAEntry) String() string {
	return fmt.Sprintf("%s hca_handle=%d hca_object=%d", r.Device, r.HcaHandles, r.HcaObjects)
}

func (r *RDMA) Values() (o []Value) {
	for _, e := range r.Limit {
		o = append(o, Value{
			filename: "rdma.max",
			value:    e.String(),
		})
	}

	return o
}
