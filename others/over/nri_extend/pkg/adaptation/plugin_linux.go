package adaptation

import (
	"errors"
	"fmt"
	stdnet "net"

	"golang.org/x/sys/unix"
)

// getPeerPid returns the process id at the other end of the connection.
func getPeerPid(conn stdnet.Conn) (int, error) {
	var cred *unix.Ucred

	uc, ok := conn.(*stdnet.UnixConn)
	if !ok {
		return 0, errors.New("invalid connection, not *net.UnixConn")
	}

	raw, err := uc.SyscallConn()
	if err != nil {
		return 0, fmt.Errorf("failed get raw unix domain connection: %w", err)
	}

	ctrlErr := raw.Control(func(fd uintptr) {
		cred, err = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get process credentials: %w", err)
	}
	if ctrlErr != nil {
		return 0, fmt.Errorf("uc.SyscallConn().Control() failed: %w", ctrlErr)
	}

	return int(cred.Pid), nil
}
