package atomic

import "sync/atomic"

// Bool is an atomic Boolean,
// Its methods are all atomic, thus safe to be called by
// multiple goroutines simultaneously.
type Bool interface {
	Set()
	Unset()
	IsSet() bool
}

// NewBool creates an Bool with given default value
func NewBool(ok bool) Bool {
	ab := new(atomicBool)
	if ok {
		ab.Set()
	}
	return ab
}

type atomicBool int32

// Set sets the Boolean to true
func (ab *atomicBool) Set() {
	atomic.StoreInt32((*int32)(ab), 1)
}

// Unset sets the Boolean to false
func (ab *atomicBool) Unset() {
	atomic.StoreInt32((*int32)(ab), 0)
}

// IsSet returns whether the Boolean is true
func (ab *atomicBool) IsSet() bool {
	return atomic.LoadInt32((*int32)(ab)) == 1
}
