package schedcore

import (
	"golang.org/x/sys/unix"
)

// PidType is the type of provided pid value and how it should be treated
type PidType int

const (
	// Pid affects the current pid
	Pid PidType = pidtypePid
	// ThreadGroup affects all threads in the group
	ThreadGroup PidType = pidtypeTgid
	// ProcessGroup affects all processes in the group
	ProcessGroup PidType = pidtypePgid
)

const (
	pidtypePid  = 0
	pidtypeTgid = 1
	pidtypePgid = 2
)

// Create a new sched core domain
func Create(t PidType) error {
	return unix.Prctl(unix.PR_SCHED_CORE, unix.PR_SCHED_CORE_CREATE, 0, uintptr(t), 0)
}

