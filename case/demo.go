package main

import (
	"fmt"
	"os/exec"
	"syscall"
)

func main() {

	// 创建多个子进程
	for i := 0; i < 10; i++ {
		go func() {
			cmd := exec.Command("sleep", "10")
			cmd.SysProcAttr = &syscall.SysProcAttr{
				Setpgid: true, // 创建新进程组，以便Ctrl+C信号不会被传递到子进程
			}
			cmd.Start()
			cmd.Wait()
		}()
	}
	// 等待任意一个子进程退出并回收资源
	var ws syscall.WaitStatus
	for {
		pid, _ := syscall.Wait4(-1, &ws, syscall.WNOHANG, nil)
		if pid > 0 {
			if ws.Exited() {
				exitStatus := ws.ExitStatus()
				fmt.Printf("子进程 %d 退出，退出状态码：%d\n", pid, exitStatus)
			} else if ws.Signaled() {
				signal := ws.Signal()
				fmt.Printf("子进程 %d 收到信号：%d\n", pid, signal)
			} else {
				fmt.Printf("子进程 %d 退出，但状态未知\n", pid)
			}
		} else {
			//fmt.Println("没有子进程退出")
		}
	}

}
