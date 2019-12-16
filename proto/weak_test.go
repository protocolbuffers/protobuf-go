// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style.
// license that can be found in the LICENSE file.

package proto_test

import (
	"testing"

	"google.golang.org/protobuf/internal/encoding/pack"
	"google.golang.org/protobuf/internal/flags"
	"google.golang.org/protobuf/proto"

	testpb "google.golang.org/protobuf/internal/testprotos/test"
	weakpb "google.golang.org/protobuf/internal/testprotos/test/weak1"
)

func init() {
	if flags.ProtoLegacy {
		testValidMessages = append(testValidMessages, testWeakValidMessages...)
		testInvalidMessages = append(testInvalidMessages, testWeakInvalidMessages...)
	}
}

var testWeakValidMessages = []testProto{
	{
		desc: "weak message",
		decodeTo: []proto.Message{
			func() proto.Message {
				if !flags.ProtoLegacy {
					return nil
				}
				m := &testpb.TestWeak{}
				m.SetWeakMessage1(&weakpb.WeakImportMessage1{
					A: proto.Int32(1000),
				})
				m.ProtoReflect().SetUnknown(pack.Message{
					pack.Tag{2, pack.BytesType}, pack.LengthPrefix(pack.Message{
						pack.Tag{1, pack.VarintType}, pack.Varint(2000),
					}),
				}.Marshal())
				return m
			}(),
		},
		wire: pack.Message{
			pack.Tag{1, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(1000),
			}),
			pack.Tag{2, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(2000),
			}),
		}.Marshal(),
	},
}

var testWeakInvalidMessages = []testProto{
	{
		desc:     "invalid field number 0 in weak message",
		decodeTo: []proto.Message{(*testpb.TestWeak)(nil)},
		wire: pack.Message{
			pack.Tag{1, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{0, pack.VarintType}, pack.Varint(1000),
			}),
		}.Marshal(),
	},
}

func TestWeakNil(t *testing.T) {
	if !flags.ProtoLegacy {
		t.SkipNow()
	}

	m := new(testpb.TestWeak)
	if v, ok := m.GetWeakMessage1().(*weakpb.WeakImportMessage1); !ok || v != nil {
		t.Errorf("m.GetWeakMessage1() = type %[1]T(%[1]v), want (*weakpb.WeakImportMessage1)", v)
	}
}

func TestWeakMarshalNil(t *testing.T) {
	if !flags.ProtoLegacy {
		t.SkipNow()
	}

	m := new(testpb.TestWeak)
	m.SetWeakMessage1(nil)
	if b, err := proto.Marshal(m); err != nil || len(b) != 0 {
		t.Errorf("Marshal(weak field set to nil) = [%x], %v; want [], nil", b, err)
	}
	m.SetWeakMessage1((*weakpb.WeakImportMessage1)(nil))
	if b, err := proto.Marshal(m); err != nil || len(b) != 0 {
		t.Errorf("Marshal(weak field set to typed nil) = [%x], %v; want [], nil", b, err)
	}
}
