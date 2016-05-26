// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"encoding/xml"
)

// Presence is an XMPP stanza that is used as an indication that an entity is
// available for communication. It is used to set a status message, broadcast
// availability, and advertise entity capabilities. It can be directed
// (one-to-one), or used as a broadcast mechanism (one-to-many).
type Presence struct {
	stanza

	XMLName xml.Name `xml:"presence"`
}

type presenceType int

const (
	// NoTypePresence is a special type that indicates that a stanza is a presence
	// stanza without a defined type (indicating availability on the network).
	NoTypePresence presenceType = iota

	// An ErrorPresence indicates that an error has occurred regarding processing
	// of a previously sent presence stanza; if the presence stanza is of type
	// "error", it MUST include an <error/> child element
	ErrorPresence presenceType = iota

	// A ProbePresence is a request for an entity's current presence. It should
	// generally only be generated and sent by servers on behalf of a user.
	ProbePresence

	// A SubscribePresence is sent when the sender wishes to subscribe to the
	// recipient's presence.
	SubscribePresence

	// A SubscribedPresence indicates that the sender has allowed the recipient to
	// receive future presence broadcasts.
	SubscribedPresence

	// An UnavailablePresence indicates that the sender is no longer available for
	// communication.
	UnavailablePresence

	// An UnsubscribePresence indicates that the sender is unsubscribing from the
	// receiver's presence.
	UnsubscribePresence

	// An UnsubscribedPresence indicates that the subscription request has been
	// denied, or a previously granted subscription has been revoked.
	UnsubscribedPresence
)
