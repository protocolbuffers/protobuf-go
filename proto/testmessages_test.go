// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style.
// license that can be found in the LICENSE file.

package proto_test

import (
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/internal/encoding/pack"
	"google.golang.org/protobuf/internal/encoding/wire"
	"google.golang.org/protobuf/internal/impl"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"

	legacypb "google.golang.org/protobuf/internal/testprotos/legacy"
	legacy1pb "google.golang.org/protobuf/internal/testprotos/legacy/proto2_20160225_2fc053c5"
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
}

var testValidMessages = []testProto{
	{
		desc:          "basic scalar types",
		checkFastInit: true,
		decodeTo: []proto.Message{&testpb.TestAllTypes{
			OptionalInt32:      proto.Int32(1001),
			OptionalInt64:      proto.Int64(1002),
			OptionalUint32:     proto.Uint32(1003),
			OptionalUint64:     proto.Uint64(1004),
			OptionalSint32:     proto.Int32(1005),
			OptionalSint64:     proto.Int64(1006),
			OptionalFixed32:    proto.Uint32(1007),
			OptionalFixed64:    proto.Uint64(1008),
			OptionalSfixed32:   proto.Int32(1009),
			OptionalSfixed64:   proto.Int64(1010),
			OptionalFloat:      proto.Float32(1011.5),
			OptionalDouble:     proto.Float64(1012.5),
			OptionalBool:       proto.Bool(true),
			OptionalString:     proto.String("string"),
			OptionalBytes:      []byte("bytes"),
			OptionalNestedEnum: testpb.TestAllTypes_BAR.Enum(),
		}, &test3pb.TestAllTypes{
			OptionalInt32:      1001,
			OptionalInt64:      1002,
			OptionalUint32:     1003,
			OptionalUint64:     1004,
			OptionalSint32:     1005,
			OptionalSint64:     1006,
			OptionalFixed32:    1007,
			OptionalFixed64:    1008,
			OptionalSfixed32:   1009,
			OptionalSfixed64:   1010,
			OptionalFloat:      1011.5,
			OptionalDouble:     1012.5,
			OptionalBool:       true,
			OptionalString:     "string",
			OptionalBytes:      []byte("bytes"),
			OptionalNestedEnum: test3pb.TestAllTypes_BAR,
		}, build(
			&testpb.TestAllExtensions{},
			extend(testpb.E_OptionalInt32Extension, int32(1001)),
			extend(testpb.E_OptionalInt64Extension, int64(1002)),
			extend(testpb.E_OptionalUint32Extension, uint32(1003)),
			extend(testpb.E_OptionalUint64Extension, uint64(1004)),
			extend(testpb.E_OptionalSint32Extension, int32(1005)),
			extend(testpb.E_OptionalSint64Extension, int64(1006)),
			extend(testpb.E_OptionalFixed32Extension, uint32(1007)),
			extend(testpb.E_OptionalFixed64Extension, uint64(1008)),
			extend(testpb.E_OptionalSfixed32Extension, int32(1009)),
			extend(testpb.E_OptionalSfixed64Extension, int64(1010)),
			extend(testpb.E_OptionalFloatExtension, float32(1011.5)),
			extend(testpb.E_OptionalDoubleExtension, float64(1012.5)),
			extend(testpb.E_OptionalBoolExtension, bool(true)),
			extend(testpb.E_OptionalStringExtension, string("string")),
			extend(testpb.E_OptionalBytesExtension, []byte("bytes")),
			extend(testpb.E_OptionalNestedEnumExtension, testpb.TestAllTypes_BAR),
		)},
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
		decodeTo: []proto.Message{&testpb.TestAllTypes{
			OptionalInt32:    proto.Int32(0),
			OptionalInt64:    proto.Int64(0),
			OptionalUint32:   proto.Uint32(0),
			OptionalUint64:   proto.Uint64(0),
			OptionalSint32:   proto.Int32(0),
			OptionalSint64:   proto.Int64(0),
			OptionalFixed32:  proto.Uint32(0),
			OptionalFixed64:  proto.Uint64(0),
			OptionalSfixed32: proto.Int32(0),
			OptionalSfixed64: proto.Int64(0),
			OptionalFloat:    proto.Float32(0),
			OptionalDouble:   proto.Float64(0),
			OptionalBool:     proto.Bool(false),
			OptionalString:   proto.String(""),
			OptionalBytes:    []byte{},
		}, &test3pb.TestAllTypes{}, build(
			&testpb.TestAllExtensions{},
			extend(testpb.E_OptionalInt32Extension, int32(0)),
			extend(testpb.E_OptionalInt64Extension, int64(0)),
			extend(testpb.E_OptionalUint32Extension, uint32(0)),
			extend(testpb.E_OptionalUint64Extension, uint64(0)),
			extend(testpb.E_OptionalSint32Extension, int32(0)),
			extend(testpb.E_OptionalSint64Extension, int64(0)),
			extend(testpb.E_OptionalFixed32Extension, uint32(0)),
			extend(testpb.E_OptionalFixed64Extension, uint64(0)),
			extend(testpb.E_OptionalSfixed32Extension, int32(0)),
			extend(testpb.E_OptionalSfixed64Extension, int64(0)),
			extend(testpb.E_OptionalFloatExtension, float32(0)),
			extend(testpb.E_OptionalDoubleExtension, float64(0)),
			extend(testpb.E_OptionalBoolExtension, bool(false)),
			extend(testpb.E_OptionalStringExtension, string("")),
			extend(testpb.E_OptionalBytesExtension, []byte{}),
		)},
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
		decodeTo: []proto.Message{&testpb.TestAllTypes{
			Optionalgroup: &testpb.TestAllTypes_OptionalGroup{
				A:               proto.Int32(1017),
				SameFieldNumber: proto.Int32(1016),
			},
		}, build(
			&testpb.TestAllExtensions{},
			extend(testpb.E_OptionalgroupExtension, &testpb.OptionalGroupExtension{
				A:               proto.Int32(1017),
				SameFieldNumber: proto.Int32(1016),
			}),
		)},
		wire: pack.Message{
			pack.Tag{16, pack.StartGroupType},
			pack.Tag{17, pack.VarintType}, pack.Varint(1017),
			pack.Tag{16, pack.VarintType}, pack.Varint(1016),
			pack.Tag{16, pack.EndGroupType},
		}.Marshal(),
	},
	{
		desc: "groups (field overridden)",
		decodeTo: []proto.Message{&testpb.TestAllTypes{
			Optionalgroup: &testpb.TestAllTypes_OptionalGroup{
				A: proto.Int32(2),
			},
		}, build(
			&testpb.TestAllExtensions{},
			extend(testpb.E_OptionalgroupExtension, &testpb.OptionalGroupExtension{
				A: proto.Int32(2),
			}),
		)},
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
		decodeTo: []proto.Message{&testpb.TestAllTypes{
			OptionalNestedMessage: &testpb.TestAllTypes_NestedMessage{
				A: proto.Int32(42),
				Corecursive: &testpb.TestAllTypes{
					OptionalInt32: proto.Int32(43),
				},
			},
		}, &test3pb.TestAllTypes{
			OptionalNestedMessage: &test3pb.TestAllTypes_NestedMessage{
				A: 42,
				Corecursive: &test3pb.TestAllTypes{
					OptionalInt32: 43,
				},
			},
		}, build(
			&testpb.TestAllExtensions{},
			extend(testpb.E_OptionalNestedMessageExtension, &testpb.TestAllExtensions_NestedMessage{
				A: proto.Int32(42),
				Corecursive: build(
					&testpb.TestAllExtensions{},
					extend(testpb.E_OptionalInt32Extension, int32(43)),
				).(*testpb.TestAllExtensions),
			}),
		)},
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
		decodeTo: []proto.Message{&testpb.TestAllTypes{
			OptionalNestedMessage: &testpb.TestAllTypes_NestedMessage{
				A: proto.Int32(42),
				Corecursive: &testpb.TestAllTypes{
					OptionalInt32: proto.Int32(43),
				},
			},
		}, &test3pb.TestAllTypes{
			OptionalNestedMessage: &test3pb.TestAllTypes_NestedMessage{
				A: 42,
				Corecursive: &test3pb.TestAllTypes{
					OptionalInt32: 43,
				},
			},
		}, build(
			&testpb.TestAllExtensions{},
			extend(testpb.E_OptionalNestedMessageExtension, &testpb.TestAllExtensions_NestedMessage{
				A: proto.Int32(42),
				Corecursive: build(
					&testpb.TestAllExtensions{},
					extend(testpb.E_OptionalInt32Extension, int32(43)),
				).(*testpb.TestAllExtensions),
			}),
		)},
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
		decodeTo: []proto.Message{&testpb.TestAllTypes{
			OptionalNestedMessage: &testpb.TestAllTypes_NestedMessage{
				A: proto.Int32(2),
			},
		}, &test3pb.TestAllTypes{
			OptionalNestedMessage: &test3pb.TestAllTypes_NestedMessage{
				A: 2,
			},
		}, build(
			&testpb.TestAllExtensions{},
			extend(testpb.E_OptionalNestedMessageExtension, &testpb.TestAllExtensions_NestedMessage{
				A: proto.Int32(2),
			}),
		)},
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
		decodeTo: []proto.Message{&testpb.TestAllTypes{
			RepeatedInt32:    []int32{1001, 2001},
			RepeatedInt64:    []int64{1002, 2002},
			RepeatedUint32:   []uint32{1003, 2003},
			RepeatedUint64:   []uint64{1004, 2004},
			RepeatedSint32:   []int32{1005, 2005},
			RepeatedSint64:   []int64{1006, 2006},
			RepeatedFixed32:  []uint32{1007, 2007},
			RepeatedFixed64:  []uint64{1008, 2008},
			RepeatedSfixed32: []int32{1009, 2009},
			RepeatedSfixed64: []int64{1010, 2010},
			RepeatedFloat:    []float32{1011.5, 2011.5},
			RepeatedDouble:   []float64{1012.5, 2012.5},
			RepeatedBool:     []bool{true, false},
			RepeatedString:   []string{"foo", "bar"},
			RepeatedBytes:    [][]byte{[]byte("FOO"), []byte("BAR")},
			RepeatedNestedEnum: []testpb.TestAllTypes_NestedEnum{
				testpb.TestAllTypes_FOO,
				testpb.TestAllTypes_BAR,
			},
		}, &test3pb.TestAllTypes{
			RepeatedInt32:    []int32{1001, 2001},
			RepeatedInt64:    []int64{1002, 2002},
			RepeatedUint32:   []uint32{1003, 2003},
			RepeatedUint64:   []uint64{1004, 2004},
			RepeatedSint32:   []int32{1005, 2005},
			RepeatedSint64:   []int64{1006, 2006},
			RepeatedFixed32:  []uint32{1007, 2007},
			RepeatedFixed64:  []uint64{1008, 2008},
			RepeatedSfixed32: []int32{1009, 2009},
			RepeatedSfixed64: []int64{1010, 2010},
			RepeatedFloat:    []float32{1011.5, 2011.5},
			RepeatedDouble:   []float64{1012.5, 2012.5},
			RepeatedBool:     []bool{true, false},
			RepeatedString:   []string{"foo", "bar"},
			RepeatedBytes:    [][]byte{[]byte("FOO"), []byte("BAR")},
			RepeatedNestedEnum: []test3pb.TestAllTypes_NestedEnum{
				test3pb.TestAllTypes_FOO,
				test3pb.TestAllTypes_BAR,
			},
		}, build(
			&testpb.TestAllExtensions{},
			extend(testpb.E_RepeatedInt32Extension, []int32{1001, 2001}),
			extend(testpb.E_RepeatedInt64Extension, []int64{1002, 2002}),
			extend(testpb.E_RepeatedUint32Extension, []uint32{1003, 2003}),
			extend(testpb.E_RepeatedUint64Extension, []uint64{1004, 2004}),
			extend(testpb.E_RepeatedSint32Extension, []int32{1005, 2005}),
			extend(testpb.E_RepeatedSint64Extension, []int64{1006, 2006}),
			extend(testpb.E_RepeatedFixed32Extension, []uint32{1007, 2007}),
			extend(testpb.E_RepeatedFixed64Extension, []uint64{1008, 2008}),
			extend(testpb.E_RepeatedSfixed32Extension, []int32{1009, 2009}),
			extend(testpb.E_RepeatedSfixed64Extension, []int64{1010, 2010}),
			extend(testpb.E_RepeatedFloatExtension, []float32{1011.5, 2011.5}),
			extend(testpb.E_RepeatedDoubleExtension, []float64{1012.5, 2012.5}),
			extend(testpb.E_RepeatedBoolExtension, []bool{true, false}),
			extend(testpb.E_RepeatedStringExtension, []string{"foo", "bar"}),
			extend(testpb.E_RepeatedBytesExtension, [][]byte{[]byte("FOO"), []byte("BAR")}),
			extend(testpb.E_RepeatedNestedEnumExtension, []testpb.TestAllTypes_NestedEnum{
				testpb.TestAllTypes_FOO,
				testpb.TestAllTypes_BAR,
			}),
		)},
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
		decodeTo: []proto.Message{&testpb.TestAllTypes{
			RepeatedInt32:    []int32{1001, 2001},
			RepeatedInt64:    []int64{1002, 2002},
			RepeatedUint32:   []uint32{1003, 2003},
			RepeatedUint64:   []uint64{1004, 2004},
			RepeatedSint32:   []int32{1005, 2005},
			RepeatedSint64:   []int64{1006, 2006},
			RepeatedFixed32:  []uint32{1007, 2007},
			RepeatedFixed64:  []uint64{1008, 2008},
			RepeatedSfixed32: []int32{1009, 2009},
			RepeatedSfixed64: []int64{1010, 2010},
			RepeatedFloat:    []float32{1011.5, 2011.5},
			RepeatedDouble:   []float64{1012.5, 2012.5},
			RepeatedBool:     []bool{true, false},
			RepeatedNestedEnum: []testpb.TestAllTypes_NestedEnum{
				testpb.TestAllTypes_FOO,
				testpb.TestAllTypes_BAR,
			},
		}, &test3pb.TestAllTypes{
			RepeatedInt32:    []int32{1001, 2001},
			RepeatedInt64:    []int64{1002, 2002},
			RepeatedUint32:   []uint32{1003, 2003},
			RepeatedUint64:   []uint64{1004, 2004},
			RepeatedSint32:   []int32{1005, 2005},
			RepeatedSint64:   []int64{1006, 2006},
			RepeatedFixed32:  []uint32{1007, 2007},
			RepeatedFixed64:  []uint64{1008, 2008},
			RepeatedSfixed32: []int32{1009, 2009},
			RepeatedSfixed64: []int64{1010, 2010},
			RepeatedFloat:    []float32{1011.5, 2011.5},
			RepeatedDouble:   []float64{1012.5, 2012.5},
			RepeatedBool:     []bool{true, false},
			RepeatedNestedEnum: []test3pb.TestAllTypes_NestedEnum{
				test3pb.TestAllTypes_FOO,
				test3pb.TestAllTypes_BAR,
			},
		}, build(
			&testpb.TestAllExtensions{},
			extend(testpb.E_RepeatedInt32Extension, []int32{1001, 2001}),
			extend(testpb.E_RepeatedInt64Extension, []int64{1002, 2002}),
			extend(testpb.E_RepeatedUint32Extension, []uint32{1003, 2003}),
			extend(testpb.E_RepeatedUint64Extension, []uint64{1004, 2004}),
			extend(testpb.E_RepeatedSint32Extension, []int32{1005, 2005}),
			extend(testpb.E_RepeatedSint64Extension, []int64{1006, 2006}),
			extend(testpb.E_RepeatedFixed32Extension, []uint32{1007, 2007}),
			extend(testpb.E_RepeatedFixed64Extension, []uint64{1008, 2008}),
			extend(testpb.E_RepeatedSfixed32Extension, []int32{1009, 2009}),
			extend(testpb.E_RepeatedSfixed64Extension, []int64{1010, 2010}),
			extend(testpb.E_RepeatedFloatExtension, []float32{1011.5, 2011.5}),
			extend(testpb.E_RepeatedDoubleExtension, []float64{1012.5, 2012.5}),
			extend(testpb.E_RepeatedBoolExtension, []bool{true, false}),
			extend(testpb.E_RepeatedNestedEnumExtension, []testpb.TestAllTypes_NestedEnum{
				testpb.TestAllTypes_FOO,
				testpb.TestAllTypes_BAR,
			}),
		)},
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
		decodeTo: []proto.Message{
			&testpb.TestAllTypes{},
			&test3pb.TestAllTypes{},
			build(
				&testpb.TestAllExtensions{},
				extend(testpb.E_RepeatedInt32Extension, []int32{}),
				extend(testpb.E_RepeatedInt64Extension, []int64{}),
				extend(testpb.E_RepeatedUint32Extension, []uint32{}),
				extend(testpb.E_RepeatedUint64Extension, []uint64{}),
				extend(testpb.E_RepeatedSint32Extension, []int32{}),
				extend(testpb.E_RepeatedSint64Extension, []int64{}),
				extend(testpb.E_RepeatedFixed32Extension, []uint32{}),
				extend(testpb.E_RepeatedFixed64Extension, []uint64{}),
				extend(testpb.E_RepeatedSfixed32Extension, []int32{}),
				extend(testpb.E_RepeatedSfixed64Extension, []int64{}),
				extend(testpb.E_RepeatedFloatExtension, []float32{}),
				extend(testpb.E_RepeatedDoubleExtension, []float64{}),
				extend(testpb.E_RepeatedBoolExtension, []bool{}),
				extend(testpb.E_RepeatedNestedEnumExtension, []testpb.TestAllTypes_NestedEnum{}),
			)},
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
		decodeTo: []proto.Message{&testpb.TestPackedTypes{
			PackedInt32:    []int32{1001, 2001},
			PackedInt64:    []int64{1002, 2002},
			PackedUint32:   []uint32{1003, 2003},
			PackedUint64:   []uint64{1004, 2004},
			PackedSint32:   []int32{1005, 2005},
			PackedSint64:   []int64{1006, 2006},
			PackedFixed32:  []uint32{1007, 2007},
			PackedFixed64:  []uint64{1008, 2008},
			PackedSfixed32: []int32{1009, 2009},
			PackedSfixed64: []int64{1010, 2010},
			PackedFloat:    []float32{1011.5, 2011.5},
			PackedDouble:   []float64{1012.5, 2012.5},
			PackedBool:     []bool{true, false},
			PackedEnum: []testpb.ForeignEnum{
				testpb.ForeignEnum_FOREIGN_FOO,
				testpb.ForeignEnum_FOREIGN_BAR,
			},
		}, build(
			&testpb.TestPackedExtensions{},
			extend(testpb.E_PackedInt32Extension, []int32{1001, 2001}),
			extend(testpb.E_PackedInt64Extension, []int64{1002, 2002}),
			extend(testpb.E_PackedUint32Extension, []uint32{1003, 2003}),
			extend(testpb.E_PackedUint64Extension, []uint64{1004, 2004}),
			extend(testpb.E_PackedSint32Extension, []int32{1005, 2005}),
			extend(testpb.E_PackedSint64Extension, []int64{1006, 2006}),
			extend(testpb.E_PackedFixed32Extension, []uint32{1007, 2007}),
			extend(testpb.E_PackedFixed64Extension, []uint64{1008, 2008}),
			extend(testpb.E_PackedSfixed32Extension, []int32{1009, 2009}),
			extend(testpb.E_PackedSfixed64Extension, []int64{1010, 2010}),
			extend(testpb.E_PackedFloatExtension, []float32{1011.5, 2011.5}),
			extend(testpb.E_PackedDoubleExtension, []float64{1012.5, 2012.5}),
			extend(testpb.E_PackedBoolExtension, []bool{true, false}),
			extend(testpb.E_PackedEnumExtension, []testpb.ForeignEnum{
				testpb.ForeignEnum_FOREIGN_FOO,
				testpb.ForeignEnum_FOREIGN_BAR,
			}),
		)},
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
		decodeTo: []proto.Message{
			&testpb.TestPackedTypes{},
			build(
				&testpb.TestPackedExtensions{},
				extend(testpb.E_PackedInt32Extension, []int32{}),
				extend(testpb.E_PackedInt64Extension, []int64{}),
				extend(testpb.E_PackedUint32Extension, []uint32{}),
				extend(testpb.E_PackedUint64Extension, []uint64{}),
				extend(testpb.E_PackedSint32Extension, []int32{}),
				extend(testpb.E_PackedSint64Extension, []int64{}),
				extend(testpb.E_PackedFixed32Extension, []uint32{}),
				extend(testpb.E_PackedFixed64Extension, []uint64{}),
				extend(testpb.E_PackedSfixed32Extension, []int32{}),
				extend(testpb.E_PackedSfixed64Extension, []int64{}),
				extend(testpb.E_PackedFloatExtension, []float32{}),
				extend(testpb.E_PackedDoubleExtension, []float64{}),
				extend(testpb.E_PackedBoolExtension, []bool{}),
				extend(testpb.E_PackedEnumExtension, []testpb.ForeignEnum{}),
			)},
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
			extend(testpb.E_RepeatedNestedMessageExtension, []*testpb.TestAllExtensions_NestedMessage{
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
		decodeTo: []proto.Message{&testpb.TestAllTypes{
			Repeatedgroup: []*testpb.TestAllTypes_RepeatedGroup{
				{A: proto.Int32(1017)},
				nil,
				{A: proto.Int32(2017)},
			},
		}, build(
			&testpb.TestAllExtensions{},
			extend(testpb.E_RepeatedgroupExtension, []*testpb.RepeatedGroupExtension{
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
		decodeTo: []proto.Message{&testpb.TestAllTypes{
			MapInt32Int32:       map[int32]int32{1056: 1156, 2056: 2156},
			MapInt64Int64:       map[int64]int64{1057: 1157, 2057: 2157},
			MapUint32Uint32:     map[uint32]uint32{1058: 1158, 2058: 2158},
			MapUint64Uint64:     map[uint64]uint64{1059: 1159, 2059: 2159},
			MapSint32Sint32:     map[int32]int32{1060: 1160, 2060: 2160},
			MapSint64Sint64:     map[int64]int64{1061: 1161, 2061: 2161},
			MapFixed32Fixed32:   map[uint32]uint32{1062: 1162, 2062: 2162},
			MapFixed64Fixed64:   map[uint64]uint64{1063: 1163, 2063: 2163},
			MapSfixed32Sfixed32: map[int32]int32{1064: 1164, 2064: 2164},
			MapSfixed64Sfixed64: map[int64]int64{1065: 1165, 2065: 2165},
			MapInt32Float:       map[int32]float32{1066: 1166.5, 2066: 2166.5},
			MapInt32Double:      map[int32]float64{1067: 1167.5, 2067: 2167.5},
			MapBoolBool:         map[bool]bool{true: false, false: true},
			MapStringString:     map[string]string{"69.1.key": "69.1.val", "69.2.key": "69.2.val"},
			MapStringBytes:      map[string][]byte{"70.1.key": []byte("70.1.val"), "70.2.key": []byte("70.2.val")},
			MapStringNestedMessage: map[string]*testpb.TestAllTypes_NestedMessage{
				"71.1.key": {A: proto.Int32(1171)},
				"71.2.key": {A: proto.Int32(2171)},
			},
			MapStringNestedEnum: map[string]testpb.TestAllTypes_NestedEnum{
				"73.1.key": testpb.TestAllTypes_FOO,
				"73.2.key": testpb.TestAllTypes_BAR,
			},
		}, &test3pb.TestAllTypes{
			MapInt32Int32:       map[int32]int32{1056: 1156, 2056: 2156},
			MapInt64Int64:       map[int64]int64{1057: 1157, 2057: 2157},
			MapUint32Uint32:     map[uint32]uint32{1058: 1158, 2058: 2158},
			MapUint64Uint64:     map[uint64]uint64{1059: 1159, 2059: 2159},
			MapSint32Sint32:     map[int32]int32{1060: 1160, 2060: 2160},
			MapSint64Sint64:     map[int64]int64{1061: 1161, 2061: 2161},
			MapFixed32Fixed32:   map[uint32]uint32{1062: 1162, 2062: 2162},
			MapFixed64Fixed64:   map[uint64]uint64{1063: 1163, 2063: 2163},
			MapSfixed32Sfixed32: map[int32]int32{1064: 1164, 2064: 2164},
			MapSfixed64Sfixed64: map[int64]int64{1065: 1165, 2065: 2165},
			MapInt32Float:       map[int32]float32{1066: 1166.5, 2066: 2166.5},
			MapInt32Double:      map[int32]float64{1067: 1167.5, 2067: 2167.5},
			MapBoolBool:         map[bool]bool{true: false, false: true},
			MapStringString:     map[string]string{"69.1.key": "69.1.val", "69.2.key": "69.2.val"},
			MapStringBytes:      map[string][]byte{"70.1.key": []byte("70.1.val"), "70.2.key": []byte("70.2.val")},
			MapStringNestedMessage: map[string]*test3pb.TestAllTypes_NestedMessage{
				"71.1.key": {A: 1171},
				"71.2.key": {A: 2171},
			},
			MapStringNestedEnum: map[string]test3pb.TestAllTypes_NestedEnum{
				"73.1.key": test3pb.TestAllTypes_FOO,
				"73.2.key": test3pb.TestAllTypes_BAR,
			},
		}},
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
		decodeTo: []proto.Message{&testpb.TestAllTypes{
			MapInt32Int32: map[int32]int32{1056: 1156},
			MapStringNestedMessage: map[string]*testpb.TestAllTypes_NestedMessage{
				"71.1.key": {A: proto.Int32(1171)},
			},
		}},
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
		decodeTo: []proto.Message{&testpb.TestAllTypes{
			MapInt32Int32: map[int32]int32{1056: 1156},
			MapStringNestedMessage: map[string]*testpb.TestAllTypes_NestedMessage{
				"71.1.key": {A: proto.Int32(1171)},
			},
		}},
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
		decodeTo: []proto.Message{
			&testpb.TestAllTypes{OneofField: &testpb.TestAllTypes_OneofUint32{1111}},
			&test3pb.TestAllTypes{OneofField: &test3pb.TestAllTypes_OneofUint32{1111}},
		},
		wire: pack.Message{pack.Tag{111, pack.VarintType}, pack.Varint(1111)}.Marshal(),
	},
	{
		desc: "oneof (message)",
		decodeTo: []proto.Message{
			&testpb.TestAllTypes{OneofField: &testpb.TestAllTypes_OneofNestedMessage{
				&testpb.TestAllTypes_NestedMessage{A: proto.Int32(1112)},
			}}, &test3pb.TestAllTypes{OneofField: &test3pb.TestAllTypes_OneofNestedMessage{
				&test3pb.TestAllTypes_NestedMessage{A: 1112},
			}},
		},
		wire: pack.Message{pack.Tag{112, pack.BytesType}, pack.LengthPrefix(pack.Message{
			pack.Message{pack.Tag{1, pack.VarintType}, pack.Varint(1112)},
		})}.Marshal(),
	},
	{
		desc: "oneof (empty message)",
		decodeTo: []proto.Message{
			&testpb.TestAllTypes{OneofField: &testpb.TestAllTypes_OneofNestedMessage{
				&testpb.TestAllTypes_NestedMessage{},
			}},
			&test3pb.TestAllTypes{OneofField: &test3pb.TestAllTypes_OneofNestedMessage{
				&test3pb.TestAllTypes_NestedMessage{},
			}},
		},
		wire: pack.Message{pack.Tag{112, pack.BytesType}, pack.LengthPrefix(pack.Message{})}.Marshal(),
	},
	{
		desc: "oneof (merged message)",
		decodeTo: []proto.Message{
			&testpb.TestAllTypes{OneofField: &testpb.TestAllTypes_OneofNestedMessage{
				&testpb.TestAllTypes_NestedMessage{
					A: proto.Int32(1),
					Corecursive: &testpb.TestAllTypes{
						OptionalInt32: proto.Int32(43),
					},
				},
			}}, &test3pb.TestAllTypes{OneofField: &test3pb.TestAllTypes_OneofNestedMessage{
				&test3pb.TestAllTypes_NestedMessage{
					A: 1,
					Corecursive: &test3pb.TestAllTypes{
						OptionalInt32: 43,
					},
				},
			}}},
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
		decodeTo: []proto.Message{
			&testpb.TestAllTypes{OneofField: &testpb.TestAllTypes_OneofString{"1113"}},
			&test3pb.TestAllTypes{OneofField: &test3pb.TestAllTypes_OneofString{"1113"}},
		},
		wire: pack.Message{pack.Tag{113, pack.BytesType}, pack.String("1113")}.Marshal(),
	},
	{
		desc: "oneof (bytes)",
		decodeTo: []proto.Message{
			&testpb.TestAllTypes{OneofField: &testpb.TestAllTypes_OneofBytes{[]byte("1114")}},
			&test3pb.TestAllTypes{OneofField: &test3pb.TestAllTypes_OneofBytes{[]byte("1114")}},
		},
		wire: pack.Message{pack.Tag{114, pack.BytesType}, pack.String("1114")}.Marshal(),
	},
	{
		desc: "oneof (bool)",
		decodeTo: []proto.Message{
			&testpb.TestAllTypes{OneofField: &testpb.TestAllTypes_OneofBool{true}},
			&test3pb.TestAllTypes{OneofField: &test3pb.TestAllTypes_OneofBool{true}},
		},
		wire: pack.Message{pack.Tag{115, pack.VarintType}, pack.Bool(true)}.Marshal(),
	},
	{
		desc: "oneof (uint64)",
		decodeTo: []proto.Message{
			&testpb.TestAllTypes{OneofField: &testpb.TestAllTypes_OneofUint64{116}},
			&test3pb.TestAllTypes{OneofField: &test3pb.TestAllTypes_OneofUint64{116}},
		},
		wire: pack.Message{pack.Tag{116, pack.VarintType}, pack.Varint(116)}.Marshal(),
	},
	{
		desc: "oneof (float)",
		decodeTo: []proto.Message{
			&testpb.TestAllTypes{OneofField: &testpb.TestAllTypes_OneofFloat{117.5}},
			&test3pb.TestAllTypes{OneofField: &test3pb.TestAllTypes_OneofFloat{117.5}},
		},
		wire: pack.Message{pack.Tag{117, pack.Fixed32Type}, pack.Float32(117.5)}.Marshal(),
	},
	{
		desc: "oneof (double)",
		decodeTo: []proto.Message{
			&testpb.TestAllTypes{OneofField: &testpb.TestAllTypes_OneofDouble{118.5}},
			&test3pb.TestAllTypes{OneofField: &test3pb.TestAllTypes_OneofDouble{118.5}},
		},
		wire: pack.Message{pack.Tag{118, pack.Fixed64Type}, pack.Float64(118.5)}.Marshal(),
	},
	{
		desc: "oneof (enum)",
		decodeTo: []proto.Message{
			&testpb.TestAllTypes{OneofField: &testpb.TestAllTypes_OneofEnum{testpb.TestAllTypes_BAR}},
			&test3pb.TestAllTypes{OneofField: &test3pb.TestAllTypes_OneofEnum{test3pb.TestAllTypes_BAR}},
		},
		wire: pack.Message{pack.Tag{119, pack.VarintType}, pack.Varint(int(testpb.TestAllTypes_BAR))}.Marshal(),
	},
	{
		desc: "oneof (zero)",
		decodeTo: []proto.Message{
			&testpb.TestAllTypes{OneofField: &testpb.TestAllTypes_OneofUint64{0}},
			&test3pb.TestAllTypes{OneofField: &test3pb.TestAllTypes_OneofUint64{0}},
		},
		wire: pack.Message{pack.Tag{116, pack.VarintType}, pack.Varint(0)}.Marshal(),
	},
	{
		desc: "oneof (overridden value)",
		decodeTo: []proto.Message{
			&testpb.TestAllTypes{OneofField: &testpb.TestAllTypes_OneofUint64{2}},
			&test3pb.TestAllTypes{OneofField: &test3pb.TestAllTypes_OneofUint64{2}},
		},
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
		decodeTo: []proto.Message{build(
			&testpb.TestAllTypes{},
			unknown(pack.Message{
				pack.Tag{100000, pack.VarintType}, pack.Varint(1),
			}.Marshal()),
		), build(
			&test3pb.TestAllTypes{},
			unknown(pack.Message{
				pack.Tag{100000, pack.VarintType}, pack.Varint(1),
			}.Marshal()),
		), build(
			&testpb.TestAllExtensions{},
			unknown(pack.Message{
				pack.Tag{100000, pack.VarintType}, pack.Varint(1),
			}.Marshal()),
		)},
		wire: pack.Message{
			pack.Tag{100000, pack.VarintType}, pack.Varint(1),
		}.Marshal(),
	},
	{
		desc: "discarded unknown fields",
		unmarshalOptions: proto.UnmarshalOptions{
			DiscardUnknown: true,
		},
		decodeTo: []proto.Message{
			&testpb.TestAllTypes{},
			&test3pb.TestAllTypes{},
		},
		wire: pack.Message{
			pack.Tag{100000, pack.VarintType}, pack.Varint(1),
		}.Marshal(),
	},
	{
		desc: "field type mismatch",
		decodeTo: []proto.Message{build(
			&testpb.TestAllTypes{},
			unknown(pack.Message{
				pack.Tag{1, pack.BytesType}, pack.String("string"),
			}.Marshal()),
		), build(
			&test3pb.TestAllTypes{},
			unknown(pack.Message{
				pack.Tag{1, pack.BytesType}, pack.String("string"),
			}.Marshal()),
		)},
		wire: pack.Message{
			pack.Tag{1, pack.BytesType}, pack.String("string"),
		}.Marshal(),
	},
	{
		desc: "map field element mismatch",
		decodeTo: []proto.Message{
			&testpb.TestAllTypes{
				MapInt32Int32: map[int32]int32{1: 0},
			}, &test3pb.TestAllTypes{
				MapInt32Int32: map[int32]int32{1: 0},
			},
		},
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
		decodeTo:      []proto.Message{&requiredpb.Int32{}},
	},
	{
		desc:          "required int32 set",
		checkFastInit: true,
		decodeTo: []proto.Message{&requiredpb.Int32{
			V: proto.Int32(1),
		}},
		wire: pack.Message{
			pack.Tag{1, pack.VarintType}, pack.Varint(1),
		}.Marshal(),
	},
	{
		desc:          "required fixed32 unset",
		checkFastInit: true,
		partial:       true,
		decodeTo:      []proto.Message{&requiredpb.Fixed32{}},
	},
	{
		desc:          "required fixed32 set",
		checkFastInit: true,
		decodeTo: []proto.Message{&requiredpb.Fixed32{
			V: proto.Uint32(1),
		}},
		wire: pack.Message{
			pack.Tag{1, pack.Fixed32Type}, pack.Int32(1),
		}.Marshal(),
	},
	{
		desc:          "required fixed64 unset",
		checkFastInit: true,
		partial:       true,
		decodeTo:      []proto.Message{&requiredpb.Fixed64{}},
	},
	{
		desc:          "required fixed64 set",
		checkFastInit: true,
		decodeTo: []proto.Message{&requiredpb.Fixed64{
			V: proto.Uint64(1),
		}},
		wire: pack.Message{
			pack.Tag{1, pack.Fixed64Type}, pack.Int64(1),
		}.Marshal(),
	},
	{
		desc:          "required bytes unset",
		checkFastInit: true,
		partial:       true,
		decodeTo:      []proto.Message{&requiredpb.Bytes{}},
	},
	{
		desc:          "required bytes set",
		checkFastInit: true,
		decodeTo: []proto.Message{&requiredpb.Bytes{
			V: []byte{},
		}},
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
		decodeTo: []proto.Message{&testpb.TestRequiredForeign{
			OptionalMessage: &testpb.TestRequired{},
		}},
		wire: pack.Message{
			pack.Tag{1, pack.BytesType}, pack.LengthPrefix(pack.Message{}),
		}.Marshal(),
	},
	{
		desc:          "required field in optional message set",
		checkFastInit: true,
		decodeTo: []proto.Message{&testpb.TestRequiredForeign{
			OptionalMessage: &testpb.TestRequired{
				RequiredField: proto.Int32(1),
			},
		}},
		wire: pack.Message{
			pack.Tag{1, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(1),
			}),
		}.Marshal(),
	},
	{
		desc:          "required field in optional message set (split across multiple tags)",
		checkFastInit: false, // fast init checks don't handle split messages
		decodeTo: []proto.Message{&testpb.TestRequiredForeign{
			OptionalMessage: &testpb.TestRequired{
				RequiredField: proto.Int32(1),
			},
		}},
		wire: pack.Message{
			pack.Tag{1, pack.BytesType}, pack.LengthPrefix(pack.Message{}),
			pack.Tag{1, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(1),
			}),
		}.Marshal(),
		validationStatus: impl.ValidationValidMaybeUninitalized,
	},
	{
		desc:          "required field in repeated message unset",
		checkFastInit: true,
		partial:       true,
		decodeTo: []proto.Message{&testpb.TestRequiredForeign{
			RepeatedMessage: []*testpb.TestRequired{
				{RequiredField: proto.Int32(1)},
				{},
			},
		}},
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
		decodeTo: []proto.Message{&testpb.TestRequiredForeign{
			RepeatedMessage: []*testpb.TestRequired{
				{RequiredField: proto.Int32(1)},
				{RequiredField: proto.Int32(2)},
			},
		}},
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
		decodeTo: []proto.Message{&testpb.TestRequiredForeign{
			MapMessage: map[int32]*testpb.TestRequired{
				1: {RequiredField: proto.Int32(1)},
				2: {},
			},
		}},
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
		decodeTo: []proto.Message{&testpb.TestRequiredForeign{
			MapMessage: map[int32]*testpb.TestRequired{
				2: {},
			},
		}},
		wire: pack.Message{
			pack.Tag{3, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(2),
			}),
		}.Marshal(),
	},
	{
		desc:          "required field in map message set",
		checkFastInit: true,
		decodeTo: []proto.Message{&testpb.TestRequiredForeign{
			MapMessage: map[int32]*testpb.TestRequired{
				1: {RequiredField: proto.Int32(1)},
				2: {RequiredField: proto.Int32(2)},
			},
		}},
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
		decodeTo: []proto.Message{&testpb.TestRequiredGroupFields{
			Optionalgroup: &testpb.TestRequiredGroupFields_OptionalGroup{},
		}},
		wire: pack.Message{
			pack.Tag{1, pack.StartGroupType},
			pack.Tag{1, pack.EndGroupType},
		}.Marshal(),
	},
	{
		desc:          "required field in optional group set",
		checkFastInit: true,
		decodeTo: []proto.Message{&testpb.TestRequiredGroupFields{
			Optionalgroup: &testpb.TestRequiredGroupFields_OptionalGroup{
				A: proto.Int32(1),
			},
		}},
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
		decodeTo: []proto.Message{&testpb.TestRequiredGroupFields{
			Repeatedgroup: []*testpb.TestRequiredGroupFields_RepeatedGroup{
				{A: proto.Int32(1)},
				{},
			},
		}},
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
		decodeTo: []proto.Message{&testpb.TestRequiredGroupFields{
			Repeatedgroup: []*testpb.TestRequiredGroupFields_RepeatedGroup{
				{A: proto.Int32(1)},
				{A: proto.Int32(2)},
			},
		}},
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
		decodeTo: []proto.Message{
			&testpb.TestRequiredForeign{OneofField: &testpb.TestRequiredForeign_OneofMessage{
				&testpb.TestRequired{},
			}},
		},
		wire: pack.Message{pack.Tag{4, pack.BytesType}, pack.LengthPrefix(pack.Message{})}.Marshal(),
	},
	{
		desc:          "required field in oneof message set",
		checkFastInit: true,
		decodeTo: []proto.Message{
			&testpb.TestRequiredForeign{OneofField: &testpb.TestRequiredForeign_OneofMessage{
				&testpb.TestRequired{
					RequiredField: proto.Int32(1),
				},
			}},
		},
		wire: pack.Message{pack.Tag{4, pack.BytesType}, pack.LengthPrefix(pack.Message{
			pack.Tag{1, pack.VarintType}, pack.Varint(1),
		})}.Marshal(),
	},
	{
		desc:          "required field in extension message unset",
		checkFastInit: true,
		partial:       true,
		decodeTo: []proto.Message{build(
			&testpb.TestAllExtensions{},
			extend(testpb.E_TestRequired_Single, &testpb.TestRequired{}),
		)},
		wire: pack.Message{
			pack.Tag{1000, pack.BytesType}, pack.LengthPrefix(pack.Message{}),
		}.Marshal(),
	},
	{
		desc:          "required field in extension message set",
		checkFastInit: true,
		decodeTo: []proto.Message{build(
			&testpb.TestAllExtensions{},
			extend(testpb.E_TestRequired_Single, &testpb.TestRequired{
				RequiredField: proto.Int32(1),
			}),
		)},
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
		decodeTo: []proto.Message{build(
			&testpb.TestAllExtensions{},
			extend(testpb.E_TestRequired_Multi, []*testpb.TestRequired{
				{RequiredField: proto.Int32(1)},
				{},
			}),
		)},
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
		decodeTo: []proto.Message{build(
			&testpb.TestAllExtensions{},
			extend(testpb.E_TestRequired_Multi, []*testpb.TestRequired{
				{RequiredField: proto.Int32(1)},
				{RequiredField: proto.Int32(2)},
			}),
		)},
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
		decodeTo: []proto.Message{
			&legacypb.Legacy{
				F1: &legacy1pb.Message{
					OptionalInt32:     proto.Int32(1),
					OptionalChildEnum: legacy1pb.Message_ALPHA.Enum(),
					OptionalChildMessage: &legacy1pb.Message_ChildMessage{
						F1: proto.String("x"),
					},
					Optionalgroup: &legacy1pb.Message_OptionalGroup{
						F1: proto.String("x"),
					},
					RepeatedChildMessage: []*legacy1pb.Message_ChildMessage{
						{F1: proto.String("x")},
					},
					Repeatedgroup: []*legacy1pb.Message_RepeatedGroup{
						{F1: proto.String("x")},
					},
					MapBoolChildMessage: map[bool]*legacy1pb.Message_ChildMessage{
						true: {F1: proto.String("x")},
					},
					OneofUnion: &legacy1pb.Message_OneofChildMessage{
						&legacy1pb.Message_ChildMessage{
							F1: proto.String("x"),
						},
					},
				},
			},
		},
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
		decodeTo: []proto.Message{build(
			&testpb.TestAllTypes{},
			unknown(pack.Message{
				pack.Tag{pack.FirstReservedNumber, pack.VarintType}, pack.Varint(1004),
			}.Marshal()),
		)},
		wire: pack.Message{
			pack.Tag{pack.FirstReservedNumber, pack.VarintType}, pack.Varint(1004),
		}.Marshal(),
	},
	{
		desc: "last reserved field number",
		decodeTo: []proto.Message{build(
			&testpb.TestAllTypes{},
			unknown(pack.Message{
				pack.Tag{pack.LastReservedNumber, pack.VarintType}, pack.Varint(1005),
			}.Marshal()),
		)},
		wire: pack.Message{
			pack.Tag{pack.LastReservedNumber, pack.VarintType}, pack.Varint(1005),
		}.Marshal(),
	},
	{
		desc: "nested unknown extension",
		unmarshalOptions: proto.UnmarshalOptions{
			DiscardUnknown: true,
			Resolver: func() protoregistry.ExtensionTypeResolver {
				types := &protoregistry.Types{}
				types.RegisterExtension(testpb.E_OptionalNestedMessageExtension)
				types.RegisterExtension(testpb.E_OptionalInt32Extension)
				return types
			}(),
		},
		decodeTo: []proto.Message{func() proto.Message {
			m := &testpb.TestAllExtensions{}
			if err := prototext.Unmarshal([]byte(`
				[goproto.proto.test.optional_nested_message_extension]: {
					corecursive: {
						[goproto.proto.test.optional_nested_message_extension]: {
							corecursive: {
								[goproto.proto.test.optional_int32_extension]: 42
							}
						}
					}
				}`), m); err != nil {
				panic(err)
			}
			return m
		}()},
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
		decodeTo: []proto.Message{&test3pb.TestAllTypes{
			OptionalString: "abc\xff",
		}},
		wire: pack.Message{
			pack.Tag{14, pack.BytesType}, pack.String("abc\xff"),
		}.Marshal(),
	},
	{
		desc: "invalid UTF-8 in repeated string field",
		decodeTo: []proto.Message{&test3pb.TestAllTypes{
			RepeatedString: []string{"foo", "abc\xff"},
		}},
		wire: pack.Message{
			pack.Tag{44, pack.BytesType}, pack.String("foo"),
			pack.Tag{44, pack.BytesType}, pack.String("abc\xff"),
		}.Marshal(),
	},
	{
		desc: "invalid UTF-8 in nested message",
		decodeTo: []proto.Message{&test3pb.TestAllTypes{
			OptionalNestedMessage: &test3pb.TestAllTypes_NestedMessage{
				Corecursive: &test3pb.TestAllTypes{
					OptionalString: "abc\xff",
				},
			},
		}},
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
		decodeTo: []proto.Message{
			&test3pb.TestAllTypes{OneofField: &test3pb.TestAllTypes_OneofString{"abc\xff"}},
		},
		wire: pack.Message{pack.Tag{113, pack.BytesType}, pack.String("abc\xff")}.Marshal(),
	},
	{
		desc: "invalid UTF-8 in map key",
		decodeTo: []proto.Message{&test3pb.TestAllTypes{
			MapStringString: map[string]string{"key\xff": "val"},
		}},
		wire: pack.Message{
			pack.Tag{69, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.BytesType}, pack.String("key\xff"),
				pack.Tag{2, pack.BytesType}, pack.String("val"),
			}),
		}.Marshal(),
	},
	{
		desc: "invalid UTF-8 in map value",
		decodeTo: []proto.Message{&test3pb.TestAllTypes{
			MapStringString: map[string]string{"key": "val\xff"},
		}},
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
}
