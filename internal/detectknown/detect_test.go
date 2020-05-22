// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package detectknown_test

import (
	"testing"

	"google.golang.org/protobuf/internal/detectknown"
	"google.golang.org/protobuf/reflect/protoreflect"

	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/apipb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/sourcecontextpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/typepb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestWhich(t *testing.T) {
	tests := []struct {
		in   protoreflect.FileDescriptor
		want detectknown.ProtoFile
	}{
		{descriptorpb.File_google_protobuf_descriptor_proto, detectknown.Unknown},
		{pluginpb.File_google_protobuf_compiler_plugin_proto, detectknown.Unknown},
		{anypb.File_google_protobuf_any_proto, detectknown.AnyProto},
		{timestamppb.File_google_protobuf_timestamp_proto, detectknown.TimestampProto},
		{durationpb.File_google_protobuf_duration_proto, detectknown.DurationProto},
		{wrapperspb.File_google_protobuf_wrappers_proto, detectknown.WrappersProto},
		{structpb.File_google_protobuf_struct_proto, detectknown.StructProto},
		{fieldmaskpb.File_google_protobuf_field_mask_proto, detectknown.FieldMaskProto},
		{emptypb.File_google_protobuf_empty_proto, detectknown.EmptyProto},
		{apipb.File_google_protobuf_api_proto, detectknown.ApiProto},
		{typepb.File_google_protobuf_type_proto, detectknown.TypeProto},
		{sourcecontextpb.File_google_protobuf_source_context_proto, detectknown.SourceContextProto},
	}

	for _, tt := range tests {
		rangeDescriptors(tt.in, func(d protoreflect.Descriptor) {
			got := detectknown.Which(d.FullName())
			if got != tt.want {
				t.Errorf("Which(%s) = %v, want %v", d.FullName(), got, tt.want)
			}
		})
	}
}

func rangeDescriptors(d interface {
	Enums() protoreflect.EnumDescriptors
	Messages() protoreflect.MessageDescriptors
}, f func(protoreflect.Descriptor)) {
	for i := 0; i < d.Enums().Len(); i++ {
		ed := d.Enums().Get(i)
		f(ed)
	}
	for i := 0; i < d.Messages().Len(); i++ {
		md := d.Messages().Get(i)
		if md.IsMapEntry() {
			continue
		}
		f(md)
		rangeDescriptors(md, f)
	}
}
