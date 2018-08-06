// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"crypto/tls"
	"io"
	"net"
	"time"
)

// Conn is a net.Conn created for the purpose of establishing an XMPP session.
type Conn struct {
	tlsConn *tls.Conn
	c       net.Conn
	rw      io.ReadWriter
	close   func() error
}

// newConn wraps an io.ReadWriter in a Conn.
func newConn(rw io.ReadWriter) *Conn {
	nc := &Conn{rw: rw}

	switch typrw := rw.(type) {
	case *Conn:
		return typrw
	case *tls.Conn:
		nc.tlsConn = typrw
		nc.c = typrw
	case net.Conn:
		nc.c = typrw
	}

	return nc
}

// ConnectionState returns basic TLS details about the connection if TLS has
// been negotiated. If TLS has not been negotiated, ok is false.
func (c *Conn) ConnectionState() (connState tls.ConnectionState, ok bool) {
	if c.tlsConn != nil {
		return c.tlsConn.ConnectionState(), true
	}
	return connState, false
}

// Close closes the connection.
func (c *Conn) Close() error {
	return c.c.Close()
}

// LocalAddr returns the local network address.
func (c *Conn) LocalAddr() net.Addr {
	return c.c.LocalAddr()
}

// Read can be made to time out and return a net.Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
func (c *Conn) Read(b []byte) (n int, err error) {
	return c.rw.Read(b)
}

// RemoteAddr returns the remote network address.
func (c *Conn) RemoteAddr() net.Addr {
	return c.c.RemoteAddr()
}

// SetDeadline sets the read and write deadlines associated with the connection.
// A zero value for t means Read and Write will not time out.
// After a Write has timed out, the TLS state is corrupt and all future writes
// will return the same error.
func (c *Conn) SetDeadline(t time.Time) error {
	return c.c.SetDeadline(t)
}

// SetReadDeadline sets the read deadline on the underlying connection.
// A zero value for t means Read will not time out.
func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.c.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline on the underlying connection.
// A zero value for t means Write will not time out.
// After a Write has timed out, the TLS state is corrupt and all future writes
// will return the same error.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	return c.c.SetWriteDeadline(t)
}

// Write writes data to the connection.
func (c *Conn) Write(b []byte) (int, error) {
	return c.rw.Write(b)
}
