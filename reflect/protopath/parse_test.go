// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package protopath provides functionality for
// representing a sequence of protobuf reflection operations on a message.
package protopath

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	pb "google.golang.org/protobuf/reflect/protopath/testmessage"
	"google.golang.org/protobuf/reflect/protoreflect"
)

//go:generate protoc --go_out=. --go_opt=module=github.com/protocolbuffers/protobuf-go/reflect/protopath testmessage.proto

func matchErr(err error, want string) bool {
	return (err == nil && want == "") ||
		(want != "" && err != nil && strings.Contains(err.Error(), want))
}

func TestParsePath(t *testing.T) {
	data := &pb.Test{}
	m := data.ProtoReflect()
	md := m.Descriptor()
	tcs := []struct {
		name    string
		md      protoreflect.MessageDescriptor
		path    string
		want    string
		wantErr string
	}{
		{
			name:    "empty",
			wantErr: "message descriptor must be non-nil",
		},
		{
			name: "root from empty",
			md:   md,
			want: "(testprotopath.Test)",
		},
		{
			name: "root from fullpath",
			md:   md,
			path: "(testprotopath.Test)",
			want: "(testprotopath.Test)",
		},
		{
			name: "implicit root",
			md:   md,
			path: "nested",
			want: "(testprotopath.Test).nested",
		},
		{
			name: "explicit root",
			md:   md,
			path: "(testprotopath.Test).nested",
			want: "(testprotopath.Test).nested",
		},
		{
			name:    "unknown field",
			md:      md,
			path:    "unknown",
			wantErr: "\"unknown\" not in message descriptor",
		},
		{
			name: "list index",
			md:   md,
			path: "repeats[5]",
			want: "(testprotopath.Test).repeats[5]",
		},
		{
			name: "map string index",
			md:   md,
			path: "strkeymap[\"key\"]",
			want: "(testprotopath.Test).strkeymap[\"key\"]",
		},
		{
			name: "map bool true index",
			md:   md,
			path: "boolkeymap[true]",
			want: "(testprotopath.Test).boolkeymap[true]",
		},
		{
			name: "map bool false index",
			md:   md,
			path: "boolkeymap[false]",
			want: "(testprotopath.Test).boolkeymap[false]",
		},
		{
			name: "map subindex",
			md:   md,
			path: "strkeymap[\"key\"].stringfield",
			want: "(testprotopath.Test).strkeymap[\"key\"].stringfield",
		},
		{
			name: "pod index",
			md:   md,
			path: "int32repeats[123]",
			want: "(testprotopath.Test).int32repeats[123]",
		},
		{
			name:    "double map index",
			md:      md,
			path:    "strkeymap[\"key\"][\"key2\"]",
			wantErr: "expected field descriptor to access with value \"key2\"",
		},
		{
			name: "big index",
			md:   md,
			path: "uint64keymap[0xffffffffffffffff]",
			want: "(testprotopath.Test).uint64keymap[18446744073709551615]",
		},
		{
			name:    "too big index",
			md:      md,
			path:    "uint64keymap[0xfffffffffffffffff]",
			wantErr: "cannot index map with key kind uint64 with key 0xfffffffffffffffff",
		},
		{
			name:    "negative index is bad",
			md:      md,
			path:    "repeats[-4]",
			wantErr: "negative index -4",
		},
		{
			name:    "negative uint is bad",
			md:      md,
			path:    "uint32keymap[-4]",
			wantErr: "cannot index map with key kind uint32 with key -4",
		},
		{
			name: "uint32 index",
			md:   md,
			path: "uint32keymap[4]",
			want: "(testprotopath.Test).uint32keymap[4]",
		},
		{
			name: "negative int is fine",
			md:   md,
			path: "int32keymap[-4]",
			want: "(testprotopath.Test).int32keymap[-4]",
		},
		{
			name:    "really negative int is not fine",
			md:      md,
			path:    "int32keymap[-0xffffffff]",
			wantErr: "cannot index map with key kind int32 with key -0xffffffff",
		},
		{
			name: "really negative int is fine for 64",
			md:   md,
			path: "int64keymap[-0xffffffff]",
			want: "(testprotopath.Test).int64keymap[-4294967295]",
		},
		{
			name:    "reaaaaally negative int is bad for 64",
			md:      md,
			path:    "int64keymap[-0xffffffffffffffff]",
			wantErr: "cannot index map with key kind int64 with key -0xffffffffffffffff",
		},
		{
			name:    "string index for int map is bad",
			md:      md,
			path:    "int32keymap[\"foo\"]",
			wantErr: "cannot index map with key kind int32 with key \"foo\"",
		},
		{
			name:    "bool index for uint map is bad",
			md:      md,
			path:    "uint32keymap[true]",
			wantErr: "cannot index map with key kind uint32 with key true",
		},
		{
			name: "recursion! with octal literals!",
			md:   md,
			path: `int32keymap[-6].uint64keymap[040000000000].repeats[0].nested.nested.strkeymap["k"].intfield`,
			want: `(testprotopath.Test).int32keymap[-6].uint64keymap[4294967296].repeats[0].nested.nested.strkeymap["k"].intfield`,
		},
		{name: "unexpected string index",
			md:      md,
			path:    `int32repeats["key"]`,
			wantErr: "non-integral type \"key\"",
		},
		{name: "unexpected string field",
			md:      md,
			path:    `nested."stringfield"`,
			wantErr: "expect field name following '.'",
		},
		{name: "unexpected string access",
			md:      md,
			path:    `nested"stringfield"`,
			wantErr: "expect one of '[', '.', or eof",
		},
		{name: "root weirdness",
			md:      md,
			path:    "(a.1)",
			wantErr: "expect next name fragment of message descriptor's full name",
		},
		{name: "no root name",
			md:      md,
			path:    "()",
			wantErr: "expect next name fragment of message descriptor's full name",
		},
		{name: "root name dot",
			md:      md,
			path:    `(a"str")`,
			wantErr: "expect either '.' for next full name fragment or ')'",
		},
		{name: "no index close",
			md:      md,
			path:    `int32repeats[32)`,
			wantErr: "expect ']'",
		},
		{name: "no field",
			md:      md,
			path:    `nested.`,
			wantErr: "finished parsing in state that expects field name following '.'",
		},
		{name: "can't index int32",
			md:      md,
			path:    `int32repeats[5].nested`,
			wantErr: "column 16: int32repeats[5].nested\n                           ^----|\nexpected message descriptor to access with field \"nested\"",
		},
		{name: "can't access map",
			md:      md,
			path:    "strkeymap.nested",
			wantErr: "field \"nested\" not in message descriptor",
		},
		{name: "can't access map key field",
			md:      md,
			path:    "strkeymap.key",
			wantErr: "map internal field \"key\" may not be traversed",
		},
		{name: "can't access map value field",
			md:      md,
			path:    "strkeymap.value",
			wantErr: "map internal field \"value\" may not be traversed",
		},
		{
			name:    "huge list index",
			md:      md,
			path:    "repeats[0xfffffffffffffffff]",
			wantErr: "non-integral type 0xfffffffffffffffff", // doesn't fit in any integral type.
		},
		{
			name:    "ident index",
			md:      md,
			path:    "repeats[x]",
			wantErr: "expected value for index, not an identifier \"x\"",
		},
		{
			name:    "root[",
			md:      md,
			path:    "(testprotopath.Test)[5]",
			wantErr: "expect either '.' or eof following root",
		},
		{
			name:    "[",
			md:      md,
			path:    "[5]",
			wantErr: "expect first field or '(' for full name",
		},
		{
			name:    "dot index",
			md:      md,
			path:    "int32repeats[.5]",
			wantErr: "got '.'",
		},
		{
			name:    "bool index",
			md:      md,
			path:    "int32repeats[false]",
			wantErr: "non-integral type false", // bool is integral, but won't be cast to 0 or 1.
		},
		{
			name:    "whitespace is illegal",
			md:      md,
			path:    " ",
			wantErr: "found illegal token ' ' at position 0",
		},
		{
			name:    "unexpected identifier",
			md:      md,
			path:    "strkeymap['foo'bar]",
			wantErr: "got identifier \"bar\"",
		},
		{
			name:    "unindexable",
			md:      md,
			path:    "nested.stringfield[0]",
			wantErr: "expected field descriptor with repeated cardinality",
		},
		{
			name:    "weird (",
			md:      md,
			path:    "nested(testprotopath.Test)",
			wantErr: "got '('",
		},
		{
			name:    "bad token",
			md:      md,
			path:    "'",
			wantErr: "found illegal token ' at position 1",
		},
		{
			name:    "bad token after good",
			md:      md,
			path:    "nested🎉",
			wantErr: "found illegal token '🎉'",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParsePath(tc.md, tc.path)
			if !matchErr(err, tc.wantErr) {
				t.Fatalf("ParsePath(%q) = %v, %v errored unexpectedly. Want %q", tc.path, got, err, tc.wantErr)
			}
			if tc.wantErr != "" {
				return
			}
			if diff := cmp.Diff(tc.want, got.String()); diff != "" {
				t.Fatalf("ParsePath(%q) diff (-want +got) %s", tc.path, diff)
			}
		})
	}
}
