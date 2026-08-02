// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto_test

import (
	"bytes"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	testpb "google.golang.org/protobuf/internal/testprotos/test"
)

func convertBenchMsg() (protoreflect.Message, protoreflect.FieldDescriptor, protoreflect.FieldDescriptor, protoreflect.FieldDescriptor, protoreflect.FieldDescriptor) {
	m := &testpb.TestAllTypes{
		OptionalString: proto.String("hello world, a reasonably sized string"),
		OptionalBytes:  bytes.Repeat([]byte{0xDE}, 32),
		OptionalInt32:  proto.Int32(42),
		RepeatedString: []string{"a", "bb", "ccc", "dddd", "eeeee", "ffffff", "ggggggg", "hhhhhhhh"},
	}
	mr := m.ProtoReflect()
	fields := mr.Descriptor().Fields()
	return mr,
		fields.ByName("optional_string"),
		fields.ByName("optional_bytes"),
		fields.ByName("optional_int32"),
		fields.ByName("repeated_string")
}

func BenchmarkConvertGetString(b *testing.B) {
	mr, fdString, _, _, _ := convertBenchMsg()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkPV = mr.Get(fdString)
	}
}

func BenchmarkConvertGetBytes(b *testing.B) {
	mr, _, fdBytes, _, _ := convertBenchMsg()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkPV = mr.Get(fdBytes)
	}
}

func BenchmarkConvertGetInt32(b *testing.B) {
	mr, _, _, fdInt32, _ := convertBenchMsg()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkPV = mr.Get(fdInt32)
	}
}

func BenchmarkConvertListGetString(b *testing.B) {
	mr, _, _, _, fdRepStr := convertBenchMsg()
	list := mr.Get(fdRepStr).List()
	n := list.Len()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j := range n {
			sinkPV = list.Get(j)
		}
	}
}

var sinkPV protoreflect.Value
