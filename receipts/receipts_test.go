// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package receipts_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"strings"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/receipts"
	"mellium.im/xmpp/stanza"
)

// TODO: find a way to test that SendMessageElement actually matches up the
// response correctly (ie. don't timeout, use the test server).
func TestRoundTrip(t *testing.T) {
	h := &receipts.Handler{}

	var req bytes.Buffer
	s := xmpptest.NewSession(0, &req)
	w := s.TokenWriter()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := h.SendMessageElement(ctx, w, nil, stanza.Message{
		ID:   "123",
		Type: stanza.NormalMessage,
	})
	if err != context.Canceled {
		t.Fatalf("unexpected error: %v", err)
	}
	err = w.Flush()
	if err != nil {
		t.Fatalf("error flushing session: %v", err)
	}

	d := xml.NewDecoder(strings.NewReader(req.String()))
	d.DefaultSpace = ns.Server
	tok, _ := d.Token()
	start := tok.(xml.StartElement)
	var b strings.Builder
	e := xml.NewEncoder(&b)

	m := mux.New(receipts.Handle(h))
	err = m.HandleXMPP(struct {
		xml.TokenReader
		xmlstream.Encoder
	}{
		TokenReader: d,
		Encoder:     e,
	}, &start)
	if err != nil {
		t.Errorf("unexpected error in handler: %v", err)
	}
	err = e.Flush()
	if err != nil {
		t.Errorf("unexpected error flushing encoder: %v", err)
	}

	out := b.String()
	const expected = `<message xmlns="jabber:server" type="normal"><received xmlns="urn:xmpp:receipts" id="123"></received></message>`
	if out != expected {
		t.Errorf("got=%s, want=%s", out, expected)
	}
}
