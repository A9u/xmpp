// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"mellium.im/xmpp/jid"
)

// A Conn represents an XMPP connection that can perform SRV lookups for a given
// server and connect to the correct ports.
type Conn struct {
	config *Config
	rwc    io.ReadWriteCloser

	// If the initial rwc is a conn, save a reference to that as well so that we
	// can set deadlines on it later even if the rwc is upgraded.
	conn net.Conn

	state SessionState

	// The actual origin of this conn (we don't want to mutate the config, so if
	// this origin exists and is different from the one in config, eg. because the
	// server did not assign us the resourcepart we requested, this is canonical).
	origin *jid.JID

	// The stream feature namespaces advertised for the current streams.
	features map[string]interface{}
	flock    sync.Mutex

	// The negotiated features (by namespace) for the current session.
	negotiated map[string]struct{}

	in struct {
		sync.Mutex
		stream
		d *xml.Decoder
	}
	out struct {
		sync.Mutex
		stream
		e *xml.Encoder
	}
}

// Feature checks if a feature with the given namespace was advertised
// by the server for the current stream. If it was data will be the canonical
// representation of the feature as returned by the feature's Parse function.
func (c *Conn) Feature(namespace string) (data interface{}, ok bool) {
	c.flock.Lock()
	defer c.flock.Unlock()

	// TODO: Make the features struct actually store the parsed representation.
	data, ok = c.features[namespace]
	return
}

// NewConn attempts to use an existing connection (or any io.ReadWriteCloser) to
// negotiate an XMPP session based on the given config. If the provided context
// is canceled before stream negotiation is complete an error is returned. After
// stream negotiation if the context is canceled it has no effect.
func NewConn(ctx context.Context, config *Config, rwc io.ReadWriteCloser) (*Conn, error) {
	c := &Conn{
		config: config,
	}

	if conn, ok := rwc.(net.Conn); ok {
		c.conn = conn
	}

	return c, c.negotiateStreams(ctx, rwc)
}

// Raw returns the Conn's backing net.Conn or other ReadWriteCloser.
func (c *Conn) Raw() io.ReadWriteCloser {
	return c.rwc
}

// Decoder returns the XML decoder that was used to negotiate the latest stream.
func (c *Conn) Decoder() *xml.Decoder {
	return c.in.d
}

// Encoder returns the XML encoder that was used to negotiate the latest stream.
func (c *Conn) Encoder() *xml.Encoder {
	return c.out.e
}

// Config returns the connections config.
func (c *Conn) Config() *Config {
	return c.config
}

// Read reads data from the connection.
func (c *Conn) Read(b []byte) (n int, err error) {
	c.in.Lock()
	defer c.in.Unlock()

	if c.state&InputStreamClosed == InputStreamClosed {
		return 0, errors.New("XML input stream is closed")
	}

	return c.rwc.Read(b)
}

// Write writes data to the connection.
func (c *Conn) Write(b []byte) (n int, err error) {
	c.out.Lock()
	defer c.out.Unlock()

	if c.state&OutputStreamClosed == OutputStreamClosed {
		return 0, errors.New("XML output stream is closed")
	}

	return c.rwc.Write(b)
}

// Close closes the underlying connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (c *Conn) Close() error {
	return c.rwc.Close()
}

// State returns the current state of the session. For more information, see the
// SessionState type.
func (c *Conn) State() SessionState {
	return c.state
}

// LocalAddr returns the Origin address for initiated connections, or the
// Location for received connections.
func (c *Conn) LocalAddr() net.Addr {
	if (c.state & Received) == Received {
		return c.config.Location
	}
	if c.origin != nil {
		return c.origin
	}
	return c.config.Origin
}

// RemoteAddr returns the Location address for initiated connections, or the
// Origin address for received connections.
func (c *Conn) RemoteAddr() net.Addr {
	if (c.state & Received) == Received {
		return c.config.Origin
	}
	return c.config.Location
}

var errSetDeadline = errors.New("xmpp: cannot set deadline: not using a net.Conn")

// SetDeadline sets the read and write deadlines associated with the connection.
// It is equivalent to calling both SetReadDeadline and SetWriteDeadline.
//
// A deadline is an absolute time after which I/O operations fail with a timeout
// (see type Error) instead of blocking. The deadline applies to all future I/O,
// not just the immediately following call to Read or Write.
//
// An idle timeout can be implemented by repeatedly extending the deadline after
// successful Read or Write calls.
//
// A zero value for t means I/O operations will not time out.
func (c *Conn) SetDeadline(t time.Time) error {
	if c.conn != nil {
		return c.conn.SetDeadline(t)
	}
	return errSetDeadline
}

// SetReadDeadline sets the deadline for future Read calls. A zero value for t
// means Read will not time out.
func (c *Conn) SetReadDeadline(t time.Time) error {
	if c.conn != nil {
		return c.conn.SetReadDeadline(t)
	}
	return errSetDeadline
}

// SetWriteDeadline sets the deadline for future Write calls. Even if write
// times out, it may return n > 0, indicating that some of the data was
// successfully written. A zero value for t means Write will not time out.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	if c.conn != nil {
		return c.conn.SetWriteDeadline(t)
	}
	return errSetDeadline
}
