// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"net"
)

var (
	ErrTLSUpgradeFailed = errors.New("The underlying connection cannot be upgraded to TLS")
)

// StartTLS returns a new stream feature that can be used for negotiating TLS.
// For StartTLS to work, the underlying connection must support TLS (it must
// implement net.Conn) and the connections config must have a TLSConfig.
func StartTLS() StreamFeature {
	return StreamFeature{
		Name: xml.Name{Local: "starttls", Space: NSStartTLS},
		Handler: func(ctx context.Context, conn *Conn) (mask SessionState, err error) {
			if _, ok := conn.rwc.(net.Conn); !ok {
				return mask, ErrTLSUpgradeFailed
			}

			if (conn.state & Received) == Received {
				fmt.Fprint(conn, `<proceed xmlns='urn:ietf:params:xml:ns:xmpp-tls'/>`)
				conn.rwc = tls.Server(conn.rwc.(net.Conn), conn.config.TLSConfig)
			} else {
				// Select starttls for negotiation.
				fmt.Fprint(conn, `<starttls xmlns='urn:ietf:params:xml:ns:xmpp-tls'/>`)

				// Receive a <proceed/> or <failure/> response from the server.
				t, err := conn.in.d.Token()
				if err != nil {
					return mask, err
				}
				switch tok := t.(type) {
				case xml.StartElement:
					switch {
					case tok.Name.Space != NSStartTLS:
						return mask, UnsupportedStanzaType
					case tok.Name.Local == "proceed":
						// Skip the </proceed> token.
						if err = conn.in.d.Skip(); err != nil {
							return EndStream, InvalidXML
						}
						conn.rwc = tls.Client(conn.rwc.(net.Conn), conn.config.TLSConfig)
					case tok.Name.Local == "failure":
						// Skip the </failure> token.
						if err = conn.in.d.Skip(); err != nil {
							err = InvalidXML
						}
						// Failure is not an "error", it's expected behavior. The server is
						// telling us to end the stream. However, if we encounter bad XML
						// while skipping the </feailure> token, return that error.
						return EndStream, err
					default:
						return mask, UnsupportedStanzaType
					}
				default:
					return mask, RestrictedXML
				}
			}
			mask = Secure | StreamRestartRequired
			return
		},
		Required: true,
	}
}
