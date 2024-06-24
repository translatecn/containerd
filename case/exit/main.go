package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

// 父进程退出，子进程不退出  TODO
func main() {

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan,
		syscall.SIGINT,
		syscall.SIGKILL,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		fmt.Println(<-sigchan)
	}()

	fmt.Println(os.Getpid(), os.Args)
	executable, _ := os.Executable()
	if len(os.Args) > 1 {
		fmt.Println(os.Getpid(), os.Args, os.Getppid())
		time.Sleep(time.Second * 10)
		fmt.Println(os.Getpid(), os.Args, os.Getppid())
	} else {
		cmd := exec.Command(executable, "start")
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
			//Pdeathsig: syscall.SIGHUP, // 当父进程退出时，子进程会收到SIGHUP信号
		}
		if err := cmd.Start(); err != nil {
			log.Println("exec the cmd ", " failed")
		}
		fmt.Println("exec the cmd ", cmd.Process.Pid)
	}

	time.Sleep(time.Second)
}
