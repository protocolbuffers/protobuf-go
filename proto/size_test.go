// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto_test

import (
	"testing"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/encoding/protowire"
	testopenpb "google.golang.org/protobuf/internal/testprotos/testeditions"
	testopaquepb "google.golang.org/protobuf/internal/testprotos/testeditions/testeditions_opaque"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
)

// Checking if [Size] returns 0 is an easy way to recognize empty messages:
func ExampleSize() {
	var m proto.Message
	if proto.Size(m) == 0 {
		// No fields set (or, in proto3, all fields matching the default);
		// skip processing this message, or return an error, or similar.
	}
}

func TestSizeAnyNonNilButEmpty(t *testing.T) {
	ne := &anypb.Any{
		TypeUrl: "abc",
		Value:   []byte{},
	}

	want := protowire.SizeBytes(len("abc")) + protowire.SizeTag(1 /* TypeUrl */)
	if got := proto.Size(ne); got != want {
		t.Errorf("proto.Size(%v) = %v, want %v", prototext.Format(ne), got, want)
	}
}

func nonZeroScalarValue(fd protoreflect.FieldDescriptor) protoreflect.Value {
	switch fd.Kind() {
	case protoreflect.BoolKind:
		return protoreflect.ValueOfBool(true)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return protoreflect.ValueOfInt32(42)
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return protoreflect.ValueOfInt64(42)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return protoreflect.ValueOfUint32(42)
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return protoreflect.ValueOfUint64(42)
	case protoreflect.FloatKind:
		return protoreflect.ValueOfFloat32(3.5)
	case protoreflect.DoubleKind:
		return protoreflect.ValueOfFloat64(3.5)
	case protoreflect.StringKind:
		return protoreflect.ValueOfString("value")
	case protoreflect.BytesKind:
		return protoreflect.ValueOfBytes([]byte("value"))
	case protoreflect.EnumKind:
		return protoreflect.ValueOfEnum(1)
	default:
		panic("unsupported scalar kind: " + fd.Kind().String())
	}
}

func populateSparseScalars(m protoreflect.Message, stride int) {
	if stride <= 0 {
		return
	}
	fields := m.Descriptor().Fields()
	n := 0
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		if !fd.HasPresence() || fd.ContainingOneof() != nil || fd.Message() != nil {
			continue
		}
		if n%stride == stride-1 {
			m.Set(fd, nonZeroScalarValue(fd))
		}
		n++
	}
}

func sparseMessages(stride int) (open, opaque proto.Message) {
	openMsg := &testopenpb.TestAllTypes{}
	opaqueMsg := &testopaquepb.TestAllTypes{}
	populateSparseScalars(openMsg.ProtoReflect(), stride)
	populateSparseScalars(opaqueMsg.ProtoReflect(), stride)
	return openMsg, opaqueMsg
}

func populateCompositeAndLazyFields(m protoreflect.Message) {
	fields := m.Descriptor().Fields()

	m.Set(fields.ByName("oneof_string"), protoreflect.ValueOfString("chosen_oneof"))
	m.Set(fields.ByName("oneof_optional_uint32"), protoreflect.ValueOfUint32(99))

	repInt32 := m.Mutable(fields.ByName("repeated_int32")).List()
	repInt32.Append(protoreflect.ValueOfInt32(10))
	repInt32.Append(protoreflect.ValueOfInt32(20))

	repNested := m.Mutable(fields.ByName("repeated_nested_message")).List()
	elem := repNested.NewElement()
	elem.Message().Set(elem.Message().Descriptor().Fields().ByName("a"), protoreflect.ValueOfInt32(7))
	repNested.Append(elem)

	strMap := m.Mutable(fields.ByName("map_string_string")).Map()
	strMap.Set(protoreflect.ValueOfString("k").MapKey(), protoreflect.ValueOfString("v"))

	nested := m.Mutable(fields.ByName("optional_nested_message")).Message()
	nested.Set(nested.Descriptor().Fields().ByName("a"), protoreflect.ValueOfInt32(11))

	lazyNested := m.Mutable(fields.ByName("optional_lazy_nested_message")).Message()
	lazyNested.Set(lazyNested.Descriptor().Fields().ByName("a"), protoreflect.ValueOfInt32(22))
}

func TestSizeSparse(t *testing.T) {
	tests := []struct {
		name      string
		stride    int
		customize func(m protoreflect.Message)
		wantEmpty bool
	}{
		{
			name:      "empty",
			stride:    0,
			wantEmpty: true,
		},
		{
			name:   "sparse",
			stride: 32,
		},
		{
			name:   "medium",
			stride: 8,
		},
		{
			name:   "dense",
			stride: 1,
		},
		{
			name:   "composite_oneof_repeated_map_nested_with_sparse_presence",
			stride: 32,
			customize: func(m protoreflect.Message) {
				populateCompositeAndLazyFields(m)
			},
		},
		{
			name:      "set_then_cleared_returns_to_zero_word",
			stride:    32,
			wantEmpty: true,
			customize: func(m protoreflect.Message) {
				fields := m.Descriptor().Fields()
				m.Clear(fields.ByName("default_string"))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			open, opaque := sparseMessages(tc.stride)
			if tc.customize != nil {
				tc.customize(open.ProtoReflect())
				tc.customize(opaque.ProtoReflect())
			}

			wireOpen, err := proto.Marshal(open)
			if err != nil {
				t.Fatalf("proto.Marshal(open) unexpected error: %v", err)
			}
			wireOpaque, err := proto.Marshal(opaque)
			if err != nil {
				t.Fatalf("proto.Marshal(opaque) unexpected error: %v", err)
			}
			if len(wireOpaque) != len(wireOpen) {
				t.Fatalf("len(wireOpaque) = %d, want len(wireOpen) = %d", len(wireOpaque), len(wireOpen))
			}
			if tc.wantEmpty && len(wireOpaque) != 0 {
				t.Fatalf("len(wireOpaque) = %d, want 0", len(wireOpaque))
			}
			if !tc.wantEmpty && len(wireOpaque) == 0 {
				t.Fatalf("len(wireOpaque) = 0, want > 0")
			}

			gotOpen := proto.Size(open)
			gotOpaque := proto.Size(opaque)
			if gotOpen != len(wireOpen) {
				t.Errorf("proto.Size(open) = %d, want %d", gotOpen, len(wireOpen))
			}
			if gotOpaque != gotOpen {
				t.Errorf("proto.Size(opaque) = %d, want %d", gotOpaque, gotOpen)
			}

			// Verify sizing after unmarshaling (exercises unexpanded lazy fields under --test_lazy_unmarshal).
			roundtripLazy := &testopaquepb.TestAllTypes{}
			if err := proto.Unmarshal(wireOpaque, roundtripLazy); err != nil {
				t.Fatalf("proto.Unmarshal(roundtripLazy) unexpected error: %v", err)
			}
			if got := proto.Size(roundtripLazy); got != len(wireOpaque) {
				t.Errorf("proto.Size(roundtripLazy) = %d, want %d", got, len(wireOpaque))
			}

			roundtripDet := &testopaquepb.TestAllTypes{}
			if err := proto.Unmarshal(wireOpaque, roundtripDet); err != nil {
				t.Fatalf("proto.Unmarshal(roundtripDet) unexpected error: %v", err)
			}
			if got := (proto.MarshalOptions{Deterministic: true}).Size(roundtripDet); got != len(wireOpaque) {
				t.Errorf("MarshalOptions{Deterministic: true}.Size(roundtripDet) = %d, want %d", got, len(wireOpaque))
			}
		})
	}
}
