// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"net"
	"strconv"
	"time"

	"bitbucket.org/mellium/xmpp/jid"
	"golang.org/x/net/context"
)

// Dial connects to the address on the named network.
//
// Known networks are "xmpp-cient" (C2S connections) and "xmpp-server" (S2S
// connections).
func Dial(network string, laddr, raddr *jid.JID) (*Conn, error) {
	var d Dialer
	return d.Dial(network, laddr, raddr)
}

// A Dialer contains options for connecting to an XMPP address.
//
// The zero value for each field is equivalent to dialing without that option.
// Dialing with the zero value of Dialer is therefore equivalent to just calling
// the Dial function.
type Dialer struct {
	net.Dialer
}

// Copied from the net package in the standard library. Copyright The Go
// Authors.
func minNonzeroTime(a, b time.Time) time.Time {
	if a.IsZero() {
		return b
	}
	if b.IsZero() || a.Before(b) {
		return a
	}
	return b
}

// Copied from the net package in the standard library. Copyright The Go
// Authors.
//
// deadline returns the earliest of:
//   - now+Timeout
//   - d.Deadline
//   - the context's deadline
// Or zero, if none of Timeout, Deadline, or context's deadline is set.
func (d *Dialer) deadline(ctx context.Context, now time.Time) (earliest time.Time) {
	if d.Timeout != 0 { // including negative, for historical reasons
		earliest = now.Add(d.Timeout)
	}
	if d, ok := ctx.Deadline(); ok {
		earliest = minNonzeroTime(earliest, d)
	}
	return minNonzeroTime(earliest, d.Deadline)
}

// Dial connects to the address on the named network.
//
// See func Dial for a description of the network and address parameters.
func (d *Dialer) Dial(network string, laddr, raddr *jid.JID) (*Conn, error) {
	return d.DialContext(context.Background(), network, laddr, raddr)
}

// DialContext connects to the address on the named network using
// the provided context. See func Dial for a description of the network and
// address parameters.
//
// The provided Context must be non-nil. If the context expires before
// the connection is complete, an error is returned. Once successfully
// connected, any expiration of the context will not affect the
// connection.
//
// See func Dial for a description of the network and address
// parameters.
func (d *Dialer) DialContext(
	ctx context.Context, network string, laddr, raddr *jid.JID) (*Conn, error) {
	if ctx == nil {
		panic("xmpp.DialContext: nil context")
	}

	deadline := d.deadline(ctx, time.Now())
	if !deadline.IsZero() {
		if d, ok := ctx.Deadline(); !ok || deadline.Before(d) {
			subCtx, cancel := context.WithDeadline(ctx, deadline)
			defer cancel()
			ctx = subCtx
		}
	}
	if oldCancel := d.Cancel; oldCancel != nil {
		subCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		go func() {
			select {
			case <-oldCancel:
				cancel()
			case <-subCtx.Done():
			}
		}()
		ctx = subCtx
	}

	c := &Conn{
		laddr:   laddr,
		network: network,
	}
	// If we're a client and no remote address was specified, connect to the
	// domain in the local address (our JID).
	if raddr == nil && network == "xmpp-client" {
		c.raddr = laddr.Domain()
	} else {
		c.raddr = raddr
	}

	_, addrs, err := net.LookupSRV(network, "tcp", laddr.Domainpart())
	if err != nil {
		// Use domain and default port.
		p, err := net.LookupPort("tcp", network)
		if err != nil {
			return nil, err
		}
		addrs = []*net.SRV{{
			Target: laddr.Domainpart(),
			Port:   uint16(p),
		}}
	}

	// Try dialing all of the SRV records we know about, breaking as soon as the
	// connection is established.
	for _, addr := range addrs {
		if conn, e := d.Dialer.Dial(
			"tcp", net.JoinHostPort(
				addr.Target, strconv.FormatUint(uint64(addr.Port), 10),
			),
		); e != nil {
			err = e
			continue
		} else {
			err = nil
			c.conn = conn
			break
		}
	}
	if err != nil {
		return nil, err
	}

	return c, nil
}
