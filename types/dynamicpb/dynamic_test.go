// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dynamicpb_test

import (
	"testing"

	"google.golang.org/protobuf/proto"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	preg "google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/testing/prototest"
	"google.golang.org/protobuf/types/dynamicpb"

	testpb "google.golang.org/protobuf/internal/testprotos/test"
	test3pb "google.golang.org/protobuf/internal/testprotos/test3"
)

func TestConformance(t *testing.T) {
	for _, message := range []proto.Message{
		(*testpb.TestAllTypes)(nil),
		(*test3pb.TestAllTypes)(nil),
		(*testpb.TestAllExtensions)(nil),
	} {
		prototest.TestMessage(t, dynamicpb.New(message.ProtoReflect().Descriptor()), prototest.MessageOptions{})
	}
}

func TestDynamicExtensions(t *testing.T) {
	file, err := preg.GlobalFiles.FindFileByPath("test/ext.proto")
	if err != nil {
		t.Fatal(err)
	}

	md := (&testpb.TestAllExtensions{}).ProtoReflect().Descriptor()
	opts := prototest.MessageOptions{
		Resolver: extResolver{},
	}
	for i := 0; i < file.Extensions().Len(); i++ {
		opts.ExtensionTypes = append(opts.ExtensionTypes, dynamicpb.NewExtensionType(file.Extensions().Get(i)))
	}
	prototest.TestMessage(t, dynamicpb.New(md), opts)
}

type extResolver struct{}

func (extResolver) FindExtensionByName(field pref.FullName) (pref.ExtensionType, error) {
	xt, err := preg.GlobalTypes.FindExtensionByName(field)
	if err != nil {
		return nil, err
	}
	return dynamicpb.NewExtensionType(xt.TypeDescriptor().Descriptor()), nil
}

func (extResolver) FindExtensionByNumber(message pref.FullName, field pref.FieldNumber) (pref.ExtensionType, error) {
	xt, err := preg.GlobalTypes.FindExtensionByNumber(message, field)
	if err != nil {
		return nil, err
	}
	return dynamicpb.NewExtensionType(xt.TypeDescriptor().Descriptor()), nil
}
