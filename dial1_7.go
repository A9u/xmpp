// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

// +build go1.7

package xmpp

import (
	"context"
	"net"
	"strconv"
)

func (d *Dialer) dial(
	ctx context.Context, network string, config *Config) (*Conn, error) {
	if ctx == nil {
		panic("xmpp.Dial: nil context")
	}

	if d.NoLookup {
		p, err := lookupPort(network, config.connType())
		if err != nil {
			return nil, err
		}
		conn, err := d.Dialer.DialContext(ctx, network, net.JoinHostPort(
			config.Location.Domainpart(),
			strconv.FormatUint(uint64(p), 10),
		))
		if err != nil {
			return nil, err
		}
		return NewConn(ctx, config, conn)
	}

	addrs, err := lookupService(config.connType(), network, config.Location)
	if err != nil {
		return nil, err
	}

	// Try dialing all of the SRV records we know about, breaking as soon as the
	// connection is established.
	for _, addr := range addrs {
		if conn, e := d.Dialer.DialContext(
			ctx, network, net.JoinHostPort(
				addr.Target, strconv.FormatUint(uint64(addr.Port), 10),
			),
		); e != nil {
			err = e
			continue
		} else {
			return NewConn(ctx, config, conn)
		}
	}
	return nil, err
}
