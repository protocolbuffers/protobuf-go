// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protodesc

import (
	"fmt"
	"reflect"
	"strings"

	"google.golang.org/protobuf/internal/encoding/defval"
	"google.golang.org/protobuf/internal/scalar"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"google.golang.org/protobuf/types/descriptorpb"
)

// ToFileDescriptorProto copies a protoreflect.FileDescriptor into a
// google.protobuf.FileDescriptorProto message.
func ToFileDescriptorProto(file protoreflect.FileDescriptor) *descriptorpb.FileDescriptorProto {
	p := &descriptorpb.FileDescriptorProto{
		Name:    scalar.String(file.Path()),
		Package: scalar.String(string(file.Package())),
		Options: clone(file.Options()).(*descriptorpb.FileOptions),
	}
	for i, imports := 0, file.Imports(); i < imports.Len(); i++ {
		imp := imports.Get(i)
		p.Dependency = append(p.Dependency, imp.Path())
		if imp.IsPublic {
			p.PublicDependency = append(p.PublicDependency, int32(i))
		}
		if imp.IsWeak {
			p.WeakDependency = append(p.WeakDependency, int32(i))
		}
	}
	for i, messages := 0, file.Messages(); i < messages.Len(); i++ {
		p.MessageType = append(p.MessageType, ToDescriptorProto(messages.Get(i)))
	}
	for i, enums := 0, file.Enums(); i < enums.Len(); i++ {
		p.EnumType = append(p.EnumType, ToEnumDescriptorProto(enums.Get(i)))
	}
	for i, services := 0, file.Services(); i < services.Len(); i++ {
		p.Service = append(p.Service, ToServiceDescriptorProto(services.Get(i)))
	}
	for i, exts := 0, file.Extensions(); i < exts.Len(); i++ {
		p.Extension = append(p.Extension, ToFieldDescriptorProto(exts.Get(i)))
	}
	if syntax := file.Syntax(); syntax != protoreflect.Proto2 {
		p.Syntax = scalar.String(file.Syntax().String())
	}
	return p
}

// ToDescriptorProto copies a protoreflect.MessageDescriptor into a
// google.protobuf.DescriptorProto message.
func ToDescriptorProto(message protoreflect.MessageDescriptor) *descriptorpb.DescriptorProto {
	p := &descriptorpb.DescriptorProto{
		Name:    scalar.String(string(message.Name())),
		Options: clone(message.Options()).(*descriptorpb.MessageOptions),
	}
	for i, fields := 0, message.Fields(); i < fields.Len(); i++ {
		p.Field = append(p.Field, ToFieldDescriptorProto(fields.Get(i)))
	}
	for i, exts := 0, message.Extensions(); i < exts.Len(); i++ {
		p.Extension = append(p.Extension, ToFieldDescriptorProto(exts.Get(i)))
	}
	for i, messages := 0, message.Messages(); i < messages.Len(); i++ {
		p.NestedType = append(p.NestedType, ToDescriptorProto(messages.Get(i)))
	}
	for i, enums := 0, message.Enums(); i < enums.Len(); i++ {
		p.EnumType = append(p.EnumType, ToEnumDescriptorProto(enums.Get(i)))
	}
	for i, xranges := 0, message.ExtensionRanges(); i < xranges.Len(); i++ {
		xrange := xranges.Get(i)
		p.ExtensionRange = append(p.ExtensionRange, &descriptorpb.DescriptorProto_ExtensionRange{
			Start:   scalar.Int32(int32(xrange[0])),
			End:     scalar.Int32(int32(xrange[1])),
			Options: clone(message.ExtensionRangeOptions(i)).(*descriptorpb.ExtensionRangeOptions),
		})
	}
	for i, oneofs := 0, message.Oneofs(); i < oneofs.Len(); i++ {
		p.OneofDecl = append(p.OneofDecl, ToOneofDescriptorProto(oneofs.Get(i)))
	}
	for i, ranges := 0, message.ReservedRanges(); i < ranges.Len(); i++ {
		rrange := ranges.Get(i)
		p.ReservedRange = append(p.ReservedRange, &descriptorpb.DescriptorProto_ReservedRange{
			Start: scalar.Int32(int32(rrange[0])),
			End:   scalar.Int32(int32(rrange[1])),
		})
	}
	for i, names := 0, message.ReservedNames(); i < names.Len(); i++ {
		p.ReservedName = append(p.ReservedName, string(names.Get(i)))
	}
	return p
}

// ToFieldDescriptorProto copies a protoreflect.FieldDescriptor into a
// google.protobuf.FieldDescriptorProto message.
func ToFieldDescriptorProto(field protoreflect.FieldDescriptor) *descriptorpb.FieldDescriptorProto {
	p := &descriptorpb.FieldDescriptorProto{
		Name:    scalar.String(string(field.Name())),
		Number:  scalar.Int32(int32(field.Number())),
		Label:   descriptorpb.FieldDescriptorProto_Label(field.Cardinality()).Enum(),
		Options: clone(field.Options()).(*descriptorpb.FieldOptions),
	}
	if field.IsExtension() {
		p.Extendee = fullNameOf(field.ContainingMessage())
	}
	if field.Kind().IsValid() {
		p.Type = descriptorpb.FieldDescriptorProto_Type(field.Kind()).Enum()
	}
	if field.Enum() != nil {
		p.TypeName = fullNameOf(field.Enum())
	}
	if field.Message() != nil {
		p.TypeName = fullNameOf(field.Message())
	}
	if field.HasJSONName() {
		p.JsonName = scalar.String(field.JSONName())
	}
	if field.HasDefault() {
		def, err := defval.Marshal(field.Default(), field.DefaultEnumValue(), field.Kind(), defval.Descriptor)
		if err != nil && field.DefaultEnumValue() != nil {
			def = string(field.DefaultEnumValue().Name()) // occurs for unresolved enum values
		} else if err != nil {
			panic(fmt.Sprintf("%v: %v", field.FullName(), err))
		}
		p.DefaultValue = scalar.String(def)
	}
	if oneof := field.ContainingOneof(); oneof != nil {
		p.OneofIndex = scalar.Int32(int32(oneof.Index()))
	}
	return p
}

// ToOneofDescriptorProto copies a protoreflect.OneofDescriptor into a
// google.protobuf.OneofDescriptorProto message.
func ToOneofDescriptorProto(oneof protoreflect.OneofDescriptor) *descriptorpb.OneofDescriptorProto {
	return &descriptorpb.OneofDescriptorProto{
		Name:    scalar.String(string(oneof.Name())),
		Options: clone(oneof.Options()).(*descriptorpb.OneofOptions),
	}
}

// ToEnumDescriptorProto copies a protoreflect.EnumDescriptor into a
// google.protobuf.EnumDescriptorProto message.
func ToEnumDescriptorProto(enum protoreflect.EnumDescriptor) *descriptorpb.EnumDescriptorProto {
	p := &descriptorpb.EnumDescriptorProto{
		Name:    scalar.String(string(enum.Name())),
		Options: clone(enum.Options()).(*descriptorpb.EnumOptions),
	}
	for i, values := 0, enum.Values(); i < values.Len(); i++ {
		p.Value = append(p.Value, ToEnumValueDescriptorProto(values.Get(i)))
	}
	for i, ranges := 0, enum.ReservedRanges(); i < ranges.Len(); i++ {
		rrange := ranges.Get(i)
		p.ReservedRange = append(p.ReservedRange, &descriptorpb.EnumDescriptorProto_EnumReservedRange{
			Start: scalar.Int32(int32(rrange[0])),
			End:   scalar.Int32(int32(rrange[1])),
		})
	}
	for i, names := 0, enum.ReservedNames(); i < names.Len(); i++ {
		p.ReservedName = append(p.ReservedName, string(names.Get(i)))
	}
	return p
}

// ToEnumValueDescriptorProto copies a protoreflect.EnumValueDescriptor into a
// google.protobuf.EnumValueDescriptorProto message.
func ToEnumValueDescriptorProto(value protoreflect.EnumValueDescriptor) *descriptorpb.EnumValueDescriptorProto {
	return &descriptorpb.EnumValueDescriptorProto{
		Name:    scalar.String(string(value.Name())),
		Number:  scalar.Int32(int32(value.Number())),
		Options: clone(value.Options()).(*descriptorpb.EnumValueOptions),
	}
}

// ToServiceDescriptorProto copies a protoreflect.ServiceDescriptor into a
// google.protobuf.ServiceDescriptorProto message.
func ToServiceDescriptorProto(service protoreflect.ServiceDescriptor) *descriptorpb.ServiceDescriptorProto {
	p := &descriptorpb.ServiceDescriptorProto{
		Name:    scalar.String(string(service.Name())),
		Options: clone(service.Options()).(*descriptorpb.ServiceOptions),
	}
	for i, methods := 0, service.Methods(); i < methods.Len(); i++ {
		p.Method = append(p.Method, ToMethodDescriptorProto(methods.Get(i)))
	}
	return p
}

// ToMethodDescriptorProto copies a protoreflect.MethodDescriptor into a
// google.protobuf.MethodDescriptorProto message.
func ToMethodDescriptorProto(method protoreflect.MethodDescriptor) *descriptorpb.MethodDescriptorProto {
	p := &descriptorpb.MethodDescriptorProto{
		Name:       scalar.String(string(method.Name())),
		InputType:  fullNameOf(method.Input()),
		OutputType: fullNameOf(method.Output()),
		Options:    clone(method.Options()).(*descriptorpb.MethodOptions),
	}
	if method.IsStreamingClient() {
		p.ClientStreaming = scalar.Bool(true)
	}
	if method.IsStreamingServer() {
		p.ServerStreaming = scalar.Bool(true)
	}
	return p
}

func fullNameOf(d protoreflect.Descriptor) *string {
	if d == nil {
		return nil
	}
	if strings.HasPrefix(string(d.FullName()), unknownPrefix) {
		return scalar.String(string(d.FullName()[len(unknownPrefix):]))
	}
	return scalar.String("." + string(d.FullName()))
}

func clone(src proto.Message) proto.Message {
	if reflect.ValueOf(src).IsNil() {
		return src
	}
	dst := src.ProtoReflect().New().Interface()
	proto.Merge(dst, src)
	return dst
}
