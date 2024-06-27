package main

import (
	"demo/over/my_mount"
	cnins "demo/over/ns"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"runtime"
	"sync"
)

func newNs() {
	var wg sync.WaitGroup
	wg.Add(1)
	// do namespace work in a dedicated goroutine, so that we can safely
	// Lock/Unlock OSThread without upsetting the lock/unlock state of
	// the caller of this function
	go (func() {
		defer wg.Done()
		runtime.LockOSThread()
		// Don't unlock. By not unlocking, golang will kill the OS thread when the
		// goroutine is done (for go1.10+)

		var origNS cnins.NetNS
		origNS, err := cnins.GetNS(getCurrentThreadNetNSPath())
		if err != nil {
			return
		}
		defer origNS.Close()

		// create a new netns on the current thread
		err = unix.Unshare(unix.CLONE_NEWNET)
		if err != nil {
			return
		}

		// Put this thread back to the orig ns, since it might get reused (pre go1.10)
		defer origNS.Set()

		// bind mount the netns from the current thread (from /proc) onto the
		// mount point. This causes the namespace to persist, even when there
		// are no threads in the ns.
		err = my_mount.Mount(getCurrentThreadNetNSPath(), "/tmp/xxxxxxx", "none", unix.MS_BIND, "")
		if err != nil {
			err = fmt.Errorf("failed to bind mount ns at %s: %w", "/tmp/xxxxxxx", err)
		}
	})()
	wg.Wait()
}
func getCurrentThreadNetNSPath() string {
	// Lock the thread in case other goroutine executes in it and changes its
	// network namespace after getCurrentThreadNetNSPath(), otherwise it might
	// return an unexpected network namespace.
	runtime.LockOSThread()
	nspath := fmt.Sprintf("/proc/%d/task/%d/ns/net", os.Getpid(), unix.Gettid())
	runtime.UnlockOSThread()
	return nspath
}
func main() {
	newNs()
	nspath := getCurrentThreadNetNSPath()
	go func() {
		runtime.LockOSThread()
		currentNs, _ := os.Open(nspath)
		unix.Setns(int(currentNs.Fd()), unix.CLONE_NEWNET)
		runtime.UnlockOSThread()
	}()

	fmt.Println(nspath)

}
