package main

import (
	"os"
	"syscall"
)

func main() {
	syscall.Mkfifo("a._in", 0700&uint32(os.ModePerm))
	syscall.Mkfifo("a.out", 0700&uint32(os.ModePerm))
}
