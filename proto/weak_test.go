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

func TestWeak(t *testing.T) {
	if !flags.ProtoLegacy {
		t.SkipNow()
	}

	m := new(testpb.TestWeak)
	b := pack.Message{
		pack.Tag{1, pack.BytesType}, pack.LengthPrefix(pack.Message{
			pack.Tag{1, pack.VarintType}, pack.Varint(1000),
		}),
		pack.Tag{2, pack.BytesType}, pack.LengthPrefix(pack.Message{
			pack.Tag{1, pack.VarintType}, pack.Varint(2000),
		}),
	}.Marshal()
	if err := proto.Unmarshal(b, m); err != nil {
		t.Errorf("Unmarshal error: %v", err)
	}

	mw := m.GetWeakMessage1().(*weakpb.WeakImportMessage1)
	if mw.GetA() != 1000 {
		t.Errorf("m.WeakMessage1.a = %d, want %d", mw.GetA(), 1000)
	}

	if len(m.ProtoReflect().GetUnknown()) == 0 {
		t.Errorf("m has no unknown fields, expected at least something")
	}

	if n := proto.Size(m); n != len(b) {
		t.Errorf("Size() = %d, want %d", n, len(b))
	}

	b2, err := proto.Marshal(m)
	if err != nil {
		t.Errorf("Marshal error: %v", err)
	}
	if len(b2) != len(b) {
		t.Errorf("len(Marshal) = %d, want %d", len(b2), len(b))
	}
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
