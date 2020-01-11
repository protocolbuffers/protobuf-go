// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style.
// license that can be found in the LICENSE file.

package proto_test

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/runtime/protoimpl"

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
			ext:         testpb.E_OptionalInt32Extension,
			wantDefault: int32(0),
			value:       int32(1),
		},
		{
			message:     &testpb.TestAllExtensions{},
			ext:         testpb.E_RepeatedStringExtension,
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
