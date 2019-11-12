// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The protoreflect tag disables fast-path methods, including legacy ones.
// +build !protoreflect

package proto_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"google.golang.org/protobuf/internal/impl"
	"google.golang.org/protobuf/proto"
)

type selfMarshaler struct {
	bytes []byte
	err   error
}

func (m selfMarshaler) Reset()        {}
func (m selfMarshaler) ProtoMessage() {}

func (m selfMarshaler) String() string {
	return fmt.Sprintf("selfMarshaler{bytes:%v, err:%v}", m.bytes, m.err)
}

func (m selfMarshaler) Marshal() ([]byte, error) {
	return m.bytes, m.err
}

func (m *selfMarshaler) Unmarshal(b []byte) error {
	m.bytes = b
	return m.err
}

func TestLegacyMarshalMethod(t *testing.T) {
	for _, test := range []selfMarshaler{
		{bytes: []byte("marshal")},
		{bytes: []byte("marshal"), err: errors.New("some error")},
	} {
		m := impl.Export{}.MessageOf(test).Interface()
		b, err := proto.Marshal(m)
		if err != test.err || !bytes.Equal(b, test.bytes) {
			t.Errorf("proto.Marshal(%v) = %v, %v; want %v, %v", test, b, err, test.bytes, test.err)
		}
		if gotSize, wantSize := proto.Size(m), len(test.bytes); gotSize != wantSize {
			t.Fatalf("proto.Size(%v) = %v, want %v", test, gotSize, wantSize)
		}

		prefix := []byte("prefix")
		want := append(prefix, test.bytes...)
		b, err = proto.MarshalOptions{}.MarshalAppend(prefix, m)
		if err != test.err || !bytes.Equal(b, want) {
			t.Errorf("MarshalAppend(%v, %v) = %v, %v; want %v, %v", prefix, test, b, err, test.bytes, test.err)
		}

		b, err = proto.MarshalOptions{
			Deterministic: true,
		}.MarshalAppend(nil, m)
		if err != test.err || !bytes.Equal(b, test.bytes) {
			t.Errorf("MarshalOptions{Deterministic:true}.MarshalAppend(nil, %v) = %v, %v; want %v, %v", test, b, err, test.bytes, test.err)
		}
	}
}

func TestLegacyUnmarshalMethod(t *testing.T) {
	sm := &selfMarshaler{}
	m := impl.Export{}.MessageOf(sm).Interface()
	want := []byte("unmarshal")
	if err := proto.Unmarshal(want, m); err != nil {
		t.Fatalf("proto.Unmarshal(selfMarshaler{}) = %v, want nil", err)
	}
	if !bytes.Equal(sm.bytes, want) {
		t.Fatalf("proto.Unmarshal(selfMarshaler{}): Marshal method not called")
	}
}

type descPanicSelfMarshaler struct{}

const descPanicSelfMarshalerBytes = "bytes"

func (m descPanicSelfMarshaler) Reset()                      {}
func (m descPanicSelfMarshaler) ProtoMessage()               {}
func (m descPanicSelfMarshaler) Descriptor() ([]byte, []int) { panic("Descriptor method panics") }
func (m descPanicSelfMarshaler) String() string              { return "descPanicSelfMarshaler{}" }
func (m descPanicSelfMarshaler) Marshal() ([]byte, error) {
	return []byte(descPanicSelfMarshalerBytes), nil
}

func TestSelfMarshalerDescriptorPanics(t *testing.T) {
	m := descPanicSelfMarshaler{}
	got, err := proto.Marshal(impl.Export{}.MessageOf(m).Interface())
	want := []byte(descPanicSelfMarshalerBytes)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("proto.Marshal(%v) = %v, %v; want %v, nil", m, got, err, want)
	}
}
