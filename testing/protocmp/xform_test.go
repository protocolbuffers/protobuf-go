// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protocmp

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"google.golang.org/protobuf/internal/detrand"
	"google.golang.org/protobuf/internal/encoding/pack"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	testpb "google.golang.org/protobuf/internal/testprotos/test"
)

func init() {
	detrand.Disable()
}

func TestTransform(t *testing.T) {
	tests := []struct {
		in         proto.Message
		want       Message
		wantString string
	}{{
		in: &testpb.TestAllTypes{
			OptionalBool:          proto.Bool(false),
			OptionalInt32:         proto.Int32(-32),
			OptionalInt64:         proto.Int64(-64),
			OptionalUint32:        proto.Uint32(32),
			OptionalUint64:        proto.Uint64(64),
			OptionalFloat:         proto.Float32(32.32),
			OptionalDouble:        proto.Float64(64.64),
			OptionalString:        proto.String("string"),
			OptionalBytes:         []byte("bytes"),
			OptionalNestedEnum:    testpb.TestAllTypes_NEG.Enum(),
			OptionalNestedMessage: &testpb.TestAllTypes_NestedMessage{A: proto.Int32(5)},
		},
		want: Message{
			messageTypeKey:            messageTypeOf(&testpb.TestAllTypes{}),
			"optional_bool":           bool(false),
			"optional_int32":          int32(-32),
			"optional_int64":          int64(-64),
			"optional_uint32":         uint32(32),
			"optional_uint64":         uint64(64),
			"optional_float":          float32(32.32),
			"optional_double":         float64(64.64),
			"optional_string":         string("string"),
			"optional_bytes":          []byte("bytes"),
			"optional_nested_enum":    enumOf(testpb.TestAllTypes_NEG),
			"optional_nested_message": Message{messageTypeKey: messageTypeOf(&testpb.TestAllTypes_NestedMessage{}), "a": int32(5)},
		},
		wantString: `{optional_int32:-32, optional_int64:-64, optional_uint32:32, optional_uint64:64, optional_float:32.32, optional_double:64.64, optional_bool:false, optional_string:"string", optional_bytes:"bytes", optional_nested_message:{a:5}, optional_nested_enum:NEG}`,
	}, {
		in: &testpb.TestAllTypes{
			RepeatedBool:   []bool{false, true},
			RepeatedInt32:  []int32{32, -32},
			RepeatedInt64:  []int64{64, -64},
			RepeatedUint32: []uint32{0, 32},
			RepeatedUint64: []uint64{0, 64},
			RepeatedFloat:  []float32{0, 32.32},
			RepeatedDouble: []float64{0, 64.64},
			RepeatedString: []string{"s1", "s2"},
			RepeatedBytes:  [][]byte{{1}, {2}},
			RepeatedNestedEnum: []testpb.TestAllTypes_NestedEnum{
				testpb.TestAllTypes_FOO,
				testpb.TestAllTypes_BAR,
			},
			RepeatedNestedMessage: []*testpb.TestAllTypes_NestedMessage{
				{A: proto.Int32(5)},
				{A: proto.Int32(-5)},
			},
		},
		want: Message{
			messageTypeKey:    messageTypeOf(&testpb.TestAllTypes{}),
			"repeated_bool":   []bool{false, true},
			"repeated_int32":  []int32{32, -32},
			"repeated_int64":  []int64{64, -64},
			"repeated_uint32": []uint32{0, 32},
			"repeated_uint64": []uint64{0, 64},
			"repeated_float":  []float32{0, 32.32},
			"repeated_double": []float64{0, 64.64},
			"repeated_string": []string{"s1", "s2"},
			"repeated_bytes":  [][]byte{{1}, {2}},
			"repeated_nested_enum": []Enum{
				enumOf(testpb.TestAllTypes_FOO),
				enumOf(testpb.TestAllTypes_BAR),
			},
			"repeated_nested_message": []Message{
				{messageTypeKey: messageTypeOf(&testpb.TestAllTypes_NestedMessage{}), "a": int32(5)},
				{messageTypeKey: messageTypeOf(&testpb.TestAllTypes_NestedMessage{}), "a": int32(-5)},
			},
		},
		wantString: `{repeated_int32:[32, -32], repeated_int64:[64, -64], repeated_uint32:[0, 32], repeated_uint64:[0, 64], repeated_float:[0, 32.32], repeated_double:[0, 64.64], repeated_bool:[false, true], repeated_string:["s1", "s2"], repeated_bytes:["\x01", "\x02"], repeated_nested_message:[{a:5}, {a:-5}], repeated_nested_enum:[FOO, BAR]}`,
	}, {
		in: &testpb.TestAllTypes{
			MapBoolBool:     map[bool]bool{true: false},
			MapInt32Int32:   map[int32]int32{-32: 32},
			MapInt64Int64:   map[int64]int64{-64: 64},
			MapUint32Uint32: map[uint32]uint32{0: 32},
			MapUint64Uint64: map[uint64]uint64{0: 64},
			MapInt32Float:   map[int32]float32{32: 32.32},
			MapInt32Double:  map[int32]float64{64: 64.64},
			MapStringString: map[string]string{"k": "v"},
			MapStringBytes:  map[string][]byte{"k": []byte("v")},
			MapStringNestedEnum: map[string]testpb.TestAllTypes_NestedEnum{
				"k": testpb.TestAllTypes_FOO,
			},
			MapStringNestedMessage: map[string]*testpb.TestAllTypes_NestedMessage{
				"k": {A: proto.Int32(5)},
			},
		},
		want: Message{
			messageTypeKey:      messageTypeOf(&testpb.TestAllTypes{}),
			"map_bool_bool":     map[bool]bool{true: false},
			"map_int32_int32":   map[int32]int32{-32: 32},
			"map_int64_int64":   map[int64]int64{-64: 64},
			"map_uint32_uint32": map[uint32]uint32{0: 32},
			"map_uint64_uint64": map[uint64]uint64{0: 64},
			"map_int32_float":   map[int32]float32{32: 32.32},
			"map_int32_double":  map[int32]float64{64: 64.64},
			"map_string_string": map[string]string{"k": "v"},
			"map_string_bytes":  map[string][]byte{"k": []byte("v")},
			"map_string_nested_enum": map[string]Enum{
				"k": enumOf(testpb.TestAllTypes_FOO),
			},
			"map_string_nested_message": map[string]Message{
				"k": {messageTypeKey: messageTypeOf(&testpb.TestAllTypes_NestedMessage{}), "a": int32(5)},
			},
		},
		wantString: `{map_int32_int32:{-32:32}, map_int64_int64:{-64:64}, map_uint32_uint32:{0:32}, map_uint64_uint64:{0:64}, map_int32_float:{32:32.32}, map_int32_double:{64:64.64}, map_bool_bool:{true:false}, map_string_string:{"k":"v"}, map_string_bytes:{"k":"v"}, map_string_nested_message:{"k":{a:5}}, map_string_nested_enum:{"k":FOO}}`,
	}, {
		in: func() proto.Message {
			m := &testpb.TestAllExtensions{}
			proto.SetExtension(m, testpb.E_OptionalBool, bool(false))
			proto.SetExtension(m, testpb.E_OptionalInt32, int32(-32))
			proto.SetExtension(m, testpb.E_OptionalInt64, int64(-64))
			proto.SetExtension(m, testpb.E_OptionalUint32, uint32(32))
			proto.SetExtension(m, testpb.E_OptionalUint64, uint64(64))
			proto.SetExtension(m, testpb.E_OptionalFloat, float32(32.32))
			proto.SetExtension(m, testpb.E_OptionalDouble, float64(64.64))
			proto.SetExtension(m, testpb.E_OptionalString, string("string"))
			proto.SetExtension(m, testpb.E_OptionalBytes, []byte("bytes"))
			proto.SetExtension(m, testpb.E_OptionalNestedEnum, testpb.TestAllTypes_NEG)
			proto.SetExtension(m, testpb.E_OptionalNestedMessage, &testpb.TestAllExtensions_NestedMessage{A: proto.Int32(5)})
			return m
		}(),
		want: Message{
			messageTypeKey:                                 messageTypeOf(&testpb.TestAllExtensions{}),
			"[goproto.proto.test.optional_bool]":           bool(false),
			"[goproto.proto.test.optional_int32]":          int32(-32),
			"[goproto.proto.test.optional_int64]":          int64(-64),
			"[goproto.proto.test.optional_uint32]":         uint32(32),
			"[goproto.proto.test.optional_uint64]":         uint64(64),
			"[goproto.proto.test.optional_float]":          float32(32.32),
			"[goproto.proto.test.optional_double]":         float64(64.64),
			"[goproto.proto.test.optional_string]":         string("string"),
			"[goproto.proto.test.optional_bytes]":          []byte("bytes"),
			"[goproto.proto.test.optional_nested_enum]":    enumOf(testpb.TestAllTypes_NEG),
			"[goproto.proto.test.optional_nested_message]": Message{messageTypeKey: messageTypeOf(&testpb.TestAllExtensions_NestedMessage{}), "a": int32(5)},
		},
		wantString: `{[goproto.proto.test.optional_bool]:false, [goproto.proto.test.optional_bytes]:"bytes", [goproto.proto.test.optional_double]:64.64, [goproto.proto.test.optional_float]:32.32, [goproto.proto.test.optional_int32]:-32, [goproto.proto.test.optional_int64]:-64, [goproto.proto.test.optional_nested_enum]:NEG, [goproto.proto.test.optional_nested_message]:{a:5}, [goproto.proto.test.optional_string]:"string", [goproto.proto.test.optional_uint32]:32, [goproto.proto.test.optional_uint64]:64}`,
	}, {
		in: func() proto.Message {
			m := &testpb.TestAllExtensions{}
			proto.SetExtension(m, testpb.E_RepeatedBool, []bool{false, true})
			proto.SetExtension(m, testpb.E_RepeatedInt32, []int32{32, -32})
			proto.SetExtension(m, testpb.E_RepeatedInt64, []int64{64, -64})
			proto.SetExtension(m, testpb.E_RepeatedUint32, []uint32{0, 32})
			proto.SetExtension(m, testpb.E_RepeatedUint64, []uint64{0, 64})
			proto.SetExtension(m, testpb.E_RepeatedFloat, []float32{0, 32.32})
			proto.SetExtension(m, testpb.E_RepeatedDouble, []float64{0, 64.64})
			proto.SetExtension(m, testpb.E_RepeatedString, []string{"s1", "s2"})
			proto.SetExtension(m, testpb.E_RepeatedBytes, [][]byte{{1}, {2}})
			proto.SetExtension(m, testpb.E_RepeatedNestedEnum, []testpb.TestAllTypes_NestedEnum{
				testpb.TestAllTypes_FOO,
				testpb.TestAllTypes_BAR,
			})
			proto.SetExtension(m, testpb.E_RepeatedNestedMessage, []*testpb.TestAllExtensions_NestedMessage{
				{A: proto.Int32(5)},
				{A: proto.Int32(-5)},
			})
			return m
		}(),
		want: Message{
			messageTypeKey:                         messageTypeOf(&testpb.TestAllExtensions{}),
			"[goproto.proto.test.repeated_bool]":   []bool{false, true},
			"[goproto.proto.test.repeated_int32]":  []int32{32, -32},
			"[goproto.proto.test.repeated_int64]":  []int64{64, -64},
			"[goproto.proto.test.repeated_uint32]": []uint32{0, 32},
			"[goproto.proto.test.repeated_uint64]": []uint64{0, 64},
			"[goproto.proto.test.repeated_float]":  []float32{0, 32.32},
			"[goproto.proto.test.repeated_double]": []float64{0, 64.64},
			"[goproto.proto.test.repeated_string]": []string{"s1", "s2"},
			"[goproto.proto.test.repeated_bytes]":  [][]byte{{1}, {2}},
			"[goproto.proto.test.repeated_nested_enum]": []Enum{
				enumOf(testpb.TestAllTypes_FOO),
				enumOf(testpb.TestAllTypes_BAR),
			},
			"[goproto.proto.test.repeated_nested_message]": []Message{
				{messageTypeKey: messageTypeOf(&testpb.TestAllExtensions_NestedMessage{}), "a": int32(5)},
				{messageTypeKey: messageTypeOf(&testpb.TestAllExtensions_NestedMessage{}), "a": int32(-5)},
			},
		},
		wantString: `{[goproto.proto.test.repeated_bool]:[false, true], [goproto.proto.test.repeated_bytes]:["\x01", "\x02"], [goproto.proto.test.repeated_double]:[0, 64.64], [goproto.proto.test.repeated_float]:[0, 32.32], [goproto.proto.test.repeated_int32]:[32, -32], [goproto.proto.test.repeated_int64]:[64, -64], [goproto.proto.test.repeated_nested_enum]:[FOO, BAR], [goproto.proto.test.repeated_nested_message]:[{a:5}, {a:-5}], [goproto.proto.test.repeated_string]:["s1", "s2"], [goproto.proto.test.repeated_uint32]:[0, 32], [goproto.proto.test.repeated_uint64]:[0, 64]}`,
	}, {
		in: func() proto.Message {
			m := &testpb.TestAllTypes{}
			m.ProtoReflect().SetUnknown(pack.Message{
				pack.Tag{Number: 50000, Type: pack.VarintType}, pack.Uvarint(100),
				pack.Tag{Number: 50001, Type: pack.Fixed32Type}, pack.Uint32(200),
				pack.Tag{Number: 50002, Type: pack.Fixed64Type}, pack.Uint64(300),
				pack.Tag{Number: 50003, Type: pack.BytesType}, pack.String("hello"),
				pack.Message{
					pack.Tag{Number: 50004, Type: pack.StartGroupType},
					pack.Tag{Number: 1, Type: pack.VarintType}, pack.Uvarint(100),
					pack.Tag{Number: 1, Type: pack.Fixed32Type}, pack.Uint32(200),
					pack.Tag{Number: 1, Type: pack.Fixed64Type}, pack.Uint64(300),
					pack.Tag{Number: 1, Type: pack.BytesType}, pack.String("hello"),
					pack.Message{
						pack.Tag{Number: 1, Type: pack.StartGroupType},
						pack.Tag{Number: 1, Type: pack.VarintType}, pack.Uvarint(100),
						pack.Tag{Number: 1, Type: pack.Fixed32Type}, pack.Uint32(200),
						pack.Tag{Number: 1, Type: pack.Fixed64Type}, pack.Uint64(300),
						pack.Tag{Number: 1, Type: pack.BytesType}, pack.String("hello"),
						pack.Tag{Number: 1, Type: pack.EndGroupType},
					},
					pack.Tag{Number: 50004, Type: pack.EndGroupType},
				},
			}.Marshal())
			return m
		}(),
		want: Message{
			messageTypeKey: messageTypeOf(&testpb.TestAllTypes{}),
			"50000":        protoreflect.RawFields(pack.Message{pack.Tag{Number: 50000, Type: pack.VarintType}, pack.Uvarint(100)}.Marshal()),
			"50001":        protoreflect.RawFields(pack.Message{pack.Tag{Number: 50001, Type: pack.Fixed32Type}, pack.Uint32(200)}.Marshal()),
			"50002":        protoreflect.RawFields(pack.Message{pack.Tag{Number: 50002, Type: pack.Fixed64Type}, pack.Uint64(300)}.Marshal()),
			"50003":        protoreflect.RawFields(pack.Message{pack.Tag{Number: 50003, Type: pack.BytesType}, pack.String("hello")}.Marshal()),
			"50004": protoreflect.RawFields(pack.Message{
				pack.Tag{Number: 50004, Type: pack.StartGroupType},
				pack.Tag{Number: 1, Type: pack.VarintType}, pack.Uvarint(100),
				pack.Tag{Number: 1, Type: pack.Fixed32Type}, pack.Uint32(200),
				pack.Tag{Number: 1, Type: pack.Fixed64Type}, pack.Uint64(300),
				pack.Tag{Number: 1, Type: pack.BytesType}, pack.String("hello"),
				pack.Message{
					pack.Tag{Number: 1, Type: pack.StartGroupType},
					pack.Tag{Number: 1, Type: pack.VarintType}, pack.Uvarint(100),
					pack.Tag{Number: 1, Type: pack.Fixed32Type}, pack.Uint32(200),
					pack.Tag{Number: 1, Type: pack.Fixed64Type}, pack.Uint64(300),
					pack.Tag{Number: 1, Type: pack.BytesType}, pack.String("hello"),
					pack.Tag{Number: 1, Type: pack.EndGroupType},
				},
				pack.Tag{Number: 50004, Type: pack.EndGroupType},
			}.Marshal()),
		},
		wantString: `{50000:100, 50001:200, 50002:300, 50003:"hello", 50004:{1:[100, 200, 300, "hello", {1:[100, 200, 300, "hello"]}]}}`,
	}}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := transformMessage(tt.in.ProtoReflect())
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Transform() mismatch (-want +got):\n%v", diff)
			}
			if diff := cmp.Diff(tt.wantString, got.String()); diff != "" {
				t.Errorf("Transform().String() mismatch (-want +got):\n%v", diff)
			}
		})
	}
}

func enumOf(e protoreflect.Enum) Enum {
	return Enum{e.Number(), e.Descriptor()}
}

func messageTypeOf(m protoreflect.ProtoMessage) messageType {
	return messageType{md: m.ProtoReflect().Descriptor()}
}
