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
	"io"

	"mellium.im/sasl"
	"mellium.im/xmpp/internal/saslerr"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/ns"
	"mellium.im/xmpp/streamerror"
)

// BUG(ssw): SASL feature does not have security layer byte precision.

// SASL returns a stream feature for performing authentication using the Simple
// Authentication and Security Layer (SASL) as defined in RFC 4422. It panics if
// no mechanisms are specified. The order in which mechanisms are specified will
// be the prefered order, so stronger mechanisms should be listed first.
func SASL(mechanisms ...sasl.Mechanism) StreamFeature {
	if len(mechanisms) == 0 {
		panic("xmpp: Must specify at least 1 SASL mechanism")
	}
	return StreamFeature{
		Name:       xml.Name{Space: ns.SASL, Local: "mechanisms"},
		Necessary:  Secure,
		Prohibited: Authn,
		List: func(ctx context.Context, e *xml.Encoder, start xml.StartElement) (req bool, err error) {
			req = true
			if err = e.EncodeToken(start); err != nil {
				return
			}

			startMechanism := xml.StartElement{Name: xml.Name{Space: "", Local: "mechanism"}}
			for _, m := range mechanisms {
				select {
				case <-ctx.Done():
					return true, ctx.Err()
				default:
				}

				if err = e.EncodeToken(startMechanism); err != nil {
					return
				}
				if err = e.EncodeToken(xml.CharData(m.Name)); err != nil {
					return
				}
				if err = e.EncodeToken(startMechanism.End()); err != nil {
					return
				}
			}
			if err = e.EncodeToken(start.End()); err != nil {
				return
			}
			err = e.Flush()
			return
		},
		Parse: func(ctx context.Context, d *xml.Decoder, start *xml.StartElement) (bool, interface{}, error) {
			parsed := struct {
				XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-sasl mechanisms"`
				List    []string `xml:"urn:ietf:params:xml:ns:xmpp-sasl mechanism"`
			}{}
			err := d.DecodeElement(&parsed, start)
			return true, parsed.List, err
		},
		Negotiate: func(ctx context.Context, conn *Conn, data interface{}) (mask SessionState, rwc io.ReadWriteCloser, err error) {
			if (conn.state & Received) == Received {
				panic("SASL server not yet implemented")
			} else {
				var selected sasl.Mechanism
				// Select a mechanism, prefering the client order.
			selectmechanism:
				for _, m := range mechanisms {
					for _, name := range data.([]string) {
						if name == m.Name {
							selected = m
							break selectmechanism
						}
					}
				}
				// No matching mechanism found…
				if selected.Name == "" {
					return mask, nil, errors.New(`No matching SASL mechanisms found`)
				}

				c := conn.Config()
				opts := []sasl.Option{
					sasl.Authz(c.Identity),
					sasl.Credentials(conn.LocalAddr().(*jid.JID).Localpart(), c.Password),
					sasl.RemoteMechanisms(data.([]string)...),
				}
				if tlsconn, ok := conn.rwc.(*tls.Conn); ok {
					opts = append(opts, sasl.ConnState(tlsconn.ConnectionState()))
				}
				client := sasl.NewClient(selected, opts...)

				more, resp, err := client.Step(nil)
				if err != nil {
					return mask, nil, err
				}

				// RFC6120 §6.4.2:
				//     If the initiating entity needs to send a zero-length initial
				//     response, it MUST transmit the response as a single equals sign
				//     character ("="), which indicates that the response is present but
				//     contains no data.
				if len(resp) == 0 {
					resp = []byte{'='}
				}

				// Send <auth/> and the initial payload to start SASL auth.
				if _, err = fmt.Fprintf(conn,
					`<auth xmlns='urn:ietf:params:xml:ns:xmpp-sasl' mechanism='%s'>%s</auth>`,
					selected.Name, resp,
				); err != nil {
					return mask, nil, err
				}

				// If we're already done after the first step, decode the <success/> or
				// <failure/> before we exit.
				if !more {
					tok, err := conn.in.d.Token()
					if err != nil {
						return mask, nil, err
					}
					if t, ok := tok.(xml.StartElement); ok {
						// TODO: Handle the additional data that could be returned if
						// success?
						_, _, err := decodeSASLChallenge(conn.in.d, t, false)
						if err != nil {
							return mask, nil, err
						}
					} else {
						return mask, nil, streamerror.BadFormat
					}
				}

				success := false
				for more {
					select {
					case <-ctx.Done():
						return mask, nil, ctx.Err()
					default:
					}
					tok, err := conn.in.d.Token()
					if err != nil {
						return mask, nil, err
					}
					var challenge []byte
					if t, ok := tok.(xml.StartElement); ok {
						challenge, success, err = decodeSASLChallenge(conn.in.d, t, true)
						if err != nil {
							return mask, nil, err
						}
					} else {
						return mask, nil, streamerror.BadFormat
					}
					if more, resp, err = client.Step(challenge); err != nil {
						return mask, nil, err
					}
					if !more && success {
						// We're done with SASL and we're successful
						break
					}
					// TODO: What happens if there's more and success (broken server)?
					if _, err = fmt.Fprintf(conn,
						`<response xmlns='urn:ietf:params:xml:ns:xmpp-sasl'>%s</response>`, resp); err != nil {
						return mask, nil, err
					}
				}
				return Authn, conn.Raw(), nil
			}
		},
	}
}

func decodeSASLChallenge(d *xml.Decoder, start xml.StartElement, allowChallenge bool) (challenge []byte, success bool, err error) {
	switch start.Name {
	case xml.Name{Space: ns.SASL, Local: "challenge"}, xml.Name{Space: ns.SASL, Local: "success"}:
		if !allowChallenge && start.Name.Local == "challenge" {
			return nil, false, streamerror.UnsupportedStanzaType
		}
		challenge := struct {
			Data []byte `xml:",chardata"`
		}{}
		if err = d.DecodeElement(&challenge, &start); err != nil {
			return nil, false, err
		}
		return challenge.Data, start.Name.Local == "success", nil
	case xml.Name{Space: ns.SASL, Local: "failure"}:
		fail := saslerr.Failure{}
		if err = d.DecodeElement(&fail, &start); err != nil {
			return nil, false, err
		}
		return nil, false, fail
	default:
		return nil, false, streamerror.UnsupportedStanzaType
	}
}
