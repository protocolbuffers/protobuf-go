// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package protopath provides functionality for
// representing a sequence of protobuf reflection operations on a message.
package protopath

import (
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestScanNumberEscape(t *testing.T) {
	tcs := []struct {
		name string
		s    *scanner
		re   *regexp.Regexp
		base int
		want escapedRune
		spos int
	}{
		{
			name: "escape \\xHH",
			s:    &scanner{buf: []byte("3c")},
			re:   hex12Re,
			base: 16,
			want: escapedRune{Rune: '<', Valid: true},
			spos: 2,
		},
		{
			name: "escape \\xH",
			s:    &scanner{buf: []byte("c")},
			re:   hex12Re,
			base: 16,
			want: escapedRune{Rune: rune(0xc), Valid: true},
			spos: 1,
		},
		{
			name: "escape \\123",
			s:    &scanner{buf: []byte("123")},
			re:   oct13Re,
			base: 8,
			want: escapedRune{Rune: 'S', Valid: true},
			spos: 3,
		},
		{
			name: "too short",
			s:    &scanner{buf: []byte("32")},
			re:   hex4Re,
			base: 16,
			want: escapedRune{},
			spos: 1, // Consumes only 1 character since 32 is not a HHHHH sequence.
		},
		{
			name: "threshold terminated by unexpected character",
			s:    &scanner{buf: []byte("118")},
			re:   oct13Re,
			base: 8,
			want: escapedRune{Rune: rune(9), Valid: true},
			spos: 2, // Not past '8' since that is still part of the string.
		},
		{
			name: "wrong chars",
			s:    &scanner{buf: []byte("ab")},
			re:   oct13Re,
			base: 8,
			want: escapedRune{},
			spos: 1,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.s.number(0, tc.re, tc.base)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("%v.number(0, %v, %d) diff (-want +got) %s", tc.s, tc.re, tc.base, diff)
			}
			if tc.s.pos != tc.spos {
				t.Errorf("%v.number(0, %v, %d) final pos %d want %d", tc.s, tc.re, tc.base, tc.s.pos, tc.spos)
			}
		})
	}
}

func TestEscape(t *testing.T) {
	tcs := []struct {
		name string
		s    *scanner
		want escapedRune
		spos int
	}{
		{
			name: "wrong chars",
			s:    &scanner{buf: []byte(`\db`)},
			want: escapedRune{Pos: 1},
			spos: 2,
		},
		{
			name: "HEX",
			s:    &scanner{buf: []byte(`\X7c`)},
			want: escapedRune{Rune: '|', Pos: 0, Valid: true},
			spos: 4,
		},
		{
			name: "hex",
			s:    &scanner{buf: []byte(`\X71`)},
			want: escapedRune{Rune: 'q', Pos: 0, Valid: true},
			spos: 4,
		},
		{
			name: "Unicode",
			s:    &scanner{buf: []byte(`\U0001f389`)},
			want: escapedRune{Rune: rune('🎉'), Pos: 0, Valid: true},
			spos: 10,
		},
		{
			name: "Unicode too short",
			s:    &scanner{buf: []byte(`\U1f389 hmm`)},
			want: escapedRune{},
			spos: 3, // consumes '\', 'U' to multiplex, then 1f389 is too short, so consumes '1' to make progress.
		},
		{
			name: "unicode",
			s:    &scanner{buf: []byte(`\u007c hmm`)},
			want: escapedRune{Rune: '|', Pos: 0, Valid: true},
			spos: 6,
		},
		{
			name: "unicode too short",
			s:    &scanner{buf: []byte(`\u7c hmm`)},
			want: escapedRune{},
			spos: 3,
		},
		{
			name: "unicode too short by eof",
			s:    &scanner{buf: []byte(`\u7c`)},
			want: escapedRune{},
			// '\' and 'u' are consumed to multiplex, then 7c is too short, so '7' is consumed to make progress.
			spos: 3,
		},
		{
			name: "octal",
			s:    &scanner{buf: []byte(`\101`)},
			want: escapedRune{Rune: 'A', Pos: 0, Valid: true},
			spos: 4,
		},
		{name: "alert", s: &scanner{buf: []byte(`\a`)}, want: escapedRune{Rune: '\a', Valid: true}, spos: 2},
		{name: "backspace", s: &scanner{buf: []byte(`\b`)}, want: escapedRune{Rune: '\b', Valid: true}, spos: 2},
		{name: "formfeed", s: &scanner{buf: []byte(`\f`)}, want: escapedRune{Rune: '\f', Valid: true}, spos: 2},
		{name: "newline", s: &scanner{buf: []byte(`\n`)}, want: escapedRune{Rune: '\n', Valid: true}, spos: 2},
		{name: "carriage return", s: &scanner{buf: []byte(`\r`)}, want: escapedRune{Rune: '\r', Valid: true}, spos: 2},
		{name: "tab", s: &scanner{buf: []byte(`\t`)}, want: escapedRune{Rune: '\t', Valid: true}, spos: 2},
		{name: "vertical tab", s: &scanner{buf: []byte(`\v`)}, want: escapedRune{Rune: '\v', Valid: true}, spos: 2},
		{name: "backslash", s: &scanner{buf: []byte(`\\`)}, want: escapedRune{Rune: '\\', Valid: true}, spos: 2},
		{name: "single quote", s: &scanner{buf: []byte(`\'`)}, want: escapedRune{Rune: '\'', Valid: true}, spos: 2},
		{name: "double quote", s: &scanner{buf: []byte(`\"`)}, want: escapedRune{Rune: '"', Valid: true}, spos: 2},
		{name: "question mark", s: &scanner{buf: []byte(`\?`)}, want: escapedRune{Rune: '?', Valid: true}, spos: 2},
		{name: "bad escape", s: &scanner{buf: []byte(`\q`)}, want: escapedRune{Pos: 1}, spos: 2},
		{name: "eof", s: &scanner{buf: []byte(`\`)}, want: escapedRune{Pos: 1}, spos: 1},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.s.escape()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("%v.escape() diff %s (-want +got)", tc.s, diff)
			}
			if tc.s.pos != tc.spos {
				t.Errorf("%v.escape() final pos %d want = %d", tc.s, tc.s.pos, tc.spos)
			}
		})
	}
}

func TestStringLiteral(t *testing.T) {
	tcs := []struct {
		name string
		s    *scanner
		want *token
		spos int
	}{
		{
			name: "double quote",
			s:    &scanner{buf: []byte(`"regular"`)},
			want: &token{Pos: 0, Kind: strlit, Text: "regular"},
			spos: 9,
		},
		{
			name: "single quote fun",
			s:    &scanner{buf: []byte(` 'not so \U0001f389 now'`), pos: 1},
			want: &token{Pos: 1, Text: "not so 🎉 now", Kind: strlit},
			spos: 24,
		},
		{
			name: "null in the string",
			s:    &scanner{buf: []byte{'\'', 'a', 0, '\''}},
			want: &token{Pos: 2, Kind: illegal, Text: `'\x00'`},
			spos: 3,
		},
		{
			name: "newline in the string",
			s:    &scanner{buf: []byte("\"oh\nyeah\"")},
			want: &token{Pos: 3, Kind: illegal, Text: `'\n'`},
			spos: 4,
		},
		{
			name: "illegal termination",
			s:    &scanner{buf: []byte("\"")},
			want: &token{Pos: 1, Kind: illegal, Text: `"`},
			spos: 1,
		},
		{
			name: "bad escape termination",
			s:    &scanner{buf: []byte(`"ab\ `)},
			want: &token{Pos: 4, Kind: illegal, Text: `"ab\ `},
			spos: 5,
		},
		{
			name: "bad codepoint", // Anything above 0x10FFFF
			s:    &scanner{buf: []byte{'"', 0xff, 0xff, 0xff, 0xff, '"'}},
			want: &token{Pos: 1, Kind: illegal},
			spos: 2,
		},
		{name: "eof",
			s: &scanner{buf: []byte(`" \`)},
			// eof position is at fault.
			want: &token{Pos: 3, Kind: illegal, Text: `" \`},
			spos: 3,
		},
		{name: "bad escape",
			s: &scanner{buf: []byte(`"\ `)},
			// illegal escape is at fault
			want: &token{Pos: 2, Kind: illegal, Text: `"\ `},
			spos: 3,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.s.string()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("%v.string() diff (-want +got) %s", tc.s, diff)
			}
			if tc.s.pos != tc.spos {
				t.Errorf("%v.string() final pos %d want = %d", tc.s, tc.s.pos, tc.spos)
			}
		})
	}
}

func TestScanner(t *testing.T) {
	tcs := []struct {
		name string
		s    *scanner
		want *token
		spos int
	}{
		{name: "oparen", s: &scanner{buf: []byte{'('}}, want: &token{Kind: oparen}, spos: 1},
		{name: "cparen", s: &scanner{buf: []byte{')'}}, want: &token{Kind: cparen}, spos: 1},
		{name: "obrack", s: &scanner{buf: []byte{'['}}, want: &token{Kind: obrack}, spos: 1},
		{name: "cbrack", s: &scanner{buf: []byte{']'}}, want: &token{Kind: cbrack}, spos: 1},
		{name: "dot", s: &scanner{buf: []byte{'.'}}, want: &token{Kind: dot}, spos: 1},
		{name: "whitespace illegal", s: &scanner{buf: []byte{' '}}, want: &token{Kind: illegal, Text: "' '"}, spos: 1},
		{name: "ident",
			s:    &scanner{buf: []byte("01234abcd123"), pos: 5},
			want: &token{Kind: ident, Text: "abcd123", Pos: 5},
			spos: 12,
		},
		{name: "strlit",
			s:    &scanner{buf: []byte(`oh "this is a \n string" oh`), pos: 3},
			want: &token{Kind: strlit, Text: "this is a \n string", Pos: 3},
			spos: 24,
		},
		{name: "intlit dec",
			s:    &scanner{buf: []byte("123")},
			want: &token{Kind: intlit, Text: "123"},
			spos: 3,
		},
		{name: "intlit negative dec",
			s:    &scanner{buf: []byte("-123")},
			want: &token{Kind: intlit, Text: "-123"},
			spos: 4,
		},
		{name: "intlit oct",
			s:    &scanner{buf: []byte("0777")},
			want: &token{Kind: intlit, Text: "0777"},
			spos: 4,
		},
		{name: "intlit hex",
			s:    &scanner{buf: []byte("0xabc89wahooo")},
			want: &token{Kind: intlit, Text: "0xabc89"},
			spos: 7,
		},
		{name: "escape hex string",
			s:    &scanner{buf: []byte(`123'this\x20was a space' ha`), pos: 3},
			spos: 24,
			want: &token{Kind: strlit, Pos: 3, Text: "this was a space"},
		},
		{name: "unclosed string",
			s:    &scanner{buf: []byte(`'this`)},
			want: &token{Kind: illegal, Pos: 5, Text: `'this`}, // the illegal character is the eof
			spos: 5,
		},
		{name: "after illegal",
			s:    &scanner{buf: []byte(`'this`), pos: 5},
			want: &token{Kind: eof, Pos: 5},
			spos: 5,
		},
		{name: "unicode",
			s:    &scanner{buf: []byte(` "\U0001F389uh huh"`), pos: 1},
			spos: 19,
			want: &token{Kind: strlit, Text: "\U0001f389uh huh", Pos: 1}, // note lowercase f doesn't matter.
		},
		{name: "eof pos",
			s:    &scanner{pos: 20},
			want: &token{Kind: eof, Pos: 20},
			spos: 20,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.s.scan()
			if diff := cmp.Diff(got, tc.want); diff != "" || tc.s.pos != tc.spos {
				t.Errorf("%v.scan() = %v want %v with pos %d", tc.s, got, tc.want, tc.spos)
			}
		})
	}
}
