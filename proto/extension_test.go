// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"

	"google.golang.org/protobuf/proto"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/runtime/protoimpl"
	"google.golang.org/protobuf/testing/protocmp"

	legacy1pb "google.golang.org/protobuf/internal/testprotos/legacy/proto2_20160225_2fc053c5"
	testpb "google.golang.org/protobuf/internal/testprotos/test"
)

func TestExtensionFuncs(t *testing.T) {
	for _, test := range []struct {
		message     proto.Message
		ext         pref.ExtensionType
		wantDefault interface{}
		value       interface{}
	}{
		{
			message:     &testpb.TestAllExtensions{},
			ext:         testpb.E_OptionalInt32,
			wantDefault: int32(0),
			value:       int32(1),
		},
		{
			message:     &testpb.TestAllExtensions{},
			ext:         testpb.E_RepeatedString,
			wantDefault: ([]string)(nil),
			value:       []string{"a", "b", "c"},
		},
		{
			message:     protoimpl.X.MessageOf(&legacy1pb.Message{}).Interface(),
			ext:         legacy1pb.E_Message_ExtensionOptionalBool,
			wantDefault: false,
			value:       true,
		},
	} {
		desc := fmt.Sprintf("Extension %v, value %v", test.ext.TypeDescriptor().FullName(), test.value)
		if proto.HasExtension(test.message, test.ext) {
			t.Errorf("%v:\nbefore setting extension HasExtension(...) = true, want false", desc)
		}
		got := proto.GetExtension(test.message, test.ext)
		if d := cmp.Diff(test.wantDefault, got); d != "" {
			t.Errorf("%v:\nbefore setting extension GetExtension(...) returns unexpected value (-want,+got):\n%v", desc, d)
		}
		proto.SetExtension(test.message, test.ext, test.value)
		if !proto.HasExtension(test.message, test.ext) {
			t.Errorf("%v:\nafter setting extension HasExtension(...) = false, want true", desc)
		}
		got = proto.GetExtension(test.message, test.ext)
		if d := cmp.Diff(test.value, got); d != "" {
			t.Errorf("%v:\nafter setting extension GetExtension(...) returns unexpected value (-want,+got):\n%v", desc, d)
		}
		proto.ClearExtension(test.message, test.ext)
		if proto.HasExtension(test.message, test.ext) {
			t.Errorf("%v:\nafter clearing extension HasExtension(...) = true, want false", desc)
		}
	}
}

func TestExtensionRanger(t *testing.T) {
	want := map[pref.ExtensionType]interface{}{
		testpb.E_OptionalInt32:         int32(5),
		testpb.E_OptionalString:        string("hello"),
		testpb.E_OptionalNestedMessage: &testpb.TestAllExtensions_NestedMessage{},
		testpb.E_OptionalNestedEnum:    testpb.TestAllTypes_BAZ,
		testpb.E_RepeatedFloat:         []float32{+32.32, -32.32},
		testpb.E_RepeatedNestedMessage: []*testpb.TestAllExtensions_NestedMessage{{}},
		testpb.E_RepeatedNestedEnum:    []testpb.TestAllTypes_NestedEnum{testpb.TestAllTypes_BAZ},
	}

	m := &testpb.TestAllExtensions{}
	for xt, v := range want {
		proto.SetExtension(m, xt, v)
	}

	got := make(map[pref.ExtensionType]interface{})
	proto.RangeExtensions(m, func(xt pref.ExtensionType, v interface{}) bool {
		got[xt] = v
		return true
	})

	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("proto.RangeExtensions mismatch (-want +got):\n%s", diff)
	}
}

func TestExtensionGetRace(t *testing.T) {
	// Concurrently fetch an extension value while marshaling the message containing it.
	// Create the message with proto.Unmarshal to give lazy extension decoding (if present)
	// a chance to occur.
	want := int32(42)
	m1 := &testpb.TestAllExtensions{}
	proto.SetExtension(m1, testpb.E_OptionalNestedMessage, &testpb.TestAllExtensions_NestedMessage{A: proto.Int32(want)})
	b, err := proto.Marshal(m1)
	if err != nil {
		t.Fatal(err)
	}
	m := &testpb.TestAllExtensions{}
	if err := proto.Unmarshal(b, m); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := proto.Marshal(m); err != nil {
				t.Error(err)
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			got := proto.GetExtension(m, testpb.E_OptionalNestedMessage).(*testpb.TestAllExtensions_NestedMessage).GetA()
			if got != want {
				t.Errorf("GetExtension(optional_nested_message).a = %v, want %v", got, want)
			}
		}()
	}
	wg.Wait()
}
