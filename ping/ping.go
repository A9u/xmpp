// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package ping implements XEP-0199: XMPP Ping.
package ping // import "mellium.im/xmpp/ping"

import (
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

// NS is the XML namespace used by XMPP pings. It is provided as a convenience.
const NS = `urn:xmpp:ping`

// IQ returns an xml.TokenReader that outputs a new IQ stanza with a ping
// payload.
func IQ(to *jid.JID) xml.TokenReader {
	start := xml.StartElement{Name: xml.Name{Local: "ping", Space: NS}}
	return stanza.WrapIQ(&stanza.IQ{
		To:   to,
		Type: stanza.GetIQ,
	}, xmlstream.Wrap(nil, start))
}
