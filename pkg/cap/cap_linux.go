// Package cap provides Linux capability utility
package cap

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// FromNumber returns a cap string like "CAP_SYS_ADMIN"
// that corresponds to the given number like 21.
//
// FromNumber returns an empty string for unknown cap number.
func FromNumber(num int) string {
	if num < 0 || num > len(capsLatest)-1 {
		return ""
	}
	return capsLatest[num]
}

// FromBitmap parses an uint64 bitmap into string slice like
// []{"CAP_SYS_ADMIN", ...}.
//
// Unknown cap numbers are returned as []int.
func FromBitmap(v uint64) ([]string, []int) {
	var (
		res     []string
		unknown []int
	)
	for i := 0; i <= 63; i++ {
		if b := (v >> i) & 0x1; b == 0x1 {
			if s := FromNumber(i); s != "" {
				res = append(res, s)
			} else {
				unknown = append(unknown, i)
			}
		}
	}
	return res, unknown
}

// Type is the type of capability
type Type int

const (
	// Effective is CapEff
	Effective Type = 1 << iota
	// Permitted is CapPrm
	Permitted
	// Inheritable is CapInh
	Inheritable
	// Bounding is CapBnd
	Bounding
	// Ambient is CapAmb
	Ambient
)

//Permitted：进程所能使用的capabilities的上限集合，在该集合中有的权限，并不代表线程可以使用。必须要保证在Effective集合中有该权限。
//Effective：有效的capabilities，这里的权限是Linux内核检查线程是否具有特权操作时检查的集合。
//Inheritable：即继承。通过exec系统调用启动新进程时可以继承给新进程权限集合。注意，该权限集合继承给新进程后，也就是新进程的Permitted集合。
//Bounding: Bounding限制了进程可以获得的集合，只有在Bounding集合中存在的权限，才能出现在Permitted和Inheritable集合中。
//Ambient: Ambient集合中的权限会被应用到所有非特权进程上（特权进程，指当用户执行某一程序时，临时获得该程序所有者的身份）。
//	然而，并不是所有在Ambient集合中的权限都会被保留，只有在Permitted和Effective集合中的权限，才会在被exec调用时保留。

//在创建新的User namespace时不需要任何权限；而在创建其他类型的namespace（如UTS、PID、Mount、IPC、Network、Cgroupnamespace）时，
//需要进程在对应User namespace中有CAP_SYS_ADMIN权限。

// ParseProcPIDStatus returns uint64 bitmap value from /proc/<PID>/status file
func ParseProcPIDStatus(r io.Reader) (map[Type]uint64, error) {
	res := make(map[Type]uint64)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		switch k {
		case "CapInh", "CapPrm", "CapEff", "CapBnd", "CapAmb":
			ui64, err := strconv.ParseUint(strings.TrimSpace(v), 16, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse line %q", line)
			}
			switch k {
			case "CapInh":
				res[Inheritable] = ui64
			case "CapPrm":
				res[Permitted] = ui64
			case "CapEff":
				res[Effective] = ui64
			case "CapBnd":
				res[Bounding] = ui64
			case "CapAmb":
				res[Ambient] = ui64
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return res, nil
}

// Current returns the list of the effective and the known caps of
// the current process.
//
// The result is like []string{"CAP_SYS_ADMIN", ...}.
func Current() ([]string, error) {
	f, err := os.Open("/proc/self/status")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	caps, err := ParseProcPIDStatus(f)
	if err != nil {
		return nil, err
	}
	capEff := caps[Effective]
	names, _ := FromBitmap(capEff)
	return names, nil
}

var (
	// caps35 is the caps of kernel 3.5 (37 entries)
	caps35 = []string{
		"CAP_CHOWN",            // 2.2
		"CAP_DAC_OVERRIDE",     // 2.2
		"CAP_DAC_READ_SEARCH",  // 2.2
		"CAP_FOWNER",           // 2.2
		"CAP_FSETID",           // 2.2
		"CAP_KILL",             // 2.2
		"CAP_SETGID",           // 2.2
		"CAP_SETUID",           // 2.2
		"CAP_SETPCAP",          // 2.2
		"CAP_LINUX_IMMUTABLE",  // 2.2
		"CAP_NET_BIND_SERVICE", // 2.2
		"CAP_NET_BROADCAST",    // 2.2
		"CAP_NET_ADMIN",        // 2.2
		"CAP_NET_RAW",          // 2.2
		"CAP_IPC_LOCK",         // 2.2
		"CAP_IPC_OWNER",        // 2.2
		"CAP_SYS_MODULE",       // 2.2
		"CAP_SYS_RAWIO",        // 2.2
		"CAP_SYS_CHROOT",       // 2.2
		"CAP_SYS_PTRACE",       // 2.2
		"CAP_SYS_PACCT",        // 2.2
		"CAP_SYS_ADMIN",        // 2.2
		"CAP_SYS_BOOT",         // 2.2
		"CAP_SYS_NICE",         // 2.2
		"CAP_SYS_RESOURCE",     // 2.2
		"CAP_SYS_TIME",         // 2.2
		"CAP_SYS_TTY_CONFIG",   // 2.2
		"CAP_MKNOD",            // 2.4
		"CAP_LEASE",            // 2.4
		"CAP_AUDIT_WRITE",      // 2.6.11
		"CAP_AUDIT_CONTROL",    // 2.6.11
		"CAP_SETFCAP",          // 2.6.24
		"CAP_MAC_OVERRIDE",     // 2.6.25
		"CAP_MAC_ADMIN",        // 2.6.25
		"CAP_SYSLOG",           // 2.6.37
		"CAP_WAKE_ALARM",       // 3.0
		"CAP_BLOCK_SUSPEND",    // 3.5
	}
	// caps316 is the caps of kernel 3.16 (38 entries)
	caps316 = append(caps35, "CAP_AUDIT_READ")
	// caps58 is the caps of kernel 5.8 (40 entries)
	caps58 = append(caps316, []string{"CAP_PERFMON", "CAP_BPF"}...)
	// caps59 is the caps of kernel 5.9 (41 entries)
	caps59     = append(caps58, "CAP_CHECKPOINT_RESTORE")
	capsLatest = caps59
)

// Known returns the known cap strings of the latest kernel.
// The current latest kernel is 5.9.
func Known() []string {
	return capsLatest
}
