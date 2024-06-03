package my_mount

import (
	"github.com/fatih/color"
	"golang.org/x/sys/unix"
	"os"
	"strings"
)

func Mount(source string, target string, fstype string, flags uintptr, data string) (err error) {
	if fstype == "overlay" {
		color.New(color.FgGreen).SetWriter(os.Stderr).Println("mount ", flag2str(fstype, int64(flags)), data, target)

	} else {
		color.New(color.FgGreen).SetWriter(os.Stderr).Println("mount ", flag2str(fstype, int64(flags)), source, target, data)

	}
	return unix.Mount(source, target, fstype, flags, data)
}

func Unmount(target string, flags int) (err error) {
	color.New(color.FgGreen).SetWriter(os.Stderr).Println("umount ", flag2str("", int64(flags)), target)
	return unix.Unmount(target, flags)
}

func flag2str(fstype string, flag int64) string {

	flags := map[string]struct {
		clear    bool
		flag     int64
		shotName string
	}{
		"atime":         {true, unix.MS_NOATIME, "-o"},
		"noatime":       {false, unix.MS_NOATIME, "-o"},
		"diratime":      {true, unix.MS_NODIRATIME, "-o"},
		"nodiratime":    {false, unix.MS_NODIRATIME, "-o"},
		"relatime":      {false, unix.MS_RELATIME, "-o"},
		"norelatime":    {true, unix.MS_RELATIME, "-o"},
		"nostrictatime": {true, unix.MS_STRICTATIME, "-o"},
		"strictatime":   {false, unix.MS_STRICTATIME, "-o"},
		"mand":          {false, unix.MS_MANDLOCK, "-o"},
		"nomand":        {true, unix.MS_MANDLOCK, "-o"},
		"bind":          {false, unix.MS_BIND, "--bind"},
		"dirsync":       {false, unix.MS_DIRSYNC, "-o"},
		"nodev":         {false, unix.MS_NODEV, "-o"},
		"dev":           {true, unix.MS_NODEV, "-o"},
		"noexec":        {false, unix.MS_NOEXEC, "-o"},
		"exec":          {true, unix.MS_NOEXEC, "-o"},
		"rbind":         {false, unix.MS_BIND | unix.MS_REC, "--rbind"},
		"remount":       {false, unix.MS_REMOUNT, "-o"},
		"ro":            {false, unix.MS_RDONLY, "-o"},
		"rw":            {true, unix.MS_RDONLY, "-o"},
		"suid":          {true, unix.MS_NOSUID, "-o"},
		"nosuid":        {false, unix.MS_NOSUID, "-o"},
		"sync":          {false, unix.MS_SYNCHRONOUS, "-o"},
		"async":         {true, unix.MS_SYNCHRONOUS, "-o"},
	}
	res := map[string][]string{}
	for k, v := range flags {
		if flag&v.flag == 0 {
			continue
		}
		if _, ok := res[v.shotName]; !ok {
			res[v.shotName] = []string{}
		}
		res[v.shotName] = append(res[v.shotName], k)
	}
	xxxx := ``
	for k, v := range res {
		if k == "--bind" {
			if _, ok := res["--rbind"]; ok {
				continue
			}
		}
		if k == "--rbind" {
			xxxx += "--rbind "
		} else {
			xxxx += k + strings.Join(v, ",") + " "
		}
	}
	if fstype == "overlay" {
		xxxx = "overlay -o" + xxxx
	}
	if fstype == "bind" && strings.Contains(xxxx, "bind") {

	} else if fstype != "" {
		xxxx = "-t " + fstype + " " + xxxx
	}
	return xxxx

}
