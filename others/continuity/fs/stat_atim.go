package fs

import (
	"syscall"
)

// StatAtime returns the Atim
func StatAtime(st *syscall.Stat_t) syscall.Timespec {
	return st.Atim
}

// StatCtime returns the Ctim

// StatMtime returns the Mtim
func StatMtime(st *syscall.Stat_t) syscall.Timespec {
	return st.Mtim
}

// StatATimeAsTime returns st.Atim as a time.Time
