// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style.
// license that can be found in the LICENSE file.

package proto_test

import (
	"google.golang.org/protobuf/internal/encoding/pack"
	"google.golang.org/protobuf/internal/encoding/wire"
	"google.golang.org/protobuf/internal/impl"
	"google.golang.org/protobuf/internal/protobuild"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	legacypb "google.golang.org/protobuf/internal/testprotos/legacy"
	requiredpb "google.golang.org/protobuf/internal/testprotos/required"
	testpb "google.golang.org/protobuf/internal/testprotos/test"
	test3pb "google.golang.org/protobuf/internal/testprotos/test3"
)

type testProto struct {
	desc             string
	decodeTo         []proto.Message
	wire             []byte
	partial          bool
	noEncode         bool
	checkFastInit    bool
	unmarshalOptions proto.UnmarshalOptions
	validationStatus impl.ValidationStatus
	nocheckValidInit bool
}

func makeMessages(in protobuild.Message, messages ...proto.Message) []proto.Message {
	if len(messages) == 0 {
		messages = []proto.Message{
			&testpb.TestAllTypes{},
			&test3pb.TestAllTypes{},
			&testpb.TestAllExtensions{},
		}
	}
	for _, m := range messages {
		in.Build(m.ProtoReflect())
	}
	return messages
}

var testValidMessages = []testProto{
	{
		desc:          "basic scalar types",
		checkFastInit: true,
		decodeTo: makeMessages(protobuild.Message{
			"optional_int32":       1001,
			"optional_int64":       1002,
			"optional_uint32":      1003,
			"optional_uint64":      1004,
			"optional_sint32":      1005,
			"optional_sint64":      1006,
			"optional_fixed32":     1007,
			"optional_fixed64":     1008,
			"optional_sfixed32":    1009,
			"optional_sfixed64":    1010,
			"optional_float":       1011.5,
			"optional_double":      1012.5,
			"optional_bool":        true,
			"optional_string":      "string",
			"optional_bytes":       []byte("bytes"),
			"optional_nested_enum": "BAR",
		}),
		wire: pack.Message{
			pack.Tag{1, pack.VarintType}, pack.Varint(1001),
			pack.Tag{2, pack.VarintType}, pack.Varint(1002),
			pack.Tag{3, pack.VarintType}, pack.Uvarint(1003),
			pack.Tag{4, pack.VarintType}, pack.Uvarint(1004),
			pack.Tag{5, pack.VarintType}, pack.Svarint(1005),
			pack.Tag{6, pack.VarintType}, pack.Svarint(1006),
			pack.Tag{7, pack.Fixed32Type}, pack.Uint32(1007),
			pack.Tag{8, pack.Fixed64Type}, pack.Uint64(1008),
			pack.Tag{9, pack.Fixed32Type}, pack.Int32(1009),
			pack.Tag{10, pack.Fixed64Type}, pack.Int64(1010),
			pack.Tag{11, pack.Fixed32Type}, pack.Float32(1011.5),
			pack.Tag{12, pack.Fixed64Type}, pack.Float64(1012.5),
			pack.Tag{13, pack.VarintType}, pack.Bool(true),
			pack.Tag{14, pack.BytesType}, pack.String("string"),
			pack.Tag{15, pack.BytesType}, pack.Bytes([]byte("bytes")),
			pack.Tag{21, pack.VarintType}, pack.Varint(int(testpb.TestAllTypes_BAR)),
		}.Marshal(),
	},
	{
		desc: "zero values",
		decodeTo: makeMessages(protobuild.Message{
			"optional_int32":    0,
			"optional_int64":    0,
			"optional_uint32":   0,
			"optional_uint64":   0,
			"optional_sint32":   0,
			"optional_sint64":   0,
			"optional_fixed32":  0,
			"optional_fixed64":  0,
			"optional_sfixed32": 0,
			"optional_sfixed64": 0,
			"optional_float":    0,
			"optional_double":   0,
			"optional_bool":     false,
			"optional_string":   "",
			"optional_bytes":    []byte{},
		}),
		wire: pack.Message{
			pack.Tag{1, pack.VarintType}, pack.Varint(0),
			pack.Tag{2, pack.VarintType}, pack.Varint(0),
			pack.Tag{3, pack.VarintType}, pack.Uvarint(0),
			pack.Tag{4, pack.VarintType}, pack.Uvarint(0),
			pack.Tag{5, pack.VarintType}, pack.Svarint(0),
			pack.Tag{6, pack.VarintType}, pack.Svarint(0),
			pack.Tag{7, pack.Fixed32Type}, pack.Uint32(0),
			pack.Tag{8, pack.Fixed64Type}, pack.Uint64(0),
			pack.Tag{9, pack.Fixed32Type}, pack.Int32(0),
			pack.Tag{10, pack.Fixed64Type}, pack.Int64(0),
			pack.Tag{11, pack.Fixed32Type}, pack.Float32(0),
			pack.Tag{12, pack.Fixed64Type}, pack.Float64(0),
			pack.Tag{13, pack.VarintType}, pack.Bool(false),
			pack.Tag{14, pack.BytesType}, pack.String(""),
			pack.Tag{15, pack.BytesType}, pack.Bytes(nil),
		}.Marshal(),
	},
	{
		desc: "groups",
		decodeTo: makeMessages(protobuild.Message{
			"optionalgroup": protobuild.Message{
				"a":                 1017,
				"same_field_number": 1016,
			},
		}, &testpb.TestAllTypes{}, &testpb.TestAllExtensions{}),
		wire: pack.Message{
			pack.Tag{16, pack.StartGroupType},
			pack.Tag{17, pack.VarintType}, pack.Varint(1017),
			pack.Tag{16, pack.VarintType}, pack.Varint(1016),
			pack.Tag{16, pack.EndGroupType},
		}.Marshal(),
	},
	{
		desc: "groups (field overridden)",
		decodeTo: makeMessages(protobuild.Message{
			"optionalgroup": protobuild.Message{
				"a": 2,
			},
		}, &testpb.TestAllTypes{}, &testpb.TestAllExtensions{}),
		wire: pack.Message{
			pack.Tag{16, pack.StartGroupType},
			pack.Tag{17, pack.VarintType}, pack.Varint(1),
			pack.Tag{16, pack.EndGroupType},
			pack.Tag{16, pack.StartGroupType},
			pack.Tag{17, pack.VarintType}, pack.Varint(2),
			pack.Tag{16, pack.EndGroupType},
		}.Marshal(),
	},
	{
		desc: "messages",
		decodeTo: makeMessages(protobuild.Message{
			"optional_nested_message": protobuild.Message{
				"a": 42,
				"corecursive": protobuild.Message{
					"optional_int32": 43,
				},
			},
		}),
		wire: pack.Message{
			pack.Tag{18, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(42),
				pack.Tag{2, pack.BytesType}, pack.LengthPrefix(pack.Message{
					pack.Tag{1, pack.VarintType}, pack.Varint(43),
				}),
			}),
		}.Marshal(),
	},
	{
		desc: "messages (split across multiple tags)",
		decodeTo: makeMessages(protobuild.Message{
			"optional_nested_message": protobuild.Message{
				"a": 42,
				"corecursive": protobuild.Message{
					"optional_int32": 43,
				},
			},
		}),
		wire: pack.Message{
			pack.Tag{18, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(42),
			}),
			pack.Tag{18, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{2, pack.BytesType}, pack.LengthPrefix(pack.Message{
					pack.Tag{1, pack.VarintType}, pack.Varint(43),
				}),
			}),
		}.Marshal(),
	},
	{
		desc: "messages (field overridden)",
		decodeTo: makeMessages(protobuild.Message{
			"optional_nested_message": protobuild.Message{
				"a": 2,
			},
		}),
		wire: pack.Message{
			pack.Tag{18, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(1),
			}),
			pack.Tag{18, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(2),
			}),
		}.Marshal(),
	},
	{
		desc: "basic repeated types",
		decodeTo: makeMessages(protobuild.Message{
			"repeated_int32":       []int32{1001, 2001},
			"repeated_int64":       []int64{1002, 2002},
			"repeated_uint32":      []uint32{1003, 2003},
			"repeated_uint64":      []uint64{1004, 2004},
			"repeated_sint32":      []int32{1005, 2005},
			"repeated_sint64":      []int64{1006, 2006},
			"repeated_fixed32":     []uint32{1007, 2007},
			"repeated_fixed64":     []uint64{1008, 2008},
			"repeated_sfixed32":    []int32{1009, 2009},
			"repeated_sfixed64":    []int64{1010, 2010},
			"repeated_float":       []float32{1011.5, 2011.5},
			"repeated_double":      []float64{1012.5, 2012.5},
			"repeated_bool":        []bool{true, false},
			"repeated_string":      []string{"foo", "bar"},
			"repeated_bytes":       []string{"FOO", "BAR"},
			"repeated_nested_enum": []string{"FOO", "BAR"},
		}),
		wire: pack.Message{
			pack.Tag{31, pack.VarintType}, pack.Varint(1001),
			pack.Tag{31, pack.VarintType}, pack.Varint(2001),
			pack.Tag{32, pack.VarintType}, pack.Varint(1002),
			pack.Tag{32, pack.VarintType}, pack.Varint(2002),
			pack.Tag{33, pack.VarintType}, pack.Uvarint(1003),
			pack.Tag{33, pack.VarintType}, pack.Uvarint(2003),
			pack.Tag{34, pack.VarintType}, pack.Uvarint(1004),
			pack.Tag{34, pack.VarintType}, pack.Uvarint(2004),
			pack.Tag{35, pack.VarintType}, pack.Svarint(1005),
			pack.Tag{35, pack.VarintType}, pack.Svarint(2005),
			pack.Tag{36, pack.VarintType}, pack.Svarint(1006),
			pack.Tag{36, pack.VarintType}, pack.Svarint(2006),
			pack.Tag{37, pack.Fixed32Type}, pack.Uint32(1007),
			pack.Tag{37, pack.Fixed32Type}, pack.Uint32(2007),
			pack.Tag{38, pack.Fixed64Type}, pack.Uint64(1008),
			pack.Tag{38, pack.Fixed64Type}, pack.Uint64(2008),
			pack.Tag{39, pack.Fixed32Type}, pack.Int32(1009),
			pack.Tag{39, pack.Fixed32Type}, pack.Int32(2009),
			pack.Tag{40, pack.Fixed64Type}, pack.Int64(1010),
			pack.Tag{40, pack.Fixed64Type}, pack.Int64(2010),
			pack.Tag{41, pack.Fixed32Type}, pack.Float32(1011.5),
			pack.Tag{41, pack.Fixed32Type}, pack.Float32(2011.5),
			pack.Tag{42, pack.Fixed64Type}, pack.Float64(1012.5),
			pack.Tag{42, pack.Fixed64Type}, pack.Float64(2012.5),
			pack.Tag{43, pack.VarintType}, pack.Bool(true),
			pack.Tag{43, pack.VarintType}, pack.Bool(false),
			pack.Tag{44, pack.BytesType}, pack.String("foo"),
			pack.Tag{44, pack.BytesType}, pack.String("bar"),
			pack.Tag{45, pack.BytesType}, pack.Bytes([]byte("FOO")),
			pack.Tag{45, pack.BytesType}, pack.Bytes([]byte("BAR")),
			pack.Tag{51, pack.VarintType}, pack.Varint(int(testpb.TestAllTypes_FOO)),
			pack.Tag{51, pack.VarintType}, pack.Varint(int(testpb.TestAllTypes_BAR)),
		}.Marshal(),
	},
	{
		desc: "basic repeated types (packed encoding)",
		decodeTo: makeMessages(protobuild.Message{
			"repeated_int32":       []int32{1001, 2001},
			"repeated_int64":       []int64{1002, 2002},
			"repeated_uint32":      []uint32{1003, 2003},
			"repeated_uint64":      []uint64{1004, 2004},
			"repeated_sint32":      []int32{1005, 2005},
			"repeated_sint64":      []int64{1006, 2006},
			"repeated_fixed32":     []uint32{1007, 2007},
			"repeated_fixed64":     []uint64{1008, 2008},
			"repeated_sfixed32":    []int32{1009, 2009},
			"repeated_sfixed64":    []int64{1010, 2010},
			"repeated_float":       []float32{1011.5, 2011.5},
			"repeated_double":      []float64{1012.5, 2012.5},
			"repeated_bool":        []bool{true, false},
			"repeated_nested_enum": []string{"FOO", "BAR"},
		}),
		wire: pack.Message{
			pack.Tag{31, pack.BytesType}, pack.LengthPrefix{
				pack.Varint(1001), pack.Varint(2001),
			},
			pack.Tag{32, pack.BytesType}, pack.LengthPrefix{
				pack.Varint(1002), pack.Varint(2002),
			},
			pack.Tag{33, pack.BytesType}, pack.LengthPrefix{
				pack.Uvarint(1003), pack.Uvarint(2003),
			},
			pack.Tag{34, pack.BytesType}, pack.LengthPrefix{
				pack.Uvarint(1004), pack.Uvarint(2004),
			},
			pack.Tag{35, pack.BytesType}, pack.LengthPrefix{
				pack.Svarint(1005), pack.Svarint(2005),
			},
			pack.Tag{36, pack.BytesType}, pack.LengthPrefix{
				pack.Svarint(1006), pack.Svarint(2006),
			},
			pack.Tag{37, pack.BytesType}, pack.LengthPrefix{
				pack.Uint32(1007), pack.Uint32(2007),
			},
			pack.Tag{38, pack.BytesType}, pack.LengthPrefix{
				pack.Uint64(1008), pack.Uint64(2008),
			},
			pack.Tag{39, pack.BytesType}, pack.LengthPrefix{
				pack.Int32(1009), pack.Int32(2009),
			},
			pack.Tag{40, pack.BytesType}, pack.LengthPrefix{
				pack.Int64(1010), pack.Int64(2010),
			},
			pack.Tag{41, pack.BytesType}, pack.LengthPrefix{
				pack.Float32(1011.5), pack.Float32(2011.5),
			},
			pack.Tag{42, pack.BytesType}, pack.LengthPrefix{
				pack.Float64(1012.5), pack.Float64(2012.5),
			},
			pack.Tag{43, pack.BytesType}, pack.LengthPrefix{
				pack.Bool(true), pack.Bool(false),
			},
			pack.Tag{51, pack.BytesType}, pack.LengthPrefix{
				pack.Varint(int(testpb.TestAllTypes_FOO)),
				pack.Varint(int(testpb.TestAllTypes_BAR)),
			},
		}.Marshal(),
	},
	{
		desc: "basic repeated types (zero-length packed encoding)",
		decodeTo: makeMessages(protobuild.Message{
			"repeated_int32":       []int32{},
			"repeated_int64":       []int64{},
			"repeated_uint32":      []uint32{},
			"repeated_uint64":      []uint64{},
			"repeated_sint32":      []int32{},
			"repeated_sint64":      []int64{},
			"repeated_fixed32":     []uint32{},
			"repeated_fixed64":     []uint64{},
			"repeated_sfixed32":    []int32{},
			"repeated_sfixed64":    []int64{},
			"repeated_float":       []float32{},
			"repeated_double":      []float64{},
			"repeated_bool":        []bool{},
			"repeated_nested_enum": []string{},
		}),
		wire: pack.Message{
			pack.Tag{31, pack.BytesType}, pack.LengthPrefix{},
			pack.Tag{32, pack.BytesType}, pack.LengthPrefix{},
			pack.Tag{33, pack.BytesType}, pack.LengthPrefix{},
			pack.Tag{34, pack.BytesType}, pack.LengthPrefix{},
			pack.Tag{35, pack.BytesType}, pack.LengthPrefix{},
			pack.Tag{36, pack.BytesType}, pack.LengthPrefix{},
			pack.Tag{37, pack.BytesType}, pack.LengthPrefix{},
			pack.Tag{38, pack.BytesType}, pack.LengthPrefix{},
			pack.Tag{39, pack.BytesType}, pack.LengthPrefix{},
			pack.Tag{40, pack.BytesType}, pack.LengthPrefix{},
			pack.Tag{41, pack.BytesType}, pack.LengthPrefix{},
			pack.Tag{42, pack.BytesType}, pack.LengthPrefix{},
			pack.Tag{43, pack.BytesType}, pack.LengthPrefix{},
			pack.Tag{51, pack.BytesType}, pack.LengthPrefix{},
		}.Marshal(),
	},
	{
		desc: "packed repeated types",
		decodeTo: makeMessages(protobuild.Message{
			"packed_int32":    []int32{1001, 2001},
			"packed_int64":    []int64{1002, 2002},
			"packed_uint32":   []uint32{1003, 2003},
			"packed_uint64":   []uint64{1004, 2004},
			"packed_sint32":   []int32{1005, 2005},
			"packed_sint64":   []int64{1006, 2006},
			"packed_fixed32":  []uint32{1007, 2007},
			"packed_fixed64":  []uint64{1008, 2008},
			"packed_sfixed32": []int32{1009, 2009},
			"packed_sfixed64": []int64{1010, 2010},
			"packed_float":    []float32{1011.5, 2011.5},
			"packed_double":   []float64{1012.5, 2012.5},
			"packed_bool":     []bool{true, false},
			"packed_enum":     []string{"FOREIGN_FOO", "FOREIGN_BAR"},
		}, &testpb.TestPackedTypes{}, &testpb.TestPackedExtensions{}),
		wire: pack.Message{
			pack.Tag{90, pack.BytesType}, pack.LengthPrefix{
				pack.Varint(1001), pack.Varint(2001),
			},
			pack.Tag{91, pack.BytesType}, pack.LengthPrefix{
				pack.Varint(1002), pack.Varint(2002),
			},
			pack.Tag{92, pack.BytesType}, pack.LengthPrefix{
				pack.Uvarint(1003), pack.Uvarint(2003),
			},
			pack.Tag{93, pack.BytesType}, pack.LengthPrefix{
				pack.Uvarint(1004), pack.Uvarint(2004),
			},
			pack.Tag{94, pack.BytesType}, pack.LengthPrefix{
				pack.Svarint(1005), pack.Svarint(2005),
			},
			pack.Tag{95, pack.BytesType}, pack.LengthPrefix{
				pack.Svarint(1006), pack.Svarint(2006),
			},
			pack.Tag{96, pack.BytesType}, pack.LengthPrefix{
				pack.Uint32(1007), pack.Uint32(2007),
			},
			pack.Tag{97, pack.BytesType}, pack.LengthPrefix{
				pack.Uint64(1008), pack.Uint64(2008),
			},
			pack.Tag{98, pack.BytesType}, pack.LengthPrefix{
				pack.Int32(1009), pack.Int32(2009),
			},
			pack.Tag{99, pack.BytesType}, pack.LengthPrefix{
				pack.Int64(1010), pack.Int64(2010),
			},
			pack.Tag{100, pack.BytesType}, pack.LengthPrefix{
				pack.Float32(1011.5), pack.Float32(2011.5),
			},
			pack.Tag{101, pack.BytesType}, pack.LengthPrefix{
				pack.Float64(1012.5), pack.Float64(2012.5),
			},
			pack.Tag{102, pack.BytesType}, pack.LengthPrefix{
				pack.Bool(true), pack.Bool(false),
			},
			pack.Tag{103, pack.BytesType}, pack.LengthPrefix{
				pack.Varint(int(testpb.ForeignEnum_FOREIGN_FOO)),
				pack.Varint(int(testpb.ForeignEnum_FOREIGN_BAR)),
			},
		}.Marshal(),
	},
	{
		desc: "packed repeated types (zero length)",
		decodeTo: makeMessages(protobuild.Message{
			"packed_int32":    []int32{},
			"packed_int64":    []int64{},
			"packed_uint32":   []uint32{},
			"packed_uint64":   []uint64{},
			"packed_sint32":   []int32{},
			"packed_sint64":   []int64{},
			"packed_fixed32":  []uint32{},
			"packed_fixed64":  []uint64{},
			"packed_sfixed32": []int32{},
			"packed_sfixed64": []int64{},
			"packed_float":    []float32{},
			"packed_double":   []float64{},
			"packed_bool":     []bool{},
			"packed_enum":     []string{},
		}, &testpb.TestPackedTypes{}, &testpb.TestPackedExtensions{}),
		wire: pack.Message{
			pack.Tag{90, pack.BytesType}, pack.LengthPrefix{},
			pack.Tag{91, pack.BytesType}, pack.LengthPrefix{},
			pack.Tag{92, pack.BytesType}, pack.LengthPrefix{},
			pack.Tag{93, pack.BytesType}, pack.LengthPrefix{},
			pack.Tag{94, pack.BytesType}, pack.LengthPrefix{},
			pack.Tag{95, pack.BytesType}, pack.LengthPrefix{},
			pack.Tag{96, pack.BytesType}, pack.LengthPrefix{},
			pack.Tag{97, pack.BytesType}, pack.LengthPrefix{},
			pack.Tag{98, pack.BytesType}, pack.LengthPrefix{},
			pack.Tag{99, pack.BytesType}, pack.LengthPrefix{},
			pack.Tag{100, pack.BytesType}, pack.LengthPrefix{},
			pack.Tag{101, pack.BytesType}, pack.LengthPrefix{},
			pack.Tag{102, pack.BytesType}, pack.LengthPrefix{},
			pack.Tag{103, pack.BytesType}, pack.LengthPrefix{},
		}.Marshal(),
	},
	{
		desc: "repeated messages",
		decodeTo: makeMessages(protobuild.Message{
			"repeated_nested_message": []protobuild.Message{
				{"a": 1},
				{},
				{"a": 2},
			},
		}),
		wire: pack.Message{
			pack.Tag{48, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(1),
			}),
			pack.Tag{48, pack.BytesType}, pack.LengthPrefix(pack.Message{}),
			pack.Tag{48, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(2),
			}),
		}.Marshal(),
	},
	{
		desc: "repeated nil messages",
		decodeTo: []proto.Message{&testpb.TestAllTypes{
			RepeatedNestedMessage: []*testpb.TestAllTypes_NestedMessage{
				{A: proto.Int32(1)},
				nil,
				{A: proto.Int32(2)},
			},
		}, &test3pb.TestAllTypes{
			RepeatedNestedMessage: []*test3pb.TestAllTypes_NestedMessage{
				{A: 1},
				nil,
				{A: 2},
			},
		}, build(
			&testpb.TestAllExtensions{},
			extend(testpb.E_RepeatedNestedMessage, []*testpb.TestAllExtensions_NestedMessage{
				{A: proto.Int32(1)},
				nil,
				{A: proto.Int32(2)},
			}),
		)},
		wire: pack.Message{
			pack.Tag{48, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(1),
			}),
			pack.Tag{48, pack.BytesType}, pack.LengthPrefix(pack.Message{}),
			pack.Tag{48, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(2),
			}),
		}.Marshal(),
	},
	{
		desc: "repeated groups",
		decodeTo: makeMessages(protobuild.Message{
			"repeatedgroup": []protobuild.Message{
				{"a": 1017},
				{},
				{"a": 2017},
			},
		}, &testpb.TestAllTypes{}, &testpb.TestAllExtensions{}),
		wire: pack.Message{
			pack.Tag{46, pack.StartGroupType},
			pack.Tag{47, pack.VarintType}, pack.Varint(1017),
			pack.Tag{46, pack.EndGroupType},
			pack.Tag{46, pack.StartGroupType},
			pack.Tag{46, pack.EndGroupType},
			pack.Tag{46, pack.StartGroupType},
			pack.Tag{47, pack.VarintType}, pack.Varint(2017),
			pack.Tag{46, pack.EndGroupType},
		}.Marshal(),
	},
	{
		desc: "repeated nil groups",
		decodeTo: []proto.Message{&testpb.TestAllTypes{
			Repeatedgroup: []*testpb.TestAllTypes_RepeatedGroup{
				{A: proto.Int32(1017)},
				nil,
				{A: proto.Int32(2017)},
			},
		}, build(
			&testpb.TestAllExtensions{},
			extend(testpb.E_Repeatedgroup, []*testpb.RepeatedGroup{
				{A: proto.Int32(1017)},
				nil,
				{A: proto.Int32(2017)},
			}),
		)},
		wire: pack.Message{
			pack.Tag{46, pack.StartGroupType},
			pack.Tag{47, pack.VarintType}, pack.Varint(1017),
			pack.Tag{46, pack.EndGroupType},
			pack.Tag{46, pack.StartGroupType},
			pack.Tag{46, pack.EndGroupType},
			pack.Tag{46, pack.StartGroupType},
			pack.Tag{47, pack.VarintType}, pack.Varint(2017),
			pack.Tag{46, pack.EndGroupType},
		}.Marshal(),
	},
	{
		desc: "maps",
		decodeTo: makeMessages(protobuild.Message{
			"map_int32_int32":       map[int32]int32{1056: 1156, 2056: 2156},
			"map_int64_int64":       map[int64]int64{1057: 1157, 2057: 2157},
			"map_uint32_uint32":     map[uint32]uint32{1058: 1158, 2058: 2158},
			"map_uint64_uint64":     map[uint64]uint64{1059: 1159, 2059: 2159},
			"map_sint32_sint32":     map[int32]int32{1060: 1160, 2060: 2160},
			"map_sint64_sint64":     map[int64]int64{1061: 1161, 2061: 2161},
			"map_fixed32_fixed32":   map[uint32]uint32{1062: 1162, 2062: 2162},
			"map_fixed64_fixed64":   map[uint64]uint64{1063: 1163, 2063: 2163},
			"map_sfixed32_sfixed32": map[int32]int32{1064: 1164, 2064: 2164},
			"map_sfixed64_sfixed64": map[int64]int64{1065: 1165, 2065: 2165},
			"map_int32_float":       map[int32]float32{1066: 1166.5, 2066: 2166.5},
			"map_int32_double":      map[int32]float64{1067: 1167.5, 2067: 2167.5},
			"map_bool_bool":         map[bool]bool{true: false, false: true},
			"map_string_string":     map[string]string{"69.1.key": "69.1.val", "69.2.key": "69.2.val"},
			"map_string_bytes":      map[string][]byte{"70.1.key": []byte("70.1.val"), "70.2.key": []byte("70.2.val")},
			"map_string_nested_message": map[string]protobuild.Message{
				"71.1.key": {"a": 1171},
				"71.2.key": {"a": 2171},
			},
			"map_string_nested_enum": map[string]string{"73.1.key": "FOO", "73.2.key": "BAR"},
		}, &testpb.TestAllTypes{}, &test3pb.TestAllTypes{}),
		wire: pack.Message{
			pack.Tag{56, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(1056),
				pack.Tag{2, pack.VarintType}, pack.Varint(1156),
			}),
			pack.Tag{56, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(2056),
				pack.Tag{2, pack.VarintType}, pack.Varint(2156),
			}),
			pack.Tag{57, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(1057),
				pack.Tag{2, pack.VarintType}, pack.Varint(1157),
			}),
			pack.Tag{57, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(2057),
				pack.Tag{2, pack.VarintType}, pack.Varint(2157),
			}),
			pack.Tag{58, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(1058),
				pack.Tag{2, pack.VarintType}, pack.Varint(1158),
			}),
			pack.Tag{58, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(2058),
				pack.Tag{2, pack.VarintType}, pack.Varint(2158),
			}),
			pack.Tag{59, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(1059),
				pack.Tag{2, pack.VarintType}, pack.Varint(1159),
			}),
			pack.Tag{59, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(2059),
				pack.Tag{2, pack.VarintType}, pack.Varint(2159),
			}),
			pack.Tag{60, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Svarint(1060),
				pack.Tag{2, pack.VarintType}, pack.Svarint(1160),
			}),
			pack.Tag{60, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Svarint(2060),
				pack.Tag{2, pack.VarintType}, pack.Svarint(2160),
			}),
			pack.Tag{61, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Svarint(1061),
				pack.Tag{2, pack.VarintType}, pack.Svarint(1161),
			}),
			pack.Tag{61, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Svarint(2061),
				pack.Tag{2, pack.VarintType}, pack.Svarint(2161),
			}),
			pack.Tag{62, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.Fixed32Type}, pack.Int32(1062),
				pack.Tag{2, pack.Fixed32Type}, pack.Int32(1162),
			}),
			pack.Tag{62, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.Fixed32Type}, pack.Int32(2062),
				pack.Tag{2, pack.Fixed32Type}, pack.Int32(2162),
			}),
			pack.Tag{63, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.Fixed64Type}, pack.Int64(1063),
				pack.Tag{2, pack.Fixed64Type}, pack.Int64(1163),
			}),
			pack.Tag{63, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.Fixed64Type}, pack.Int64(2063),
				pack.Tag{2, pack.Fixed64Type}, pack.Int64(2163),
			}),
			pack.Tag{64, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.Fixed32Type}, pack.Int32(1064),
				pack.Tag{2, pack.Fixed32Type}, pack.Int32(1164),
			}),
			pack.Tag{64, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.Fixed32Type}, pack.Int32(2064),
				pack.Tag{2, pack.Fixed32Type}, pack.Int32(2164),
			}),
			pack.Tag{65, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.Fixed64Type}, pack.Int64(1065),
				pack.Tag{2, pack.Fixed64Type}, pack.Int64(1165),
			}),
			pack.Tag{65, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.Fixed64Type}, pack.Int64(2065),
				pack.Tag{2, pack.Fixed64Type}, pack.Int64(2165),
			}),
			pack.Tag{66, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(1066),
				pack.Tag{2, pack.Fixed32Type}, pack.Float32(1166.5),
			}),
			pack.Tag{66, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(2066),
				pack.Tag{2, pack.Fixed32Type}, pack.Float32(2166.5),
			}),
			pack.Tag{67, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(1067),
				pack.Tag{2, pack.Fixed64Type}, pack.Float64(1167.5),
			}),
			pack.Tag{67, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(2067),
				pack.Tag{2, pack.Fixed64Type}, pack.Float64(2167.5),
			}),
			pack.Tag{68, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Bool(true),
				pack.Tag{2, pack.VarintType}, pack.Bool(false),
			}),
			pack.Tag{68, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Bool(false),
				pack.Tag{2, pack.VarintType}, pack.Bool(true),
			}),
			pack.Tag{69, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.BytesType}, pack.String("69.1.key"),
				pack.Tag{2, pack.BytesType}, pack.String("69.1.val"),
			}),
			pack.Tag{69, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.BytesType}, pack.String("69.2.key"),
				pack.Tag{2, pack.BytesType}, pack.String("69.2.val"),
			}),
			pack.Tag{70, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.BytesType}, pack.String("70.1.key"),
				pack.Tag{2, pack.BytesType}, pack.String("70.1.val"),
			}),
			pack.Tag{70, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.BytesType}, pack.String("70.2.key"),
				pack.Tag{2, pack.BytesType}, pack.String("70.2.val"),
			}),
			pack.Tag{71, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.BytesType}, pack.String("71.1.key"),
				pack.Tag{2, pack.BytesType}, pack.LengthPrefix(pack.Message{
					pack.Tag{1, pack.VarintType}, pack.Varint(1171),
				}),
			}),
			pack.Tag{71, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.BytesType}, pack.String("71.2.key"),
				pack.Tag{2, pack.BytesType}, pack.LengthPrefix(pack.Message{
					pack.Tag{1, pack.VarintType}, pack.Varint(2171),
				}),
			}),
			pack.Tag{73, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.BytesType}, pack.String("73.1.key"),
				pack.Tag{2, pack.VarintType}, pack.Varint(int(testpb.TestAllTypes_FOO)),
			}),
			pack.Tag{73, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.BytesType}, pack.String("73.2.key"),
				pack.Tag{2, pack.VarintType}, pack.Varint(int(testpb.TestAllTypes_BAR)),
			}),
		}.Marshal(),
	},
	{
		desc: "map with value before key",
		decodeTo: makeMessages(protobuild.Message{
			"map_int32_int32": map[int32]int32{1056: 1156},
			"map_string_nested_message": map[string]protobuild.Message{
				"71.1.key": {"a": 1171},
			},
		}, &testpb.TestAllTypes{}, &test3pb.TestAllTypes{}),
		wire: pack.Message{
			pack.Tag{56, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{2, pack.VarintType}, pack.Varint(1156),
				pack.Tag{1, pack.VarintType}, pack.Varint(1056),
			}),
			pack.Tag{71, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{2, pack.BytesType}, pack.LengthPrefix(pack.Message{
					pack.Tag{1, pack.VarintType}, pack.Varint(1171),
				}),
				pack.Tag{1, pack.BytesType}, pack.String("71.1.key"),
			}),
		}.Marshal(),
	},
	{
		desc: "map with repeated key and value",
		decodeTo: makeMessages(protobuild.Message{
			"map_int32_int32": map[int32]int32{1056: 1156},
			"map_string_nested_message": map[string]protobuild.Message{
				"71.1.key": {"a": 1171},
			},
		}, &testpb.TestAllTypes{}, &test3pb.TestAllTypes{}),
		wire: pack.Message{
			pack.Tag{56, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(0),
				pack.Tag{2, pack.VarintType}, pack.Varint(0),
				pack.Tag{1, pack.VarintType}, pack.Varint(1056),
				pack.Tag{2, pack.VarintType}, pack.Varint(1156),
			}),
			pack.Tag{71, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.BytesType}, pack.String(0),
				pack.Tag{2, pack.BytesType}, pack.LengthPrefix(pack.Message{}),
				pack.Tag{1, pack.BytesType}, pack.String("71.1.key"),
				pack.Tag{2, pack.BytesType}, pack.LengthPrefix(pack.Message{
					pack.Tag{1, pack.VarintType}, pack.Varint(1171),
				}),
			}),
		}.Marshal(),
	},
	{
		desc: "oneof (uint32)",
		decodeTo: makeMessages(protobuild.Message{
			"oneof_uint32": 1111,
		}, &testpb.TestAllTypes{}, &test3pb.TestAllTypes{}),
		wire: pack.Message{pack.Tag{111, pack.VarintType}, pack.Varint(1111)}.Marshal(),
	},
	{
		desc: "oneof (message)",
		decodeTo: makeMessages(protobuild.Message{
			"oneof_nested_message": protobuild.Message{
				"a": 1112,
			},
		}, &testpb.TestAllTypes{}, &test3pb.TestAllTypes{}),
		wire: pack.Message{pack.Tag{112, pack.BytesType}, pack.LengthPrefix(pack.Message{
			pack.Message{pack.Tag{1, pack.VarintType}, pack.Varint(1112)},
		})}.Marshal(),
	},
	{
		desc: "oneof (empty message)",
		decodeTo: makeMessages(protobuild.Message{
			"oneof_nested_message": protobuild.Message{},
		}, &testpb.TestAllTypes{}, &test3pb.TestAllTypes{}),
		wire: pack.Message{pack.Tag{112, pack.BytesType}, pack.LengthPrefix(pack.Message{})}.Marshal(),
	},
	{
		desc: "oneof (merged message)",
		decodeTo: makeMessages(protobuild.Message{
			"oneof_nested_message": protobuild.Message{
				"a": 1,
				"corecursive": protobuild.Message{
					"optional_int32": 43,
				},
			},
		}, &testpb.TestAllTypes{}, &test3pb.TestAllTypes{}),
		wire: pack.Message{
			pack.Tag{112, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Message{pack.Tag{1, pack.VarintType}, pack.Varint(1)},
			}),
			pack.Tag{112, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{2, pack.BytesType}, pack.LengthPrefix(pack.Message{
					pack.Tag{1, pack.VarintType}, pack.Varint(43),
				}),
			}),
		}.Marshal(),
	},
	{
		desc: "oneof (string)",
		decodeTo: makeMessages(protobuild.Message{
			"oneof_string": "1113",
		}, &testpb.TestAllTypes{}, &test3pb.TestAllTypes{}),
		wire: pack.Message{pack.Tag{113, pack.BytesType}, pack.String("1113")}.Marshal(),
	},
	{
		desc: "oneof (bytes)",
		decodeTo: makeMessages(protobuild.Message{
			"oneof_bytes": "1114",
		}, &testpb.TestAllTypes{}, &test3pb.TestAllTypes{}),
		wire: pack.Message{pack.Tag{114, pack.BytesType}, pack.String("1114")}.Marshal(),
	},
	{
		desc: "oneof (bool)",
		decodeTo: makeMessages(protobuild.Message{
			"oneof_bool": true,
		}, &testpb.TestAllTypes{}, &test3pb.TestAllTypes{}),
		wire: pack.Message{pack.Tag{115, pack.VarintType}, pack.Bool(true)}.Marshal(),
	},
	{
		desc: "oneof (uint64)",
		decodeTo: makeMessages(protobuild.Message{
			"oneof_uint64": 116,
		}, &testpb.TestAllTypes{}, &test3pb.TestAllTypes{}),
		wire: pack.Message{pack.Tag{116, pack.VarintType}, pack.Varint(116)}.Marshal(),
	},
	{
		desc: "oneof (float)",
		decodeTo: makeMessages(protobuild.Message{
			"oneof_float": 117.5,
		}, &testpb.TestAllTypes{}, &test3pb.TestAllTypes{}),
		wire: pack.Message{pack.Tag{117, pack.Fixed32Type}, pack.Float32(117.5)}.Marshal(),
	},
	{
		desc: "oneof (double)",
		decodeTo: makeMessages(protobuild.Message{
			"oneof_double": 118.5,
		}, &testpb.TestAllTypes{}, &test3pb.TestAllTypes{}),
		wire: pack.Message{pack.Tag{118, pack.Fixed64Type}, pack.Float64(118.5)}.Marshal(),
	},
	{
		desc: "oneof (enum)",
		decodeTo: makeMessages(protobuild.Message{
			"oneof_enum": "BAR",
		}, &testpb.TestAllTypes{}, &test3pb.TestAllTypes{}),
		wire: pack.Message{pack.Tag{119, pack.VarintType}, pack.Varint(int(testpb.TestAllTypes_BAR))}.Marshal(),
	},
	{
		desc: "oneof (zero)",
		decodeTo: makeMessages(protobuild.Message{
			"oneof_uint64": 0,
		}, &testpb.TestAllTypes{}, &test3pb.TestAllTypes{}),
		wire: pack.Message{pack.Tag{116, pack.VarintType}, pack.Varint(0)}.Marshal(),
	},
	{
		desc: "oneof (overridden value)",
		decodeTo: makeMessages(protobuild.Message{
			"oneof_uint64": 2,
		}, &testpb.TestAllTypes{}, &test3pb.TestAllTypes{}),
		wire: pack.Message{
			pack.Tag{111, pack.VarintType}, pack.Varint(1),
			pack.Tag{116, pack.VarintType}, pack.Varint(2),
		}.Marshal(),
	},
	// TODO: More unknown field tests for ordering, repeated fields, etc.
	//
	// It is currently impossible to produce results that the v1 Equal
	// considers equivalent to those of the v1 decoder. Figure out if
	// that's a problem or not.
	{
		desc:          "unknown fields",
		checkFastInit: true,
		decodeTo: makeMessages(protobuild.Message{
			protobuild.Unknown: pack.Message{
				pack.Tag{100000, pack.VarintType}, pack.Varint(1),
			}.Marshal(),
		}),
		wire: pack.Message{
			pack.Tag{100000, pack.VarintType}, pack.Varint(1),
		}.Marshal(),
	},
	{
		desc: "discarded unknown fields",
		unmarshalOptions: proto.UnmarshalOptions{
			DiscardUnknown: true,
		},
		decodeTo: makeMessages(protobuild.Message{}),
		wire: pack.Message{
			pack.Tag{100000, pack.VarintType}, pack.Varint(1),
		}.Marshal(),
	},
	{
		desc: "field type mismatch",
		decodeTo: makeMessages(protobuild.Message{
			protobuild.Unknown: pack.Message{
				pack.Tag{1, pack.BytesType}, pack.String("string"),
			}.Marshal(),
		}),
		wire: pack.Message{
			pack.Tag{1, pack.BytesType}, pack.String("string"),
		}.Marshal(),
	},
	{
		desc: "map field element mismatch",
		decodeTo: makeMessages(protobuild.Message{
			"map_int32_int32": map[int32]int32{1: 0},
		}, &testpb.TestAllTypes{}, &test3pb.TestAllTypes{}),
		wire: pack.Message{
			pack.Tag{56, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(1),
				pack.Tag{2, pack.BytesType}, pack.String("string"),
			}),
		}.Marshal(),
	},
	{
		desc:          "required field in nil message unset",
		checkFastInit: true,
		partial:       true,
		decodeTo:      []proto.Message{(*testpb.TestRequired)(nil)},
	},
	{
		desc:          "required int32 unset",
		checkFastInit: true,
		partial:       true,
		decodeTo:      makeMessages(protobuild.Message{}, &requiredpb.Int32{}),
	},
	{
		desc:          "required int32 set",
		checkFastInit: true,
		decodeTo: makeMessages(protobuild.Message{
			"v": 1,
		}, &requiredpb.Int32{}),
		wire: pack.Message{
			pack.Tag{1, pack.VarintType}, pack.Varint(1),
		}.Marshal(),
	},
	{
		desc:          "required fixed32 unset",
		checkFastInit: true,
		partial:       true,
		decodeTo:      makeMessages(protobuild.Message{}, &requiredpb.Fixed32{}),
	},
	{
		desc:          "required fixed32 set",
		checkFastInit: true,
		decodeTo: makeMessages(protobuild.Message{
			"v": 1,
		}, &requiredpb.Fixed32{}),
		wire: pack.Message{
			pack.Tag{1, pack.Fixed32Type}, pack.Int32(1),
		}.Marshal(),
	},
	{
		desc:          "required fixed64 unset",
		checkFastInit: true,
		partial:       true,
		decodeTo:      makeMessages(protobuild.Message{}, &requiredpb.Fixed64{}),
	},
	{
		desc:          "required fixed64 set",
		checkFastInit: true,
		decodeTo: makeMessages(protobuild.Message{
			"v": 1,
		}, &requiredpb.Fixed64{}),
		wire: pack.Message{
			pack.Tag{1, pack.Fixed64Type}, pack.Int64(1),
		}.Marshal(),
	},
	{
		desc:          "required bytes unset",
		checkFastInit: true,
		partial:       true,
		decodeTo:      makeMessages(protobuild.Message{}, &requiredpb.Bytes{}),
	},
	{
		desc:          "required bytes set",
		checkFastInit: true,
		decodeTo: makeMessages(protobuild.Message{
			"v": "",
		}, &requiredpb.Bytes{}),
		wire: pack.Message{
			pack.Tag{1, pack.BytesType}, pack.Bytes(nil),
		}.Marshal(),
	},
	{
		desc:          "required field with incompatible wire type",
		checkFastInit: true,
		partial:       true,
		decodeTo: []proto.Message{build(
			&testpb.TestRequired{},
			unknown(pack.Message{
				pack.Tag{1, pack.Fixed32Type}, pack.Int32(2),
			}.Marshal()),
		)},
		wire: pack.Message{
			pack.Tag{1, pack.Fixed32Type}, pack.Int32(2),
		}.Marshal(),
	},
	{
		desc:          "required field in optional message unset",
		checkFastInit: true,
		partial:       true,
		decodeTo: makeMessages(protobuild.Message{
			"optional_message": protobuild.Message{},
		}, &testpb.TestRequiredForeign{}),
		wire: pack.Message{
			pack.Tag{1, pack.BytesType}, pack.LengthPrefix(pack.Message{}),
		}.Marshal(),
	},
	{
		desc:          "required field in optional message set",
		checkFastInit: true,
		decodeTo: makeMessages(protobuild.Message{
			"optional_message": protobuild.Message{
				"required_field": 1,
			},
		}, &testpb.TestRequiredForeign{}),
		wire: pack.Message{
			pack.Tag{1, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(1),
			}),
		}.Marshal(),
	},
	{
		desc:             "required field in optional message set (split across multiple tags)",
		checkFastInit:    false, // fast init checks don't handle split messages
		nocheckValidInit: true,  // validation doesn't either
		decodeTo: makeMessages(protobuild.Message{
			"optional_message": protobuild.Message{
				"required_field": 1,
			},
		}, &testpb.TestRequiredForeign{}),
		wire: pack.Message{
			pack.Tag{1, pack.BytesType}, pack.LengthPrefix(pack.Message{}),
			pack.Tag{1, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(1),
			}),
		}.Marshal(),
	},
	{
		desc:          "required field in repeated message unset",
		checkFastInit: true,
		partial:       true,
		decodeTo: makeMessages(protobuild.Message{
			"repeated_message": []protobuild.Message{
				{"required_field": 1},
				{},
			},
		}, &testpb.TestRequiredForeign{}),
		wire: pack.Message{
			pack.Tag{2, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(1),
			}),
			pack.Tag{2, pack.BytesType}, pack.LengthPrefix(pack.Message{}),
		}.Marshal(),
	},
	{
		desc:          "required field in repeated message set",
		checkFastInit: true,
		decodeTo: makeMessages(protobuild.Message{
			"repeated_message": []protobuild.Message{
				{"required_field": 1},
				{"required_field": 2},
			},
		}, &testpb.TestRequiredForeign{}),
		wire: pack.Message{
			pack.Tag{2, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(1),
			}),
			pack.Tag{2, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(2),
			}),
		}.Marshal(),
	},
	{
		desc:          "required field in map message unset",
		checkFastInit: true,
		partial:       true,
		decodeTo: makeMessages(protobuild.Message{
			"map_message": map[int32]protobuild.Message{
				1: {"required_field": 1},
				2: {},
			},
		}, &testpb.TestRequiredForeign{}),
		wire: pack.Message{
			pack.Tag{3, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(1),
				pack.Tag{2, pack.BytesType}, pack.LengthPrefix(pack.Message{
					pack.Tag{1, pack.VarintType}, pack.Varint(1),
				}),
			}),
			pack.Tag{3, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(2),
				pack.Tag{2, pack.BytesType}, pack.LengthPrefix(pack.Message{}),
			}),
		}.Marshal(),
	},
	{
		desc:          "required field in absent map message value",
		checkFastInit: true,
		partial:       true,
		decodeTo: makeMessages(protobuild.Message{
			"map_message": map[int32]protobuild.Message{
				2: {},
			},
		}, &testpb.TestRequiredForeign{}),
		wire: pack.Message{
			pack.Tag{3, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(2),
			}),
		}.Marshal(),
	},
	{
		desc:          "required field in map message set",
		checkFastInit: true,
		decodeTo: makeMessages(protobuild.Message{
			"map_message": map[int32]protobuild.Message{
				1: {"required_field": 1},
				2: {"required_field": 2},
			},
		}, &testpb.TestRequiredForeign{}),
		wire: pack.Message{
			pack.Tag{3, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(1),
				pack.Tag{2, pack.BytesType}, pack.LengthPrefix(pack.Message{
					pack.Tag{1, pack.VarintType}, pack.Varint(1),
				}),
			}),
			pack.Tag{3, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(2),
				pack.Tag{2, pack.BytesType}, pack.LengthPrefix(pack.Message{
					pack.Tag{1, pack.VarintType}, pack.Varint(2),
				}),
			}),
		}.Marshal(),
	},
	{
		desc:          "required field in optional group unset",
		checkFastInit: true,
		partial:       true,
		decodeTo: makeMessages(protobuild.Message{
			"optionalgroup": protobuild.Message{},
		}, &testpb.TestRequiredGroupFields{}),
		wire: pack.Message{
			pack.Tag{1, pack.StartGroupType},
			pack.Tag{1, pack.EndGroupType},
		}.Marshal(),
	},
	{
		desc:          "required field in optional group set",
		checkFastInit: true,
		decodeTo: makeMessages(protobuild.Message{
			"optionalgroup": protobuild.Message{
				"a": 1,
			},
		}, &testpb.TestRequiredGroupFields{}),
		wire: pack.Message{
			pack.Tag{1, pack.StartGroupType},
			pack.Tag{2, pack.VarintType}, pack.Varint(1),
			pack.Tag{1, pack.EndGroupType},
		}.Marshal(),
	},
	{
		desc:          "required field in repeated group unset",
		checkFastInit: true,
		partial:       true,
		decodeTo: makeMessages(protobuild.Message{
			"repeatedgroup": []protobuild.Message{
				{"a": 1},
				{},
			},
		}, &testpb.TestRequiredGroupFields{}),
		wire: pack.Message{
			pack.Tag{3, pack.StartGroupType},
			pack.Tag{4, pack.VarintType}, pack.Varint(1),
			pack.Tag{3, pack.EndGroupType},
			pack.Tag{3, pack.StartGroupType},
			pack.Tag{3, pack.EndGroupType},
		}.Marshal(),
	},
	{
		desc:          "required field in repeated group set",
		checkFastInit: true,
		decodeTo: makeMessages(protobuild.Message{
			"repeatedgroup": []protobuild.Message{
				{"a": 1},
				{"a": 2},
			},
		}, &testpb.TestRequiredGroupFields{}),
		wire: pack.Message{
			pack.Tag{3, pack.StartGroupType},
			pack.Tag{4, pack.VarintType}, pack.Varint(1),
			pack.Tag{3, pack.EndGroupType},
			pack.Tag{3, pack.StartGroupType},
			pack.Tag{4, pack.VarintType}, pack.Varint(2),
			pack.Tag{3, pack.EndGroupType},
		}.Marshal(),
	},
	{
		desc:          "required field in oneof message unset",
		checkFastInit: true,
		partial:       true,
		decodeTo: makeMessages(protobuild.Message{
			"oneof_message": protobuild.Message{},
		}, &testpb.TestRequiredForeign{}),
		wire: pack.Message{pack.Tag{4, pack.BytesType}, pack.LengthPrefix(pack.Message{})}.Marshal(),
	},
	{
		desc:          "required field in oneof message set",
		checkFastInit: true,
		decodeTo: makeMessages(protobuild.Message{
			"oneof_message": protobuild.Message{
				"required_field": 1,
			},
		}, &testpb.TestRequiredForeign{}),
		wire: pack.Message{pack.Tag{4, pack.BytesType}, pack.LengthPrefix(pack.Message{
			pack.Tag{1, pack.VarintType}, pack.Varint(1),
		})}.Marshal(),
	},
	{
		desc:          "required field in extension message unset",
		checkFastInit: true,
		partial:       true,
		decodeTo: makeMessages(protobuild.Message{
			"single": protobuild.Message{},
		}, &testpb.TestAllExtensions{}),
		wire: pack.Message{
			pack.Tag{1000, pack.BytesType}, pack.LengthPrefix(pack.Message{}),
		}.Marshal(),
	},
	{
		desc:          "required field in extension message set",
		checkFastInit: true,
		decodeTo: makeMessages(protobuild.Message{
			"single": protobuild.Message{
				"required_field": 1,
			},
		}, &testpb.TestAllExtensions{}),
		wire: pack.Message{
			pack.Tag{1000, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(1),
			}),
		}.Marshal(),
	},
	{
		desc:          "required field in repeated extension message unset",
		checkFastInit: true,
		partial:       true,
		decodeTo: makeMessages(protobuild.Message{
			"multi": []protobuild.Message{
				{"required_field": 1},
				{},
			},
		}, &testpb.TestAllExtensions{}),
		wire: pack.Message{
			pack.Tag{1001, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(1),
			}),
			pack.Tag{1001, pack.BytesType}, pack.LengthPrefix(pack.Message{}),
		}.Marshal(),
	},
	{
		desc:          "required field in repeated extension message set",
		checkFastInit: true,
		decodeTo: makeMessages(protobuild.Message{
			"multi": []protobuild.Message{
				{"required_field": 1},
				{"required_field": 2},
			},
		}, &testpb.TestAllExtensions{}),
		wire: pack.Message{
			pack.Tag{1001, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(1),
			}),
			pack.Tag{1001, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(2),
			}),
		}.Marshal(),
	},
	{
		desc: "nil messages",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*test3pb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
	},
	{
		desc:    "legacy",
		partial: true,
		decodeTo: makeMessages(protobuild.Message{
			"f1": protobuild.Message{
				"optional_int32":      1,
				"optional_child_enum": "ALPHA",
				"optional_child_message": protobuild.Message{
					"f1": "x",
				},
				"optionalgroup": protobuild.Message{
					"f1": "x",
				},
				"repeated_child_message": []protobuild.Message{
					{"f1": "x"},
				},
				"repeatedgroup": []protobuild.Message{
					{"f1": "x"},
				},
				"map_bool_child_message": map[bool]protobuild.Message{
					true: {"f1": "x"},
				},
				"oneof_child_message": protobuild.Message{
					"f1": "x",
				},
			},
		}, &legacypb.Legacy{}),
		wire: pack.Message{
			pack.Tag{1, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{101, pack.VarintType}, pack.Varint(1),
				pack.Tag{115, pack.VarintType}, pack.Varint(0),
				pack.Tag{116, pack.BytesType}, pack.LengthPrefix(pack.Message{
					pack.Tag{1, pack.BytesType}, pack.String("x"),
				}),
				pack.Tag{120, pack.StartGroupType},
				pack.Tag{1, pack.BytesType}, pack.String("x"),
				pack.Tag{120, pack.EndGroupType},
				pack.Tag{516, pack.BytesType}, pack.LengthPrefix(pack.Message{
					pack.Tag{1, pack.BytesType}, pack.String("x"),
				}),
				pack.Tag{520, pack.StartGroupType},
				pack.Tag{1, pack.BytesType}, pack.String("x"),
				pack.Tag{520, pack.EndGroupType},
				pack.Tag{616, pack.BytesType}, pack.LengthPrefix(pack.Message{
					pack.Tag{1, pack.VarintType}, pack.Varint(1),
					pack.Tag{2, pack.BytesType}, pack.LengthPrefix(pack.Message{
						pack.Tag{1, pack.BytesType}, pack.String("x"),
					}),
				}),
				pack.Tag{716, pack.BytesType}, pack.LengthPrefix(pack.Message{
					pack.Tag{1, pack.BytesType}, pack.String("x"),
				}),
			}),
		}.Marshal(),
		validationStatus: impl.ValidationUnknown,
	},
	{
		desc: "first reserved field number",
		decodeTo: makeMessages(protobuild.Message{
			protobuild.Unknown: pack.Message{
				pack.Tag{pack.FirstReservedNumber, pack.VarintType}, pack.Varint(1004),
			}.Marshal(),
		}),
		wire: pack.Message{
			pack.Tag{pack.FirstReservedNumber, pack.VarintType}, pack.Varint(1004),
		}.Marshal(),
	},
	{
		desc: "last reserved field number",
		decodeTo: makeMessages(protobuild.Message{
			protobuild.Unknown: pack.Message{
				pack.Tag{pack.LastReservedNumber, pack.VarintType}, pack.Varint(1005),
			}.Marshal(),
		}),
		wire: pack.Message{
			pack.Tag{pack.LastReservedNumber, pack.VarintType}, pack.Varint(1005),
		}.Marshal(),
	},
	{
		desc: "nested unknown extension",
		unmarshalOptions: proto.UnmarshalOptions{
			DiscardUnknown: true,
			Resolver: filterResolver{
				filter: func(name protoreflect.FullName) bool {
					switch name.Name() {
					case "optional_nested_message",
						"optional_int32":
						return true
					}
					return false
				},
				resolver: protoregistry.GlobalTypes,
			},
		},
		decodeTo: makeMessages(protobuild.Message{
			"optional_nested_message": protobuild.Message{
				"corecursive": protobuild.Message{
					"optional_nested_message": protobuild.Message{
						"corecursive": protobuild.Message{
							"optional_int32": 42,
						},
					},
				},
			},
		}, &testpb.TestAllExtensions{}),
		wire: pack.Message{
			pack.Tag{18, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{2, pack.BytesType}, pack.LengthPrefix(pack.Message{
					pack.Tag{18, pack.BytesType}, pack.LengthPrefix(pack.Message{
						pack.Tag{2, pack.BytesType}, pack.LengthPrefix(pack.Message{
							pack.Tag{1, pack.VarintType}, pack.Varint(42),
							pack.Tag{2, pack.VarintType}, pack.Varint(43),
						}),
					}),
				}),
			}),
		}.Marshal(),
	},
}

var testInvalidMessages = []testProto{
	{
		desc: "invalid UTF-8 in optional string field",
		decodeTo: makeMessages(protobuild.Message{
			"optional_string": "abc\xff",
		}, &test3pb.TestAllTypes{}),
		wire: pack.Message{
			pack.Tag{14, pack.BytesType}, pack.String("abc\xff"),
		}.Marshal(),
	},
	{
		desc: "invalid UTF-8 in repeated string field",
		decodeTo: makeMessages(protobuild.Message{
			"repeated_string": []string{"foo", "abc\xff"},
		}, &test3pb.TestAllTypes{}),
		wire: pack.Message{
			pack.Tag{44, pack.BytesType}, pack.String("foo"),
			pack.Tag{44, pack.BytesType}, pack.String("abc\xff"),
		}.Marshal(),
	},
	{
		desc: "invalid UTF-8 in nested message",
		decodeTo: makeMessages(protobuild.Message{
			"optional_nested_message": protobuild.Message{
				"corecursive": protobuild.Message{
					"optional_string": "abc\xff",
				},
			},
		}, &test3pb.TestAllTypes{}),
		wire: pack.Message{
			pack.Tag{18, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{2, pack.BytesType}, pack.LengthPrefix(pack.Message{
					pack.Tag{14, pack.BytesType}, pack.String("abc\xff"),
				}),
			}),
		}.Marshal(),
	},
	{
		desc: "invalid UTF-8 in oneof field",
		decodeTo: makeMessages(protobuild.Message{
			"oneof_string": "abc\xff",
		}, &test3pb.TestAllTypes{}),
		wire: pack.Message{pack.Tag{113, pack.BytesType}, pack.String("abc\xff")}.Marshal(),
	},
	{
		desc: "invalid UTF-8 in map key",
		decodeTo: makeMessages(protobuild.Message{
			"map_string_string": map[string]string{"key\xff": "val"},
		}, &test3pb.TestAllTypes{}),
		wire: pack.Message{
			pack.Tag{69, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.BytesType}, pack.String("key\xff"),
				pack.Tag{2, pack.BytesType}, pack.String("val"),
			}),
		}.Marshal(),
	},
	{
		desc: "invalid UTF-8 in map value",
		decodeTo: makeMessages(protobuild.Message{
			"map_string_string": map[string]string{"key": "val\xff"},
		}, &test3pb.TestAllTypes{}),
		wire: pack.Message{
			pack.Tag{69, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.BytesType}, pack.String("key"),
				pack.Tag{2, pack.BytesType}, pack.String("val\xff"),
			}),
		}.Marshal(),
	},
	{
		desc: "invalid field number zero",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{pack.MinValidNumber - 1, pack.VarintType}, pack.Varint(1001),
		}.Marshal(),
	},
	{
		desc: "invalid field numbers zero and one",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{pack.MinValidNumber - 1, pack.VarintType}, pack.Varint(1002),
			pack.Tag{pack.MinValidNumber, pack.VarintType}, pack.Varint(1003),
		}.Marshal(),
	},
	{
		desc: "invalid field numbers max and max+1",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{pack.MaxValidNumber, pack.VarintType}, pack.Varint(1006),
			pack.Tag{pack.MaxValidNumber + 1, pack.VarintType}, pack.Varint(1007),
		}.Marshal(),
	},
	{
		desc: "invalid field number max+1",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{pack.MaxValidNumber + 1, pack.VarintType}, pack.Varint(1008),
		}.Marshal(),
	},
	{
		desc: "invalid field number wraps int32",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Varint(2234993595104), pack.Varint(0),
		}.Marshal(),
	},
	{
		desc:     "invalid field number in map",
		decodeTo: []proto.Message{(*testpb.TestAllTypes)(nil)},
		wire: pack.Message{
			pack.Tag{56, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(1056),
				pack.Tag{2, pack.VarintType}, pack.Varint(1156),
				pack.Tag{pack.MaxValidNumber + 1, pack.VarintType}, pack.Varint(0),
			}),
		}.Marshal(),
	},
	{
		desc: "invalid tag varint",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: []byte{0xff},
	},
	{
		desc: "field number too small",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{0, pack.VarintType}, pack.Varint(0),
		}.Marshal(),
	},
	{
		desc: "field number too large",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{wire.MaxValidNumber + 1, pack.VarintType}, pack.Varint(0),
		}.Marshal(),
	},
	{
		desc: "invalid tag varint in message field",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{18, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Raw{0xff},
			}),
		}.Marshal(),
	},
	{
		desc: "invalid tag varint in repeated message field",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{48, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Raw{0xff},
			}),
		}.Marshal(),
	},
	{
		desc: "invalid varint in group field",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{16, pack.StartGroupType},
			pack.Tag{1000, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Raw{0xff},
			}),
			pack.Tag{16, pack.EndGroupType},
		}.Marshal(),
	},
	{
		desc: "invalid varint in repeated group field",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{46, pack.StartGroupType},
			pack.Tag{1001, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Raw{0xff},
			}),
			pack.Tag{46, pack.EndGroupType},
		}.Marshal(),
	},
	{
		desc: "unterminated repeated group field",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{46, pack.StartGroupType},
		}.Marshal(),
	},
	{
		desc: "invalid tag varint in map item",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
		},
		wire: pack.Message{
			pack.Tag{56, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(0),
				pack.Tag{2, pack.VarintType}, pack.Varint(0),
				pack.Raw{0xff},
			}),
		}.Marshal(),
	},
	{
		desc: "invalid tag varint in map message value",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
		},
		wire: pack.Message{
			pack.Tag{71, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(0),
				pack.Tag{2, pack.BytesType}, pack.LengthPrefix(pack.Message{
					pack.Raw{0xff},
				}),
			}),
		}.Marshal(),
	},
	{
		desc: "invalid packed int32 field",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{31, pack.BytesType}, pack.Bytes{0xff},
		}.Marshal(),
	},
	{
		desc: "invalid packed int64 field",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{32, pack.BytesType}, pack.Bytes{0xff},
		}.Marshal(),
	},
	{
		desc: "invalid packed uint32 field",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{33, pack.BytesType}, pack.Bytes{0xff},
		}.Marshal(),
	},
	{
		desc: "invalid packed uint64 field",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{34, pack.BytesType}, pack.Bytes{0xff},
		}.Marshal(),
	},
	{
		desc: "invalid packed sint32 field",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{35, pack.BytesType}, pack.Bytes{0xff},
		}.Marshal(),
	},
	{
		desc: "invalid packed sint64 field",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{36, pack.BytesType}, pack.Bytes{0xff},
		}.Marshal(),
	},
	{
		desc: "invalid packed fixed32 field",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{37, pack.BytesType}, pack.Bytes{0x00},
		}.Marshal(),
	},
	{
		desc: "invalid packed fixed64 field",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{38, pack.BytesType}, pack.Bytes{0x00},
		}.Marshal(),
	},
	{
		desc: "invalid packed sfixed32 field",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{39, pack.BytesType}, pack.Bytes{0x00},
		}.Marshal(),
	},
	{
		desc: "invalid packed sfixed64 field",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{40, pack.BytesType}, pack.Bytes{0x00},
		}.Marshal(),
	},
	{
		desc: "invalid packed float field",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{41, pack.BytesType}, pack.Bytes{0x00},
		}.Marshal(),
	},
	{
		desc: "invalid packed double field",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{42, pack.BytesType}, pack.Bytes{0x00},
		}.Marshal(),
	},
	{
		desc: "invalid packed bool field",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{43, pack.BytesType}, pack.Bytes{0xff},
		}.Marshal(),
	},
	{
		desc: "bytes field overruns message",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{18, pack.BytesType}, pack.LengthPrefix{pack.Message{
				pack.Tag{2, pack.BytesType}, pack.LengthPrefix{pack.Message{
					pack.Tag{15, pack.BytesType}, pack.Varint(2),
				}},
				pack.Tag{1, pack.VarintType}, pack.Varint(0),
			}},
		}.Marshal(),
	},
	{
		desc: "varint field overruns message",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{1, pack.VarintType},
		}.Marshal(),
	},
	{
		desc: "bytes field lacks size",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{18, pack.BytesType},
		}.Marshal(),
	},
	{
		desc: "varint overflow",
		decodeTo: []proto.Message{
			(*testpb.TestAllTypes)(nil),
			(*testpb.TestAllExtensions)(nil),
		},
		wire: pack.Message{
			pack.Tag{1, pack.VarintType},
			pack.Raw("\xff\xff\xff\xff\xff\xff\xff\xff\xff\x02"),
		}.Marshal(),
	},
}

type filterResolver struct {
	filter   func(name protoreflect.FullName) bool
	resolver protoregistry.ExtensionTypeResolver
}

func (f filterResolver) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	if !f.filter(field) {
		return nil, protoregistry.NotFound
	}
	return f.resolver.FindExtensionByName(field)
}

func (f filterResolver) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	xt, err := f.resolver.FindExtensionByNumber(message, field)
	if err != nil {
		return nil, err
	}
	if !f.filter(xt.TypeDescriptor().FullName()) {
		return nil, protoregistry.NotFound
	}
	return xt, nil
}
