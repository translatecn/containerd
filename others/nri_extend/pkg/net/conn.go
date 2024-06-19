package net

import (
	"io"
	"net"
	"sync"
)

// connListener wraps a pre-connected socket in a net.Listener.
type connListener struct {
	next   chan net.Conn
	conn   net.Conn
	addr   net.Addr
	lock   sync.RWMutex // for Close()
	closed bool
}

// NewConnListener wraps an existing net.Conn in a net.Listener.
//
// The first call to Accept() on the listener will return the wrapped
// connection. Subsequent calls to Accept() block until the listener
// is closed, then return io.EOF. Close() closes the listener and the
// wrapped connection.
func NewConnListener(conn net.Conn) net.Listener {
	next := make(chan net.Conn, 1)
	next <- conn

	return &connListener{
		next: next,
		conn: conn,
		addr: conn.LocalAddr(),
	}
}

// Accept returns the wrapped connection when it is called the first
// time. Later calls to Accept block until the listener is closed, then
// return io.EOF.
func (l *connListener) Accept() (net.Conn, error) {
	conn := <-l.next
	if conn == nil {
		return nil, io.EOF
	}
	return conn, nil
}

// Close closes the listener and the wrapped connection.
func (l *connListener) Close() error {
	l.lock.Lock()
	defer l.lock.Unlock()
	if l.closed {
		return nil
	}
	close(l.next)
	l.closed = true
	return l.conn.Close()
}

// Addr returns the local address of the wrapped connection.
func (l *connListener) Addr() net.Addr {
	return l.addr
}
