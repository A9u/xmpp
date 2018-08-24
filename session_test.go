// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
	"mellium.im/xmpp/stream"
)

func TestClosedInputStream(t *testing.T) {
	for i := 0; i <= math.MaxUint8; i++ {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			mask := xmpp.SessionState(i)
			buf := new(bytes.Buffer)
			s := xmpptest.NewSession(mask, buf)

			_, err := s.Token()
			switch {
			case mask&xmpp.InputStreamClosed == xmpp.InputStreamClosed && err != xmpp.ErrInputStreamClosed:
				t.Errorf("Unexpected error: want=`%v', got=`%v'", xmpp.ErrInputStreamClosed, err)
			case mask&xmpp.InputStreamClosed == 0 && err != io.EOF:
				t.Errorf("Unexpected error: `%v'", err)
			}
		})
	}
}

func TestClosedOutputStream(t *testing.T) {
	for i := 0; i <= math.MaxUint8; i++ {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			mask := xmpp.SessionState(i)
			buf := new(bytes.Buffer)
			s := xmpptest.NewSession(mask, buf)

			switch err := s.EncodeToken(xml.CharData("chartoken")); {
			case mask&xmpp.OutputStreamClosed == xmpp.OutputStreamClosed && err != xmpp.ErrOutputStreamClosed:
				t.Errorf("Unexpected error: want=`%v', got=`%v'", xmpp.ErrOutputStreamClosed, err)
			case mask&xmpp.OutputStreamClosed == 0 && err != nil:
				t.Errorf("Unexpected error: `%v'", err)
			}
			switch err := s.Flush(); {
			case mask&xmpp.OutputStreamClosed == xmpp.OutputStreamClosed && err != xmpp.ErrOutputStreamClosed:
				t.Errorf("Unexpected error flushing: want=`%v', got=`%v'", xmpp.ErrOutputStreamClosed, err)
			case mask&xmpp.OutputStreamClosed == 0 && err != nil:
				t.Errorf("Unexpected error: `%v'", err)
			}
		})
	}
}

func TestNilNegotiatorPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic, did not get one")
		}
	}()
	xmpp.NegotiateSession(context.Background(), jid.JID{}, jid.JID{}, nil, nil)
}

var errTestNegotiate = errors.New("a test error")

func errNegotiator(ctx context.Context, session *xmpp.Session, data interface{}) (mask xmpp.SessionState, rw io.ReadWriter, cache interface{}, err error) {
	err = errTestNegotiate
	return
}

type negotiateTestCase struct {
	negotiator xmpp.Negotiator
	in         string
	out        string
	location   jid.JID
	origin     jid.JID
	err        error
}

var negotiateTests = [...]negotiateTestCase{
	0: {negotiator: errNegotiator, err: errTestNegotiate},
	1: {
		negotiator: xmpp.NewNegotiator(xmpp.StreamConfig{
			Features: []xmpp.StreamFeature{xmpp.StartTLS(true, nil)},
		}),
		in:  `<stream:stream id='316732270768047465' version='1.0' xml:lang='en' xmlns:stream='http://etherx.jabber.org/streams' xmlns='jabber:client'><stream:features><other/></stream:features>`,
		out: `<?xml version="1.0" encoding="UTF-8"?><stream:stream to='' from='' version='1.0' xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams'><starttls xmlns='urn:ietf:params:xml:ns:xmpp-tls'/>`,
		err: errors.New("XML syntax error on line 1: unexpected EOF"),
	},
	2: {
		negotiator: xmpp.NewNegotiator(xmpp.StreamConfig{}),
		in:         `<stream:stream id='316732270768047465' version='1.0' xml:lang='en' xmlns:stream='http://etherx.jabber.org/streams' xmlns='jabber:client'><stream:features><other/></stream:features>`,
		out:        `<?xml version="1.0" encoding="UTF-8"?><stream:stream to='' from='' version='1.0' xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams'>`,
		err:        errors.New("xmpp: features advertised out of order"),
	},
}

func TestNegotiator(t *testing.T) {
	for i, tc := range negotiateTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			buf := &bytes.Buffer{}
			rw := struct {
				io.Reader
				io.Writer
			}{
				Reader: strings.NewReader(tc.in),
				Writer: buf,
			}
			_, err := xmpp.NegotiateSession(context.Background(), tc.location, tc.origin, rw, tc.negotiator)
			if ((err == nil || tc.err == nil) && (err != nil || tc.err != nil)) || err.Error() != tc.err.Error() {
				t.Errorf("Unexpected error: want=%q, got=%q", tc.err, err)
			}
			if out := buf.String(); out != tc.out {
				t.Errorf("Unexpected output:\nwant=%q,\n got=%q", tc.out, out)
			}
		})
	}
}

const invalidIQ = `<iq type="error" id="1234"><error type="cancel"><service-unavailable xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></service-unavailable></error></iq>`

var serveTests = [...]struct {
	handler xmpp.Handler
	out     string
	in      string
	err     error
}{
	0: {
		in:  `<test></test>`,
		out: `</stream:stream>`,
		err: io.EOF,
	},
	1: {
		in:  `a`,
		out: `</stream:stream>`,
		err: stream.BadFormat,
	},
	2: {
		in:  `<iq type="get" id="1234"><unknownpayload xmlns="unknown"/></iq>`,
		out: invalidIQ + `</stream:stream>`,
		err: io.EOF,
	},
	3: {
		handler: xmpp.HandlerFunc(func(rw xmlstream.TokenReadWriter, start *xml.StartElement) error {
			_, err := xmlstream.Copy(rw, stanza.WrapIQ(&stanza.IQ{
				ID:   "1234",
				Type: stanza.ResultIQ,
			}, nil))
			return err
		}),
		in:  `<iq type="get" id="1234"><unknownpayload xmlns="unknown"/></iq>`,
		out: `<iq type="result" id="1234"></iq></stream:stream>`,
		err: io.EOF,
	},
	4: {
		handler: xmpp.HandlerFunc(func(rw xmlstream.TokenReadWriter, start *xml.StartElement) error {
			_, err := xmlstream.Copy(rw, stanza.WrapIQ(&stanza.IQ{
				ID:   "wrongid",
				Type: stanza.ResultIQ,
			}, nil))
			return err
		}),
		in:  `<iq type="get" id="1234"><unknownpayload xmlns="unknown"/></iq>`,
		out: `<iq type="result" id="wrongid"></iq>` + invalidIQ + `</stream:stream>`,
		err: io.EOF,
	},
	5: {
		handler: xmpp.HandlerFunc(func(rw xmlstream.TokenReadWriter, start *xml.StartElement) error {
			_, err := xmlstream.Copy(rw, stanza.WrapIQ(&stanza.IQ{
				ID:   "1234",
				Type: stanza.ErrorIQ,
			}, nil))
			return err
		}),
		in:  `<iq type="get" id="1234"><unknownpayload xmlns="unknown"/></iq>`,
		out: `<iq type="error" id="1234"></iq></stream:stream>`,
		err: io.EOF,
	},
	6: {
		handler: xmpp.HandlerFunc(func(rw xmlstream.TokenReadWriter, start *xml.StartElement) error {
			_, err := xmlstream.Copy(rw, stanza.WrapIQ(&stanza.IQ{
				ID:   "1234",
				Type: stanza.GetIQ,
			}, nil))
			return err
		}),
		in:  `<iq type="get" id="1234"><unknownpayload xmlns="unknown"/></iq>`,
		out: `<iq type="get" id="1234"></iq>` + invalidIQ + `</stream:stream>`,
		err: io.EOF,
	},
}

func TestServe(t *testing.T) {
	for i, tc := range serveTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			out := &bytes.Buffer{}
			in := strings.NewReader(tc.in)
			s := xmpptest.NewSession(0, struct {
				io.Reader
				io.Writer
			}{
				Reader: in,
				Writer: out,
			})

			err := s.Serve(tc.handler)
			if err != tc.err {
				t.Errorf("Unexpected error: want=%q, got=%q", tc.err, err)
			}
			if s := out.String(); s != tc.out {
				t.Errorf("Unexpected output:\nwant=%q,\n got=%q", tc.out, s)
			}
			if l := in.Len(); l != 0 {
				t.Errorf("Did not finish read, %d bytes left", l)
			}
		})
	}
}
