// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl_test

import (
	"fmt"
	"testing"

	cmp "github.com/google/go-cmp/cmp"
	testpb "google.golang.org/protobuf/internal/testprotos/test"
	"google.golang.org/protobuf/proto"
	pref "google.golang.org/protobuf/reflect/protoreflect"
)

func TestExtensionType(t *testing.T) {
	cmpOpts := cmp.Options{
		cmp.Comparer(func(x, y proto.Message) bool {
			return proto.Equal(x, y)
		}),
	}
	for _, test := range []struct {
		xt    pref.ExtensionType
		value interface{}
	}{
		{
			xt:    testpb.E_OptionalInt32Extension,
			value: int32(0),
		},
		{
			xt:    testpb.E_OptionalInt64Extension,
			value: int64(0),
		},
		{
			xt:    testpb.E_OptionalUint32Extension,
			value: uint32(0),
		},
		{
			xt:    testpb.E_OptionalUint64Extension,
			value: uint64(0),
		},
		{
			xt:    testpb.E_OptionalFloatExtension,
			value: float32(0),
		},
		{
			xt:    testpb.E_OptionalDoubleExtension,
			value: float64(0),
		},
		{
			xt:    testpb.E_OptionalBoolExtension,
			value: true,
		},
		{
			xt:    testpb.E_OptionalStringExtension,
			value: "",
		},
		{
			xt:    testpb.E_OptionalBytesExtension,
			value: []byte{},
		},
		{
			xt:    testpb.E_OptionalNestedMessageExtension,
			value: &testpb.TestAllTypes_NestedMessage{},
		},
		{
			xt:    testpb.E_OptionalNestedEnumExtension,
			value: testpb.TestAllTypes_FOO,
		},
		{
			xt:    testpb.E_RepeatedInt32Extension,
			value: []int32{0},
		},
		{
			xt:    testpb.E_RepeatedInt64Extension,
			value: []int64{0},
		},
		{
			xt:    testpb.E_RepeatedUint32Extension,
			value: []uint32{0},
		},
		{
			xt:    testpb.E_RepeatedUint64Extension,
			value: []uint64{0},
		},
		{
			xt:    testpb.E_RepeatedFloatExtension,
			value: []float32{0},
		},
		{
			xt:    testpb.E_RepeatedDoubleExtension,
			value: []float64{0},
		},
		{
			xt:    testpb.E_RepeatedBoolExtension,
			value: []bool{true},
		},
		{
			xt:    testpb.E_RepeatedStringExtension,
			value: []string{""},
		},
		{
			xt:    testpb.E_RepeatedBytesExtension,
			value: [][]byte{nil},
		},
		{
			xt:    testpb.E_RepeatedNestedMessageExtension,
			value: []*testpb.TestAllTypes_NestedMessage{{}},
		},
		{
			xt:    testpb.E_RepeatedNestedEnumExtension,
			value: []testpb.TestAllTypes_NestedEnum{testpb.TestAllTypes_FOO},
		},
	} {
		name := test.xt.TypeDescriptor().FullName()
		t.Run(fmt.Sprint(name), func(t *testing.T) {
			if !test.xt.IsValidInterface(test.value) {
				t.Fatalf("IsValidInterface(%[1]T(%[1]v)) = false, want true", test.value)
			}
			v := test.xt.ValueOf(test.value)
			if !test.xt.IsValidValue(v) {
				t.Fatalf("IsValidValue(%[1]T(%[1]v)) = false, want true", v)
			}
			if got, want := test.xt.InterfaceOf(v), test.value; !cmp.Equal(got, want, cmpOpts) {
				t.Fatalf("round trip InterfaceOf(ValueOf(x)) = %v, want %v", got, want)
			}
		})
	}
}
