// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

// Package ns provides namespace constants that are used by the xmpp package and
// other internal packages.
package ns // import "mellium.im/xmpp/ns"

// Namespaces used by the mellium.im/xmpp package.
const (
	Bind     = "urn:ietf:params:xml:ns:xmpp-bind"
	Client   = "jabber:client"
	SASL     = "urn:ietf:params:xml:ns:xmpp-sasl"
	Server   = "jabber:server"
	Stanza   = "urn:ietf:params:xml:ns:xmpp-stanzas"
	StartTLS = "urn:ietf:params:xml:ns:xmpp-tls"
	Stream   = "http://etherx.jabber.org/streams"
	XML      = "http://www.w3.org/XML/1998/namespace"
)
