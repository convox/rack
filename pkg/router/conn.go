package router

import (
	"net"
	"time"
)

// nopDeadlineConn wraps a net.Conn with Deadline related methods performing a no-op
// Useful with net.Pipe which doesn't support deadline
type nopDeadlineConn struct {
	net.Conn
}

// SetDeadline is a no-op method
func (c *nopDeadlineConn) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline is a no-op method
func (c *nopDeadlineConn) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline is a no-op method
func (c *nopDeadlineConn) SetWriteDeadline(t time.Time) error {
	return nil
}
