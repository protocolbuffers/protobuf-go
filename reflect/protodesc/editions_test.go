// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protodesc

import (
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/gofeaturespb"
)

func TestGoFeaturesDynamic(t *testing.T) {
	md := (*gofeaturespb.GoFeatures)(nil).ProtoReflect().Descriptor()
	gf := dynamicpb.NewMessage(md)
	opaque := protoreflect.ValueOfEnum(gofeaturespb.GoFeatures_API_OPAQUE.Number())
	gf.Set(md.Fields().ByName("api_level"), opaque)
	featureSet := &descriptorpb.FeatureSet{}
	dynamicExt := dynamicpb.NewExtensionType(gofeaturespb.E_Go.TypeDescriptor().Descriptor())
	proto.SetExtension(featureSet, dynamicExt, gf)

	fd := &descriptorpb.FileDescriptorProto{
		Name: proto.String("a.proto"),
		Dependency: []string{
			"google/protobuf/go_features.proto",
		},
		Edition: descriptorpb.Edition_EDITION_2023.Enum(),
		Syntax:  proto.String("editions"),
		Options: &descriptorpb.FileOptions{
			Features: featureSet,
		},
	}
	fds := &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			ToFileDescriptorProto(descriptorpb.File_google_protobuf_descriptor_proto),
			ToFileDescriptorProto(gofeaturespb.File_google_protobuf_go_features_proto),
			fd,
		},
	}
	if _, err := NewFiles(fds); err != nil {
		t.Fatal(err)
	}
}
