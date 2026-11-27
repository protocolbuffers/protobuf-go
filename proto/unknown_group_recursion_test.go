// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto_test

import (
	"bytes"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

// nestedUnknownGroups returns depth nested empty groups on field 1:
// SGROUP(1) x depth, EGROUP(1) x depth.
func nestedUnknownGroups(depth int) []byte {
	return append(bytes.Repeat([]byte{0x0B}, depth), bytes.Repeat([]byte{0x0C}, depth)...)
}

func TestRecursionLimitUnknownGroups(t *testing.T) {
	opts := proto.UnmarshalOptions{RecursionLimit: 100}
	for _, depth := range []int{101, 1000} {
		if err := opts.Unmarshal(nestedUnknownGroups(depth), &emptypb.Empty{}); err == nil {
			t.Errorf("unknown-group depth %d with RecursionLimit=100: got nil error, want recursion-depth error", depth)
		}
	}
	if err := opts.Unmarshal(nestedUnknownGroups(1), &emptypb.Empty{}); err != nil {
		t.Errorf("unknown-group depth 1: %v", err)
	}
}
