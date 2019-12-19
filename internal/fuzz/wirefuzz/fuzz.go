// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package wirefuzz includes a fuzzer for the wire marshaler and unmarshaler.
package wirefuzz

import (
	"google.golang.org/protobuf/proto"

	fuzzpb "google.golang.org/protobuf/internal/testprotos/fuzz"
)

// Fuzz is a fuzzer for proto.Marshal and proto.Unmarshal.
func Fuzz(data []byte) (score int) {
	m1 := &fuzzpb.Fuzz{}
	if err := (proto.UnmarshalOptions{
		AllowPartial: true,
	}).Unmarshal(data, m1); err != nil {
		return 0
	}
	data1, err := proto.MarshalOptions{
		AllowPartial: true,
	}.Marshal(m1)
	if err != nil {
		panic(err)
	}
	if proto.Size(m1) != len(data1) {
		panic("size does not match output")
	}
	m2 := &fuzzpb.Fuzz{}
	if err := (proto.UnmarshalOptions{
		AllowPartial: true,
	}).Unmarshal(data1, m2); err != nil {
		panic(err)
	}
	if !proto.Equal(m1, m2) {
		panic("not equal")
	}
	return 1
}
