// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

const (
	testIQID = "123"
)

type errReader struct{ err error }

func (r errReader) Token() (xml.Token, error) {
	return nil, r.err
}

type toksReader []xml.Token

func (r *toksReader) Token() (xml.Token, error) {
	if len(*r) == 0 {
		return nil, io.EOF
	}

	var t xml.Token
	t, *r = (*r)[0], (*r)[1:]
	return t, nil
}

var (
	errExpected = errors.New("Expected error")
	to          = jid.MustParse("test@example.net")
)

var sendIQTests = [...]struct {
	iq         stanza.IQ
	payload    xml.TokenReader
	err        error
	writesBody bool
	resp       xml.TokenReader
}{
	0: {
		iq:         stanza.IQ{ID: testIQID, Type: stanza.GetIQ},
		writesBody: true,
		resp:       stanza.WrapIQ(stanza.IQ{ID: testIQID, Type: stanza.ResultIQ}, nil),
	},
	1: {
		iq:         stanza.IQ{ID: testIQID, Type: stanza.SetIQ},
		writesBody: true,
		resp:       stanza.WrapIQ(stanza.IQ{ID: testIQID, Type: stanza.ErrorIQ}, nil),
	},
	2: {
		iq:         stanza.IQ{Type: stanza.ResultIQ, ID: testIQID},
		writesBody: true,
	},
	3: {
		iq:         stanza.IQ{Type: stanza.ErrorIQ, ID: testIQID},
		writesBody: true,
	},
}

func TestSendIQ(t *testing.T) {
	for i, tc := range sendIQTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			br := &bytes.Buffer{}
			if tc.resp != nil {
				e := xml.NewEncoder(br)
				_, err := xmlstream.Copy(e, tc.resp)
				if err != nil {
					t.Logf("error responding: %q", err)
				}
				err = e.Flush()
				if err != nil {
					t.Logf("error flushing after responding: %q", err)
				}
			}

			t.Run("SendIQElement", func(t *testing.T) {
				bw := &bytes.Buffer{}
				s := xmpptest.NewSession(0, struct {
					io.Reader
					io.Writer
				}{
					Reader: strings.NewReader(br.String()),
					Writer: bw,
				})
				defer func() {
					if err := s.Close(); err != nil {
						t.Errorf("Error closing session: %q", err)
					}
				}()

				go func() {
					err := s.Serve(nil)
					if err != nil && err != io.EOF {
						panic(err)
					}
				}()

				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
				defer cancel()

				resp, err := s.SendIQElement(ctx, tc.payload, tc.iq)
				if err != tc.err {
					t.Errorf("Unexpected error, want=%q, got=%q", tc.err, err)
				}
				if resp != nil {
					defer func() {
						if err := resp.Close(); err != nil {
							t.Errorf("Error closing response: %q", err)
						}
					}()
				}
				if empty := bw.Len() != 0; tc.writesBody != empty {
					t.Errorf("Unexpected body, want=%t, got=%t", tc.writesBody, empty)
				}
				switch {
				case resp == nil && tc.resp != nil:
					t.Errorf("Expected response, but got none")
				case resp != nil && tc.resp == nil:
					buf := &bytes.Buffer{}
					_, err := xmlstream.Copy(xml.NewEncoder(buf), resp)
					if err != nil {
						t.Errorf("Error encoding unexpected response")
					}
					t.Errorf("Did not expect response, but got: %s", buf.String())
				}
			})
			t.Run("SendIQ", func(t *testing.T) {
				bw := &bytes.Buffer{}
				s := xmpptest.NewSession(0, struct {
					io.Reader
					io.Writer
				}{
					Reader: strings.NewReader(br.String()),
					Writer: bw,
				})
				defer func() {
					if err := s.Close(); err != nil {
						t.Errorf("Error closing session: %q", err)
					}
				}()

				go func() {
					err := s.Serve(nil)
					if err != nil && err != io.EOF {
						panic(err)
					}
				}()

				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
				defer cancel()

				resp, err := s.SendIQ(ctx, stanza.WrapIQ(tc.iq, tc.payload))
				if err != tc.err {
					t.Errorf("Unexpected error, want=%q, got=%q", tc.err, err)
				}
				if resp != nil {
					defer func() {
						if err := resp.Close(); err != nil {
							t.Errorf("Error closing response: %q", err)
						}
					}()
				}
				if empty := bw.Len() != 0; tc.writesBody != empty {
					t.Errorf("Unexpected body, want=%t, got=%t", tc.writesBody, empty)
				}
				switch {
				case resp == nil && tc.resp != nil:
					t.Errorf("Expected response, but got none")
				case resp != nil && tc.resp == nil:
					buf := &bytes.Buffer{}
					_, err := xmlstream.Copy(xml.NewEncoder(buf), resp)
					if err != nil {
						t.Errorf("Error encoding unexpected response")
					}
					t.Errorf("Did not expect response, but got: %s", buf.String())
				}
			})
		})
	}
}

var sendTests = [...]struct {
	r          xml.TokenReader
	err        error
	resp       xml.TokenReader
	writesBody bool
}{
	0: {
		r:   errReader{err: errExpected},
		err: errExpected,
	},
	1: {
		r: &toksReader{
			xml.EndElement{Name: xml.Name{Local: "iq"}},
		},
		err: xmpp.ErrNotStart,
	},
	2: {
		r:          stanza.WrapMessage(to, stanza.NormalMessage, nil),
		writesBody: true,
	},
	3: {
		r:          stanza.WrapPresence(to, stanza.AvailablePresence, nil),
		writesBody: true,
	},
	4: {
		r:          stanza.WrapIQ(stanza.IQ{Type: stanza.ResultIQ}, nil),
		writesBody: true,
	},
	5: {
		r:          stanza.WrapIQ(stanza.IQ{Type: stanza.ErrorIQ}, nil),
		writesBody: true,
	},
}

func TestSend(t *testing.T) {
	for i, tc := range sendTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			br := &bytes.Buffer{}
			bw := &bytes.Buffer{}
			s := xmpptest.NewSession(0, struct {
				io.Reader
				io.Writer
			}{
				Reader: br,
				Writer: bw,
			})
			defer func() {
				if err := s.Close(); err != nil {
					t.Errorf("Error closing session: %q", err)
				}
			}()
			if tc.resp != nil {
				e := xml.NewEncoder(br)
				_, err := xmlstream.Copy(e, tc.resp)
				if err != nil {
					t.Logf("error responding: %q", err)
				}
				err = e.Flush()
				if err != nil {
					t.Logf("error flushing after responding: %q", err)
				}
			}

			go func() {
				err := s.Serve(nil)
				if err != nil && err != io.EOF {
					panic(err)
				}
			}()

			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
			defer cancel()
			err := s.Send(ctx, tc.r)
			if err != tc.err {
				t.Errorf("Unexpected error, want=%q, got=%q", tc.err, err)
			}
			if empty := bw.Len() != 0; tc.writesBody != empty {
				t.Errorf("Unexpected body, want=%t, got=%t", tc.writesBody, empty)
			}
		})
	}
}
