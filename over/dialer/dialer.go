package dialer

import (
	"context"
	"fmt"
	"net"
	"time"
)

type dialResult struct {
	c   net.Conn
	err error
}

// ContextDialer returns a GRPC net.Conn connected to the provided address
func ContextDialer(ctx context.Context, address string) (net.Conn, error) {
	if deadline, ok := ctx.Deadline(); ok {
		return timeoutDialer(address, time.Until(deadline))
	}
	return timeoutDialer(address, 0)
}

// Dialer returns a GRPC net.Conn connected to the provided address
// Deprecated: use ContextDialer and grpc.WithContextDialer.
var _ = timeoutDialer

func timeoutDialer(address string, timeout time.Duration) (net.Conn, error) {
	var (
		stopC = make(chan struct{})
		synC  = make(chan *dialResult)
	)
	go func() {
		defer close(synC)
		for {
			select {
			case <-stopC:
				return
			default:
				c, err := dialer(address, timeout)
				if isNoent(err) {
					<-time.After(10 * time.Millisecond)
					continue
				}
				synC <- &dialResult{c, err}
				return
			}
		}
	}()
	select {
	case dr := <-synC:
		return dr.c, dr.err
	case <-time.After(timeout):
		close(stopC)
		go func() {
			dr := <-synC
			if dr != nil && dr.c != nil {
				dr.c.Close()
			}
		}()
		return nil, fmt.Errorf("dial %s: timeout", address)
	}
}
