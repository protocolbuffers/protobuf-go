// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Go native fuzzing was added in go1.18. Remove this once we stop supporting
// go1.17.
//go:build go1.18

package proto_test

import (
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	testfuzzpb "google.golang.org/protobuf/internal/testprotos/editionsfuzztest"
)

func TestUnmarshalInvalidGroupField(t *testing.T) {
	in := []byte("\x82\x01\x010")
	// Test proto2 proto
	proto2Proto := &testfuzzpb.TestAllTypesProto2{}

	if err := proto.Unmarshal(in, proto2Proto); err != nil {
		t.Error(err)
	}
	// Test equivalent editions proto
	editionsProto := &testfuzzpb.TestAllTypesProto2Editions{}

	if err := proto.Unmarshal(in, editionsProto); err != nil {
		t.Error(err)
	}
}

func FuzzProto2EditionConversion(f *testing.F) {
	f.Add([]byte("Hello World!"))
	f.Fuzz(func(t *testing.T, in []byte) {
		editionsProto := &testfuzzpb.TestAllTypesProto2Editions{}
		errEditions := proto.Unmarshal(in, editionsProto)
		proto2Proto := &testfuzzpb.TestAllTypesProto2{}
		errProto2 := proto.Unmarshal(in, proto2Proto)

		// Check that the error are the same (possible nil)
		errorsMatch := (errEditions != nil) == (errProto2 != nil)
		if errEditions != nil && errProto2 != nil {
			errorsMatch = errEditions.Error() == errProto2.Error()
		}
		if !errorsMatch {
			t.Fatalf("errors not equal:\neditions error: %v\nproto2 error:%v", errEditions, errProto2)
		}

		// Marshal the editions proto and unmarshal it into the equivalent proto2
		// message to be able to compare the messages.
		// This tests slightly more than necessary but should only lead to more
		// coverage (unless the marshalling would undo errors of the unmarshalling
		// which is very unlikely).
		marshalledEditionsProto, err := proto.Marshal(editionsProto)
		if err != nil {
			t.Fatalf("failed to marshal unmarshaled editions proto: %v", err)
		}
		roundTrippedProto2Proto := &testfuzzpb.TestAllTypesProto2{}
		err = proto.Unmarshal(marshalledEditionsProto, roundTrippedProto2Proto)
		if err != nil {
			t.Fatalf("failed to unmarshal marshaled editions proto into proto2 proto: %v", err)
		}

		// The cmp package does not deal with NaN on its own and will report
		// NaN != NaN.
		optNaN64 := cmp.Comparer(func(x, y float32) bool {
			return (math.IsNaN(float64(x)) && math.IsNaN(float64(y))) || x == y
		})
		optNaN32 := cmp.Comparer(func(x, y float64) bool {
			return (math.IsNaN(x) && math.IsNaN(y)) || x == y
		})
		if diff := cmp.Diff(proto2Proto, roundTrippedProto2Proto, protocmp.Transform(), optNaN64, optNaN32); diff != "" {
			t.Error(diff)
		}
	})
}
