// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protodelim_test

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/encoding/protodelim"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/internal/testprotos/test3"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestRoundTrip(t *testing.T) {
	msgs := []*test3.TestAllTypes{
		{SingularInt32: 1},
		{SingularString: "hello"},
		{RepeatedDouble: []float64{1.2, 3.4}},
		{
			SingularNestedMessage:  &test3.TestAllTypes_NestedMessage{A: 1},
			RepeatedForeignMessage: []*test3.ForeignMessage{{C: 2}, {D: 3}},
		},
	}

	buf := &bytes.Buffer{}

	// Write all messages to buf.
	for _, m := range msgs {
		if n, err := protodelim.MarshalTo(buf, m); err != nil {
			t.Errorf("protodelim.MarshalTo(_, %v) = %d, %v", m, n, err)
		}
	}

	// Read and collect messages from buf.
	var got []*test3.TestAllTypes
	r := bufio.NewReader(buf)
	for {
		m := &test3.TestAllTypes{}
		err := protodelim.UnmarshalFrom(r, m)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Errorf("protodelim.UnmarshalFrom(_) = %v", err)
			continue
		}
		got = append(got, m)
	}

	want := msgs
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("Unmarshaler collected messages: diff -want +got = %s", diff)
	}
}

func TestMaxSize(t *testing.T) {
	in := &test3.TestAllTypes{SingularInt32: 1}

	buf := &bytes.Buffer{}

	if n, err := protodelim.MarshalTo(buf, in); err != nil {
		t.Errorf("protodelim.MarshalTo(_, %v) = %d, %v", in, n, err)
	}

	out := &test3.TestAllTypes{}
	err := protodelim.UnmarshalOptions{MaxSize: 1}.UnmarshalFrom(bufio.NewReader(buf), out)

	var errSize *protodelim.SizeTooLargeError
	if !errors.As(err, &errSize) {
		t.Errorf("protodelim.UnmarshalOptions{MaxSize: 1}.UnmarshalFrom(_, _) = %v (%T), want %T", err, err, errSize)
	}
	got, want := errSize, &protodelim.SizeTooLargeError{Size: 3, MaxSize: 1}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("protodelim.UnmarshalOptions{MaxSize: 1}.UnmarshalFrom(_, _): diff -want +got = %s", diff)
	}
}

func TestUnmarshalFrom_UnexpectedEOF(t *testing.T) {
	buf := &bytes.Buffer{}

	// Write a size (42), but no subsequent message.
	sb := protowire.AppendVarint(nil, 42)
	if _, err := buf.Write(sb); err != nil {
		t.Fatalf("buf.Write(%v) = _, %v", sb, err)
	}

	out := &test3.TestAllTypes{}
	err := protodelim.UnmarshalFrom(bufio.NewReader(buf), out)
	if got, want := err, io.ErrUnexpectedEOF; got != want {
		t.Errorf("protodelim.UnmarshalFrom(size-only buf, _) = %v, want %v", got, want)
	}
}
