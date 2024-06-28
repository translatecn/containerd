package fifo

import "errors"

var (
	ErrClosed      = errors.New("fifo closed")
	ErrCtrlClosed  = errors.New("control of closed fifo")
	ErrRdFrmWRONLY = errors.New("reading from write-only fifo")
	ErrReadClosed  = errors.New("reading from a closed fifo")
	ErrWrToRDONLY  = errors.New("writing to read-only fifo")
	ErrWriteClosed = errors.New("writing to a closed fifo")
)
