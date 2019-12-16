// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package wirefuzz includes a fuzzer for the wire marshaler and unmarshaler.
package wirefuzz

import (
	"fmt"

	"google.golang.org/protobuf/internal/impl"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"
	piface "google.golang.org/protobuf/runtime/protoiface"

	fuzzpb "google.golang.org/protobuf/internal/testprotos/fuzz"
)

// Fuzz is a fuzzer for proto.Marshal and proto.Unmarshal.
func Fuzz(data []byte) (score int) {
	m1 := &fuzzpb.Fuzz{}
	valid := impl.Validate(data, m1.ProtoReflect().Type(), piface.UnmarshalOptions{
		Resolver: protoregistry.GlobalTypes,
	})
	if err := (proto.UnmarshalOptions{
		AllowPartial: true,
	}).Unmarshal(data, m1); err != nil {
		switch valid {
		case impl.ValidationUnknown:
		case impl.ValidationInvalid:
		default:
			panic("unmarshal error with validation status: " + valid.String())
		}
		return 0
	}
	if proto.IsInitialized(m1) == nil {
		switch valid {
		case impl.ValidationUnknown:
		case impl.ValidationValidInitialized:
		case impl.ValidationValidMaybeUninitalized:
		default:
			panic("unmarshal ok with validation status: " + valid.String())
		}
	} else {
		switch valid {
		case impl.ValidationUnknown:
		case impl.ValidationValidMaybeUninitalized:
		default:
			panic("partial unmarshal ok with validation status: " + valid.String())
		}
	}
	data1, err := proto.MarshalOptions{
		AllowPartial: true,
	}.Marshal(m1)
	if err != nil {
		panic(err)
	}
	if proto.Size(m1) != len(data1) {
		panic(fmt.Errorf("size does not match output %v", m1))
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
