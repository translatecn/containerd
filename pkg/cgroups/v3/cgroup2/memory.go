package cgroup2

type Memory struct {
	Swap *int64
	Min  *int64
	Max  *int64
	Low  *int64
	High *int64
}

func (r *Memory) Values() (o []Value) {
	if r.Swap != nil {
		o = append(o, Value{
			filename: "memory.swap.max",
			value:    *r.Swap,
		})
	}
	if r.Min != nil {
		o = append(o, Value{
			filename: "memory.min",
			value:    *r.Min,
		})
	}
	if r.Max != nil {
		o = append(o, Value{
			filename: "memory.max",
			value:    *r.Max,
		})
	}
	if r.Low != nil {
		o = append(o, Value{
			filename: "memory.low",
			value:    *r.Low,
		})
	}
	if r.High != nil {
		o = append(o, Value{
			filename: "memory.high",
			value:    *r.High,
		})
	}
	return o
}
