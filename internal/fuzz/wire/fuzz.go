// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package wire includes a fuzzer for the wire marshaler and unmarshaler.
package wire

import (
	"google.golang.org/protobuf/proto"

	testpb "google.golang.org/protobuf/internal/testprotos/test"
	test3pb "google.golang.org/protobuf/internal/testprotos/test3"
)

// Fuzz is a fuzzer for proto.Marshal and proto.Unmarshal.
func Fuzz(data []byte) int {
	score := 0
	for _, newf := range []func() proto.Message{
		func() proto.Message { return &testpb.TestAllTypes{} },
		func() proto.Message { return &testpb.TestAllExtensions{} },
		func() proto.Message { return &test3pb.TestAllTypes{} },
	} {
		m1 := newf()
		if err := proto.Unmarshal(data, m1); err != nil {
			continue
		}
		score = 1
		data1, err := proto.Marshal(m1)
		if err != nil {
			panic(err)
		}
		if proto.Size(m1) != len(data1) {
			panic("size does not match output")
		}
		m2 := newf()
		if err := proto.Unmarshal(data1, m2); err != nil {
			panic(err)
		}
		if !proto.Equal(m1, m2) {
			panic("not equal")
		}
	}
	return score
}
