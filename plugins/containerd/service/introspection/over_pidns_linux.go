package introspection

import (
	"fmt"
	"os"
	"syscall"
)

func statPIDNS(pid int) (uint64, error) {
	f := fmt.Sprintf("/proc/%d/ns/pid", pid)
	st, err := os.Stat(f)
	if err != nil {
		return 0, err
	}
	stSys, ok := st.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("%T is not *syscall.Stat_t", st.Sys())
	}
	return stSys.Ino, nil
}
