// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"encoding/xml"

	"bitbucket.org/mellium/xmpp/jid"
)

type Stream struct {
	to      jid.JID
	from    jid.JID
	version string
	xmlns   string
	lang    string
	id      string
}

// A StreamError represents an unrecoverable stream-level error. If a stream
// error is received, the stream will be immediately closed.
type StreamError struct {
	Err xml.Name `xml:",any"`
}

// Error satisfies the builtin error interface and returns the name of the
// StreamError. For instance, given the error:
//
//     <stream:error>
//       <restricted-xml xmlns="urn:ietf:params:xml:ns:xmpp-streams"/>
//     </stream:error>
//
// Error() would return "restricted-xml".
func (e *StreamError) Error() string {
	return e.Err.Local
}

// MarshalXML satisfies the xml package's Marshaler interface and allows
// StreamError's to be correctly marshaled back into XML.
func (s *StreamError) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	e.EncodeElement(struct {
		Err struct{ XMLName xml.Name }
	}{
		struct{ XMLName xml.Name }{s.Err},
	}, xml.StartElement{
		xml.Name{"", "stream:error"},
		[]xml.Attr{},
	})
	return nil
}

// func NewStream(
// 	to, from jid.JID,
// 	lang language.Tag,
// 	f ...StreamFeature) Stream {
// 	return Stream{}
// }

// StreamFromStartElement constructs a new Stream from the given
// xml.StartElement (which must be of the form <stream:stream>).
// func StreamFromStartElement(start xml.StartElement) (*Stream, error) {
//
// 	if start.Name.Local != "stream" || start.Name.Space != "stream" {
// 		return nil, errors.New("Start element must be stream:stream")
// 	}
//
// 	stream := &Stream{}
// 	for _, attr := range start.Attr {
// 		switch attr.Name.Local {
// 		case "from":
// 			j, err := jid.SafeFromString(attr.Value)
// 			if err != nil {
// 				return nil, err
// 			}
// 			stream.from = j
// 		case "to":
// 			j, err := jid.SafeFromString(attr.Value)
// 			if err != nil {
// 				return nil, err
// 			}
// 			stream.to = j
// 		case "xmlns":
// 			stream.xmlns = attr.Value
// 		case "lang":
// 			if attr.Name.Space == "xml" {
// 				stream.lang = attr.Value
// 			}
// 		case "id":
// 			stream.id = attr.Value
// 		}
// 	}
//
// 	return stream, nil
// }
//
// // StartElement creates an XML start element from the given stream which is
// // suitable for starting an XMPP stream.
// func (s *Stream) StartElement() xml.StartElement {
// 	return xml.StartElement{
// 		Name: xml.Name{"stream", "stream"},
// 		Attr: []xml.Attr{
// 			xml.Attr{
// 				xml.Name{"", "to"},
// 				s.to.String(),
// 			},
// 			xml.Attr{
// 				xml.Name{"", "from"},
// 				s.from.String(),
// 			},
// 			xml.Attr{
// 				xml.Name{"", "version"},
// 				s.version,
// 			},
// 			xml.Attr{
// 				xml.Name{"xml", "lang"},
// 				s.lang,
// 			},
// 			xml.Attr{
// 				xml.Name{"", "id"},
// 				s.id,
// 			},
// 			xml.Attr{
// 				xml.Name{"", "xmlns"},
// 				s.xmlns,
// 			},
// 		},
// 	}
// }

// func (s *Stream) Handle(encoder *xml.Encoder, decoder *xml.Decoder) error {
// 	return errors.New("Test me")
// }
