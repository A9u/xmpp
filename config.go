// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"golang.org/x/text/language"
	"mellium.im/xmpp/jid"
)

// Config represents the configuration of an XMPP session.
type Config struct {
	// An XMPP server address.
	Location *jid.JID

	// An XMPP connection origin (local address). If the origin has a resource
	// part and this is a client config, the given resource will be requested (but
	// not necessarily assigned) during the initial connection handshake.
	// Generally it is recommended to leave this a bare JID and let the server
	// assign a resource part.
	Origin *jid.JID

	// The default language for any streams constructed using this config.
	Lang language.Tag

	// The authorization identity, and password to authenticate with.
	// Identity is used when a user wants to act on behalf of another user. For
	// instance, an admin might want to log in as another user to help them
	// troubleshoot an issue. Normally it is left blank and the localpart of the
	// Origin JID is used.
	Identity, Password string
}

// NewClientConfig constructs a new client-to-server session configuration with
// sane defaults.
func NewClientConfig(origin *jid.JID) (c *Config) {
	c = NewServerConfig(origin.Domain(), origin)
	return c
}

// NewServerConfig constructs a new server-to-server session configuration with
// sane defaults.
func NewServerConfig(location, origin *jid.JID) (c *Config) {
	c = &Config{
		Location: location,
		Origin:   origin,
	}
	return c
}
