// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package jid

import (
	"testing"
)

func BenchmarkSplit(b *testing.B) {
	for i := 0; i < b.N; i++ {
		SplitString("user@example.com/resource")
	}
}

func BenchmarkParseString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseString("user@example.com/resource")
	}
}

func BenchmarkParseStringIPv4(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseString("user@127.0.0.1/resource")
	}
}

func BenchmarkParseStringIPv6(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseString("user@[::1]/resource")
	}
}

func BenchmarkNew(b *testing.B) {
	for i := 0; i < b.N; i++ {
		New("user", "example.com", "resource")
	}
}

func BenchmarkCopy(b *testing.B) {
	j := &JID{"user", "example.com", "resource"}
	for i := 0; i < b.N; i++ {
		j.Copy()
	}
}

func BenchmarkBare(b *testing.B) {
	j := &JID{"user", "example.com", "resource"}
	for i := 0; i < b.N; i++ {
		j.Bare()
	}
}

func BenchmarkString(b *testing.B) {
	j := &JID{"user", "example.com", "resource"}
	for i := 0; i < b.N; i++ {
		j.String()
	}
}
