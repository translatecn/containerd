package mount

import "github.com/moby/sys/mountinfo"

// Info reveals information about a particular mounted filesystem. This
// struct is populated from the content in the /proc/<pid>/mountinfo file.
type Info = mountinfo.Info
