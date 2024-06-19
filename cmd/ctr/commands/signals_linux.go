package commands

import (
	"os"

	"golang.org/x/sys/unix"
)

func canIgnoreSignal(s os.Signal) bool {
	return s == unix.SIGURG
}
