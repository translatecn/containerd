package cgroup2

import "fmt"

type IOType string

const (
	ReadBPS   IOType = "rbps"
	WriteBPS  IOType = "wbps"
	ReadIOPS  IOType = "riops"
	WriteIOPS IOType = "wiops"
)

type BFQ struct {
	Weight uint16
}

type Entry struct {
	Type  IOType
	Major int64
	Minor int64
	Rate  uint64
}

func (e Entry) String() string {
	return fmt.Sprintf("%d:%d %s=%d", e.Major, e.Minor, e.Type, e.Rate)
}

type IO struct {
	BFQ BFQ
	Max []Entry
}

func (i *IO) Values() (o []Value) {
	if i.BFQ.Weight != 0 {
		o = append(o, Value{
			filename: "io.bfq.weight",
			value:    i.BFQ.Weight,
		})
	}
	for _, e := range i.Max {
		o = append(o, Value{
			filename: "io.max",
			value:    e.String(),
		})
	}
	return o
}
