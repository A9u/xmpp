// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"

	"golang.org/x/text/language"
	"mellium.im/xmpp/internal"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stream"
)

const (
	xmlHeader = `<?xml version="1.0" encoding="UTF-8"?>`
)

type streamInfo struct {
	to      *jid.JID
	from    *jid.JID
	id      string
	version internal.Version
	xmlns   string
	lang    language.Tag
}

// This MUST only return stream errors.
// TODO: Is the above true? Just make it return a StreamError?
func streamFromStartElement(s xml.StartElement) (streamInfo, error) {
	streamData := streamInfo{}
	for _, attr := range s.Attr {
		switch attr.Name {
		case xml.Name{Space: "", Local: "to"}:
			streamData.to = &jid.JID{}
			if err := streamData.to.UnmarshalXMLAttr(attr); err != nil {
				return streamData, stream.ImproperAddressing
			}
		case xml.Name{Space: "", Local: "from"}:
			streamData.from = &jid.JID{}
			if err := streamData.from.UnmarshalXMLAttr(attr); err != nil {
				return streamData, stream.ImproperAddressing
			}
		case xml.Name{Space: "", Local: "id"}:
			streamData.id = attr.Value
		case xml.Name{Space: "", Local: "version"}:
			(&streamData.version).UnmarshalXMLAttr(attr)
		case xml.Name{Space: "", Local: "xmlns"}:
			if attr.Value != "jabber:client" && attr.Value != "jabber:server" {
				return streamData, stream.InvalidNamespace
			}
			streamData.xmlns = attr.Value
		case xml.Name{Space: "xmlns", Local: "stream"}:
			if attr.Value != ns.Stream {
				return streamData, stream.InvalidNamespace
			}
		case xml.Name{Space: "xml", Local: "lang"}:
			streamData.lang = language.Make(attr.Value)
		}
	}
	return streamData, nil
}

// Sends a new XML header followed by a stream start element on the given
// io.Writer. We don't use an xml.Encoder both because Go's standard library xml
// package really doesn't like the namespaced stream:stream attribute and
// because we can guarantee well-formedness of the XML with a print in this case
// and printing is much faster than encoding. Afterwards, clear the
// StreamRestartRequired bit and set the output stream information.
func sendNewStream(s *Session, cfg *Config, id string) error {
	streamData := streamInfo{
		to:      cfg.Location,
		from:    cfg.Origin,
		lang:    cfg.Lang,
		version: cfg.Version,
	}
	switch cfg.S2S {
	case true:
		streamData.xmlns = ns.Server
	case false:
		streamData.xmlns = ns.Client
	}

	streamData.id = id
	if id == "" {
		id = " "
	} else {
		id = ` id='` + id + `' `
	}

	_, err := fmt.Fprintf(s.Conn(),
		xmlHeader+`<stream:stream%sto='%s' from='%s' version='%s' xml:lang='%s' xmlns='%s' xmlns:stream='http://etherx.jabber.org/streams'>`,
		id,
		cfg.Location.String(),
		cfg.Origin.String(),
		cfg.Version,
		cfg.Lang,
		streamData.xmlns,
	)
	if err != nil {
		return err
	}

	s.out.streamInfo = streamData
	return nil
}

func expectNewStream(ctx context.Context, s *Session) error {
	var foundHeader bool

	d := s.in.d
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		t, err := d.Token()
		if err != nil {
			return err
		}
		switch tok := t.(type) {
		case xml.StartElement:
			switch {
			case tok.Name.Local == "error" && tok.Name.Space == ns.Stream:
				se := stream.Error{}
				if err := d.DecodeElement(&se, &tok); err != nil {
					return err
				}
				return se
			case tok.Name.Local != "stream":
				return stream.BadFormat
			case tok.Name.Space != ns.Stream:
				return stream.InvalidNamespace
			}

			streamData, err := streamFromStartElement(tok)
			switch {
			case err != nil:
				return err
			case streamData.version != internal.DefaultVersion:
				return stream.UnsupportedVersion
			}

			if (s.state&Received) != Received && streamData.id == "" {
				// if we are the initiating entity and there is no stream ID…
				return stream.BadFormat
			}
			s.in.streamInfo = streamData
			return nil
		case xml.ProcInst:
			// TODO: If version or encoding are declared, validate XML 1.0 and UTF-8
			if !foundHeader && tok.Target == "xml" {
				foundHeader = true
				continue
			}
			return stream.RestrictedXML
		case xml.EndElement:
			return stream.NotWellFormed
		default:
			return stream.RestrictedXML
		}
	}
}

func (s *Session) negotiateStreams(ctx context.Context, rw io.ReadWriter) (err error) {
	// Loop for as long as we're not done negotiating features or a stream restart
	// is still required.
	for done := false; !done || rw != nil; {
		if rw != nil {
			s.features = make(map[string]interface{})
			s.negotiated = make(map[string]struct{})
			s.rw = rw
			s.in.d = xml.NewDecoder(s.rw)
			s.out.e = xml.NewEncoder(s.rw)
			rw = nil

			if (s.state & Received) == Received {
				// If we're the receiving entity wait for a new stream, then send one in
				// response.
				if err = expectNewStream(ctx, s); err != nil {
					return err
				}
				if err = sendNewStream(s, s.config, internal.RandomID()); err != nil {
					return err
				}
			} else {
				// If we're the initiating entity, send a new stream and then wait for
				// one in response.
				if err = sendNewStream(s, s.config, ""); err != nil {
					return err
				}
				if err = expectNewStream(ctx, s); err != nil {
					return err
				}
			}
		}

		if done, rw, err = s.negotiateFeatures(ctx); err != nil {
			return err
		}
	}
	return nil
}
