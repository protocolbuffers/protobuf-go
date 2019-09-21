// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto_test

import (
	"reflect"
	"sync"
	"testing"

	"google.golang.org/protobuf/internal/encoding/pack"
	"google.golang.org/protobuf/proto"

	testpb "google.golang.org/protobuf/internal/testprotos/test"
	test3pb "google.golang.org/protobuf/internal/testprotos/test3"
)

func TestMerge(t *testing.T) {

	t.Run("Deep", func(t *testing.T) { testMerge(t, false) })
	t.Run("Shallow", func(t *testing.T) { testMerge(t, true) })
}

func testMerge(t *testing.T, shallow bool) {
	tests := []struct {
		desc string
		dst  proto.Message
		src  proto.Message
		want proto.Message

		// If provided, mutator is run on src after merging.
		// It reports whether a mutation is expected to be observable in dst
		// if Shallow is enabled.
		mutator func(proto.Message) bool
	}{{
		desc: "merge from nil message",
		dst:  new(testpb.TestAllTypes),
		src:  (*testpb.TestAllTypes)(nil),
		want: new(testpb.TestAllTypes),
	}, {
		desc: "clone a large message",
		dst:  new(testpb.TestAllTypes),
		src: &testpb.TestAllTypes{
			OptionalInt64:      proto.Int64(0),
			OptionalNestedEnum: testpb.TestAllTypes_NestedEnum(1).Enum(),
			OptionalNestedMessage: &testpb.TestAllTypes_NestedMessage{
				A: proto.Int32(100),
			},
			RepeatedSfixed32: []int32{1, 2, 3},
			RepeatedNestedMessage: []*testpb.TestAllTypes_NestedMessage{
				{A: proto.Int32(200)},
				{A: proto.Int32(300)},
			},
			MapStringNestedEnum: map[string]testpb.TestAllTypes_NestedEnum{
				"fizz": 400,
				"buzz": 500,
			},
			MapStringNestedMessage: map[string]*testpb.TestAllTypes_NestedMessage{
				"foo": {A: proto.Int32(600)},
				"bar": {A: proto.Int32(700)},
			},
			OneofField: &testpb.TestAllTypes_OneofNestedMessage{
				&testpb.TestAllTypes_NestedMessage{
					A: proto.Int32(800),
				},
			},
		},
		want: &testpb.TestAllTypes{
			OptionalInt64:      proto.Int64(0),
			OptionalNestedEnum: testpb.TestAllTypes_NestedEnum(1).Enum(),
			OptionalNestedMessage: &testpb.TestAllTypes_NestedMessage{
				A: proto.Int32(100),
			},
			RepeatedSfixed32: []int32{1, 2, 3},
			RepeatedNestedMessage: []*testpb.TestAllTypes_NestedMessage{
				{A: proto.Int32(200)},
				{A: proto.Int32(300)},
			},
			MapStringNestedEnum: map[string]testpb.TestAllTypes_NestedEnum{
				"fizz": 400,
				"buzz": 500,
			},
			MapStringNestedMessage: map[string]*testpb.TestAllTypes_NestedMessage{
				"foo": {A: proto.Int32(600)},
				"bar": {A: proto.Int32(700)},
			},
			OneofField: &testpb.TestAllTypes_OneofNestedMessage{
				&testpb.TestAllTypes_NestedMessage{
					A: proto.Int32(800),
				},
			},
		},
		mutator: func(mi proto.Message) bool {
			m := mi.(*testpb.TestAllTypes)
			*m.OptionalInt64++
			*m.OptionalNestedEnum++
			*m.OptionalNestedMessage.A++
			m.RepeatedSfixed32[0]++
			*m.RepeatedNestedMessage[0].A++
			delete(m.MapStringNestedEnum, "fizz")
			*m.MapStringNestedMessage["foo"].A++
			*m.OneofField.(*testpb.TestAllTypes_OneofNestedMessage).OneofNestedMessage.A++
			return true
		},
	}, {
		desc: "merge bytes",
		dst: &testpb.TestAllTypes{
			OptionalBytes:  []byte{1, 2, 3},
			RepeatedBytes:  [][]byte{{1, 2}, {3, 4}},
			MapStringBytes: map[string][]byte{"alpha": {1, 2, 3}},
		},
		src: &testpb.TestAllTypes{
			OptionalBytes:  []byte{4, 5, 6},
			RepeatedBytes:  [][]byte{{5, 6}, {7, 8}},
			MapStringBytes: map[string][]byte{"alpha": {4, 5, 6}, "bravo": {1, 2, 3}},
		},
		want: &testpb.TestAllTypes{
			OptionalBytes:  []byte{4, 5, 6},
			RepeatedBytes:  [][]byte{{1, 2}, {3, 4}, {5, 6}, {7, 8}},
			MapStringBytes: map[string][]byte{"alpha": {4, 5, 6}, "bravo": {1, 2, 3}},
		},
		mutator: func(mi proto.Message) bool {
			m := mi.(*testpb.TestAllTypes)
			m.OptionalBytes[0]++
			m.RepeatedBytes[0][0]++
			m.MapStringBytes["alpha"][0]++
			return true
		},
	}, {
		desc: "merge singular fields",
		dst: &testpb.TestAllTypes{
			OptionalInt32:      proto.Int32(1),
			OptionalInt64:      proto.Int64(1),
			OptionalNestedEnum: testpb.TestAllTypes_NestedEnum(10).Enum(),
			OptionalNestedMessage: &testpb.TestAllTypes_NestedMessage{
				A: proto.Int32(100),
				Corecursive: &testpb.TestAllTypes{
					OptionalInt64: proto.Int64(1000),
				},
			},
		},
		src: &testpb.TestAllTypes{
			OptionalInt64:      proto.Int64(2),
			OptionalNestedEnum: testpb.TestAllTypes_NestedEnum(20).Enum(),
			OptionalNestedMessage: &testpb.TestAllTypes_NestedMessage{
				A: proto.Int32(200),
			},
		},
		want: &testpb.TestAllTypes{
			OptionalInt32:      proto.Int32(1),
			OptionalInt64:      proto.Int64(2),
			OptionalNestedEnum: testpb.TestAllTypes_NestedEnum(20).Enum(),
			OptionalNestedMessage: &testpb.TestAllTypes_NestedMessage{
				A: proto.Int32(200),
				Corecursive: &testpb.TestAllTypes{
					OptionalInt64: proto.Int64(1000),
				},
			},
		},
		mutator: func(mi proto.Message) bool {
			m := mi.(*testpb.TestAllTypes)
			*m.OptionalInt64++
			*m.OptionalNestedEnum++
			*m.OptionalNestedMessage.A++
			return false // scalar mutations are not observable in shallow copy
		},
	}, {
		desc: "merge list fields",
		dst: &testpb.TestAllTypes{
			RepeatedSfixed32: []int32{1, 2, 3},
			RepeatedNestedMessage: []*testpb.TestAllTypes_NestedMessage{
				{A: proto.Int32(100)},
				{A: proto.Int32(200)},
			},
		},
		src: &testpb.TestAllTypes{
			RepeatedSfixed32: []int32{4, 5, 6},
			RepeatedNestedMessage: []*testpb.TestAllTypes_NestedMessage{
				{A: proto.Int32(300)},
				{A: proto.Int32(400)},
			},
		},
		want: &testpb.TestAllTypes{
			RepeatedSfixed32: []int32{1, 2, 3, 4, 5, 6},
			RepeatedNestedMessage: []*testpb.TestAllTypes_NestedMessage{
				{A: proto.Int32(100)},
				{A: proto.Int32(200)},
				{A: proto.Int32(300)},
				{A: proto.Int32(400)},
			},
		},
		mutator: func(mi proto.Message) bool {
			m := mi.(*testpb.TestAllTypes)
			m.RepeatedSfixed32[0]++
			*m.RepeatedNestedMessage[0].A++
			return true
		},
	}, {
		desc: "merge map fields",
		dst: &testpb.TestAllTypes{
			MapStringNestedEnum: map[string]testpb.TestAllTypes_NestedEnum{
				"fizz": 100,
				"buzz": 200,
				"guzz": 300,
			},
			MapStringNestedMessage: map[string]*testpb.TestAllTypes_NestedMessage{
				"foo": {A: proto.Int32(400)},
			},
		},
		src: &testpb.TestAllTypes{
			MapStringNestedEnum: map[string]testpb.TestAllTypes_NestedEnum{
				"fizz": 1000,
				"buzz": 2000,
			},
			MapStringNestedMessage: map[string]*testpb.TestAllTypes_NestedMessage{
				"foo": {A: proto.Int32(3000)},
				"bar": {},
			},
		},
		want: &testpb.TestAllTypes{
			MapStringNestedEnum: map[string]testpb.TestAllTypes_NestedEnum{
				"fizz": 1000,
				"buzz": 2000,
				"guzz": 300,
			},
			MapStringNestedMessage: map[string]*testpb.TestAllTypes_NestedMessage{
				"foo": {A: proto.Int32(3000)},
				"bar": {},
			},
		},
		mutator: func(mi proto.Message) bool {
			m := mi.(*testpb.TestAllTypes)
			delete(m.MapStringNestedEnum, "fizz")
			m.MapStringNestedMessage["bar"].A = proto.Int32(1)
			return true
		},
	}, {
		desc: "merge oneof message fields",
		dst: &testpb.TestAllTypes{
			OneofField: &testpb.TestAllTypes_OneofNestedMessage{
				&testpb.TestAllTypes_NestedMessage{
					A: proto.Int32(100),
				},
			},
		},
		src: &testpb.TestAllTypes{
			OneofField: &testpb.TestAllTypes_OneofNestedMessage{
				&testpb.TestAllTypes_NestedMessage{
					Corecursive: &testpb.TestAllTypes{
						OptionalInt64: proto.Int64(1000),
					},
				},
			},
		},
		want: &testpb.TestAllTypes{
			OneofField: &testpb.TestAllTypes_OneofNestedMessage{
				&testpb.TestAllTypes_NestedMessage{
					A: proto.Int32(100),
					Corecursive: &testpb.TestAllTypes{
						OptionalInt64: proto.Int64(1000),
					},
				},
			},
		},
		mutator: func(mi proto.Message) bool {
			m := mi.(*testpb.TestAllTypes)
			*m.OneofField.(*testpb.TestAllTypes_OneofNestedMessage).OneofNestedMessage.Corecursive.OptionalInt64++
			return true
		},
	}, {
		desc: "merge oneof scalar fields",
		dst: &testpb.TestAllTypes{
			OneofField: &testpb.TestAllTypes_OneofUint32{100},
		},
		src: &testpb.TestAllTypes{
			OneofField: &testpb.TestAllTypes_OneofFloat{3.14152},
		},
		want: &testpb.TestAllTypes{
			OneofField: &testpb.TestAllTypes_OneofFloat{3.14152},
		},
		mutator: func(mi proto.Message) bool {
			m := mi.(*testpb.TestAllTypes)
			m.OneofField.(*testpb.TestAllTypes_OneofFloat).OneofFloat++
			return false // scalar mutations are not observable in shallow copy
		},
	}, {
		desc: "merge extension fields",
		dst: func() proto.Message {
			m := new(testpb.TestAllExtensions)
			proto.SetExtension(m, testpb.E_OptionalInt32, int32(32))
			proto.SetExtension(m, testpb.E_OptionalNestedMessage,
				&testpb.TestAllExtensions_NestedMessage{
					A: proto.Int32(50),
				},
			)
			proto.SetExtension(m, testpb.E_RepeatedFixed32, []uint32{1, 2, 3})
			return m
		}(),
		src: func() proto.Message {
			m2 := new(testpb.TestAllExtensions)
			proto.SetExtension(m2, testpb.E_OptionalInt64, int64(1000))
			m := new(testpb.TestAllExtensions)
			proto.SetExtension(m, testpb.E_OptionalInt64, int64(64))
			proto.SetExtension(m, testpb.E_OptionalNestedMessage,
				&testpb.TestAllExtensions_NestedMessage{
					Corecursive: m2,
				},
			)
			proto.SetExtension(m, testpb.E_RepeatedFixed32, []uint32{4, 5, 6})
			return m
		}(),
		want: func() proto.Message {
			m2 := new(testpb.TestAllExtensions)
			proto.SetExtension(m2, testpb.E_OptionalInt64, int64(1000))
			m := new(testpb.TestAllExtensions)
			proto.SetExtension(m, testpb.E_OptionalInt32, int32(32))
			proto.SetExtension(m, testpb.E_OptionalInt64, int64(64))
			proto.SetExtension(m, testpb.E_OptionalNestedMessage,
				&testpb.TestAllExtensions_NestedMessage{
					A:           proto.Int32(50),
					Corecursive: m2,
				},
			)
			proto.SetExtension(m, testpb.E_RepeatedFixed32, []uint32{1, 2, 3, 4, 5, 6})
			return m
		}(),
	}, {
		desc: "merge unknown fields",
		dst: func() proto.Message {
			m := new(testpb.TestAllTypes)
			m.ProtoReflect().SetUnknown(pack.Message{
				pack.Tag{Number: 50000, Type: pack.VarintType}, pack.Svarint(-5),
			}.Marshal())
			return m
		}(),
		src: func() proto.Message {
			m := new(testpb.TestAllTypes)
			m.ProtoReflect().SetUnknown(pack.Message{
				pack.Tag{Number: 500000, Type: pack.VarintType}, pack.Svarint(-50),
			}.Marshal())
			return m
		}(),
		want: func() proto.Message {
			m := new(testpb.TestAllTypes)
			m.ProtoReflect().SetUnknown(pack.Message{
				pack.Tag{Number: 50000, Type: pack.VarintType}, pack.Svarint(-5),
				pack.Tag{Number: 500000, Type: pack.VarintType}, pack.Svarint(-50),
			}.Marshal())
			return m
		}(),
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			// Merge should be semantically equivalent to unmarshaling the
			// encoded form of src into the current dst.
			b1, err := proto.MarshalOptions{AllowPartial: true}.Marshal(tt.dst)
			if err != nil {
				t.Fatalf("Marshal(dst) error: %v", err)
			}
			b2, err := proto.MarshalOptions{AllowPartial: true}.Marshal(tt.src)
			if err != nil {
				t.Fatalf("Marshal(src) error: %v", err)
			}
			dst := tt.dst.ProtoReflect().New().Interface()
			err = proto.UnmarshalOptions{AllowPartial: true}.Unmarshal(append(b1, b2...), dst)
			if err != nil {
				t.Fatalf("Unmarshal() error: %v", err)
			}
			if !proto.Equal(dst, tt.want) {
				t.Fatalf("Unmarshal(Marshal(dst)+Marshal(src)) mismatch: got %v, want %v", dst, tt.want)
			}

			proto.MergeOptions{Shallow: shallow}.Merge(tt.dst, tt.src)
			if !proto.Equal(tt.dst, tt.want) {
				t.Fatalf("Merge() mismatch:\n got %v\nwant %v", tt.dst, tt.want)
			}
			if tt.mutator != nil {
				wantObservable := tt.mutator(tt.src) && shallow
				gotObservable := !proto.Equal(tt.dst, tt.want)
				if gotObservable != wantObservable {
					t.Fatalf("mutation observed:\n got %v\nwant %v", gotObservable, wantObservable)
				}
			}
		})
	}
}

// TestMergeAberrant tests inputs that are beyond the protobuf data model.
// Just because there is a test for the current behavior does not mean that
// this will behave the same way in the future.
func TestMergeAberrant(t *testing.T) {
	tests := []struct {
		label string
		dst   proto.Message
		src   proto.Message
		check func(proto.Message) bool
	}{{
		label: "Proto2EmptyBytes",
		dst:   &testpb.TestAllTypes{OptionalBytes: nil},
		src:   &testpb.TestAllTypes{OptionalBytes: []byte{}},
		check: func(m proto.Message) bool {
			return m.(*testpb.TestAllTypes).OptionalBytes != nil
		},
	}, {
		label: "Proto3EmptyBytes",
		dst:   &test3pb.TestAllTypes{OptionalBytes: nil},
		src:   &test3pb.TestAllTypes{OptionalBytes: []byte{}},
		check: func(m proto.Message) bool {
			return m.(*test3pb.TestAllTypes).OptionalBytes == nil
		},
	}, {
		label: "EmptyList",
		dst:   &testpb.TestAllTypes{RepeatedInt32: nil},
		src:   &testpb.TestAllTypes{RepeatedInt32: []int32{}},
		check: func(m proto.Message) bool {
			return m.(*testpb.TestAllTypes).RepeatedInt32 == nil
		},
	}, {
		label: "ListWithNilBytes",
		dst:   &testpb.TestAllTypes{RepeatedBytes: nil},
		src:   &testpb.TestAllTypes{RepeatedBytes: [][]byte{nil}},
		check: func(m proto.Message) bool {
			return reflect.DeepEqual(m.(*testpb.TestAllTypes).RepeatedBytes, [][]byte{{}})
		},
	}, {
		label: "ListWithEmptyBytes",
		dst:   &testpb.TestAllTypes{RepeatedBytes: nil},
		src:   &testpb.TestAllTypes{RepeatedBytes: [][]byte{{}}},
		check: func(m proto.Message) bool {
			return reflect.DeepEqual(m.(*testpb.TestAllTypes).RepeatedBytes, [][]byte{{}})
		},
	}, {
		label: "ListWithNilMessage",
		dst:   &testpb.TestAllTypes{RepeatedNestedMessage: nil},
		src:   &testpb.TestAllTypes{RepeatedNestedMessage: []*testpb.TestAllTypes_NestedMessage{nil}},
		check: func(m proto.Message) bool {
			return m.(*testpb.TestAllTypes).RepeatedNestedMessage[0] != nil
		},
	}, {
		label: "EmptyMap",
		dst:   &testpb.TestAllTypes{MapStringString: nil},
		src:   &testpb.TestAllTypes{MapStringString: map[string]string{}},
		check: func(m proto.Message) bool {
			return m.(*testpb.TestAllTypes).MapStringString == nil
		},
	}, {
		label: "MapWithNilBytes",
		dst:   &testpb.TestAllTypes{MapStringBytes: nil},
		src:   &testpb.TestAllTypes{MapStringBytes: map[string][]byte{"k": nil}},
		check: func(m proto.Message) bool {
			return reflect.DeepEqual(m.(*testpb.TestAllTypes).MapStringBytes, map[string][]byte{"k": {}})
		},
	}, {
		label: "MapWithEmptyBytes",
		dst:   &testpb.TestAllTypes{MapStringBytes: nil},
		src:   &testpb.TestAllTypes{MapStringBytes: map[string][]byte{"k": {}}},
		check: func(m proto.Message) bool {
			return reflect.DeepEqual(m.(*testpb.TestAllTypes).MapStringBytes, map[string][]byte{"k": {}})
		},
	}, {
		label: "MapWithNilMessage",
		dst:   &testpb.TestAllTypes{MapStringNestedMessage: nil},
		src:   &testpb.TestAllTypes{MapStringNestedMessage: map[string]*testpb.TestAllTypes_NestedMessage{"k": nil}},
		check: func(m proto.Message) bool {
			return m.(*testpb.TestAllTypes).MapStringNestedMessage["k"] != nil
		},
	}, {
		label: "OneofWithTypedNilWrapper",
		dst:   &testpb.TestAllTypes{OneofField: nil},
		src:   &testpb.TestAllTypes{OneofField: (*testpb.TestAllTypes_OneofNestedMessage)(nil)},
		check: func(m proto.Message) bool {
			return m.(*testpb.TestAllTypes).OneofField == nil
		},
	}, {
		label: "OneofWithNilMessage",
		dst:   &testpb.TestAllTypes{OneofField: nil},
		src:   &testpb.TestAllTypes{OneofField: &testpb.TestAllTypes_OneofNestedMessage{OneofNestedMessage: nil}},
		check: func(m proto.Message) bool {
			return m.(*testpb.TestAllTypes).OneofField.(*testpb.TestAllTypes_OneofNestedMessage).OneofNestedMessage != nil
		},
		// TODO: extension, nil message
		// TODO: repeated extension, nil
		// TODO: extension bytes
		// TODO: repeated extension, nil message
	}}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			var pass bool
			func() {
				defer func() { recover() }()
				proto.Merge(tt.dst, tt.src)
				pass = tt.check(tt.dst)
			}()
			if !pass {
				t.Error("check failed")
			}
		})
	}
}

func TestMergeRace(t *testing.T) {
	dst := new(testpb.TestAllTypes)
	srcs := []*testpb.TestAllTypes{
		{OptionalInt32: proto.Int32(1)},
		{OptionalString: proto.String("hello")},
		{RepeatedInt32: []int32{2, 3, 4}},
		{RepeatedString: []string{"goodbye"}},
		{MapStringString: map[string]string{"key": "value"}},
		{OptionalNestedMessage: &testpb.TestAllTypes_NestedMessage{
			A: proto.Int32(5),
		}},
		func() *testpb.TestAllTypes {
			m := new(testpb.TestAllTypes)
			m.ProtoReflect().SetUnknown(pack.Message{
				pack.Tag{Number: 50000, Type: pack.VarintType}, pack.Svarint(-5),
			}.Marshal())
			return m
		}(),
	}

	// It should be safe to concurrently merge non-overlapping fields.
	var wg sync.WaitGroup
	defer wg.Wait()
	for _, src := range srcs {
		wg.Add(1)
		go func(src proto.Message) {
			defer wg.Done()
			proto.Merge(dst, src)
		}(src)
	}
}

func TestMergeSelf(t *testing.T) {
	got := &testpb.TestAllTypes{
		OptionalInt32:   proto.Int32(1),
		OptionalString:  proto.String("hello"),
		RepeatedInt32:   []int32{2, 3, 4},
		RepeatedString:  []string{"goodbye"},
		MapStringString: map[string]string{"key": "value"},
		OptionalNestedMessage: &testpb.TestAllTypes_NestedMessage{
			A: proto.Int32(5),
		},
	}
	got.ProtoReflect().SetUnknown(pack.Message{
		pack.Tag{Number: 50000, Type: pack.VarintType}, pack.Svarint(-5),
	}.Marshal())
	proto.Merge(got, got)

	// The main impact of merging to self is that repeated fields and
	// unknown fields are doubled.
	want := &testpb.TestAllTypes{
		OptionalInt32:   proto.Int32(1),
		OptionalString:  proto.String("hello"),
		RepeatedInt32:   []int32{2, 3, 4, 2, 3, 4},
		RepeatedString:  []string{"goodbye", "goodbye"},
		MapStringString: map[string]string{"key": "value"},
		OptionalNestedMessage: &testpb.TestAllTypes_NestedMessage{
			A: proto.Int32(5),
		},
	}
	want.ProtoReflect().SetUnknown(pack.Message{
		pack.Tag{Number: 50000, Type: pack.VarintType}, pack.Svarint(-5),
		pack.Tag{Number: 50000, Type: pack.VarintType}, pack.Svarint(-5),
	}.Marshal())

	if !proto.Equal(got, want) {
		t.Errorf("Equal mismatch:\ngot  %v\nwant %v", got, want)
	}
}
