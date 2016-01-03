// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package stream

import (
	"net"

	"golang.org/x/text/language"
)

// Option's can be used to configure the stream.
type Option func(*options)
type options struct {
	lang  language.Tag
	conn  net.Conn
	xmlns string
	id    string
}

func getOpts(o ...Option) (res options) {
	for _, f := range o {
		f(&res)
	}
	return
}

// The Language option specifies the default language for the stream. Clients
// that support multiple languages will assume that all messages, alerts,
// and other textual data in the stream is in the given language unless it is
// specifically overridden.
func Language(l language.Tag) Option {
	return func(o *options) {
		o.lang = l
	}
}

// Conn is the connection which the stream will use for sending and receiving
// data.
func Conn(c net.Conn) Option {
	return func(o *options) {
		o.conn = c
	}
}
