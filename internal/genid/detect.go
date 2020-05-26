// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package genid

import "google.golang.org/protobuf/reflect/protoreflect"

type ProtoFile int

const (
	Unknown_file ProtoFile = iota
	Any_file
	Timestamp_file
	Duration_file
	Wrappers_file
	Struct_file
	FieldMask_file
	Api_file
	Type_file
	SourceContext_file
	Empty_file
)

var wellKnownTypes = map[protoreflect.FullName]ProtoFile{
	Any_message_fullname:            Any_file,
	Timestamp_message_fullname:      Timestamp_file,
	Duration_message_fullname:       Duration_file,
	BoolValue_message_fullname:      Wrappers_file,
	Int32Value_message_fullname:     Wrappers_file,
	Int64Value_message_fullname:     Wrappers_file,
	UInt32Value_message_fullname:    Wrappers_file,
	UInt64Value_message_fullname:    Wrappers_file,
	FloatValue_message_fullname:     Wrappers_file,
	DoubleValue_message_fullname:    Wrappers_file,
	BytesValue_message_fullname:     Wrappers_file,
	StringValue_message_fullname:    Wrappers_file,
	Struct_message_fullname:         Struct_file,
	ListValue_message_fullname:      Struct_file,
	Value_message_fullname:          Struct_file,
	NullValue_enum_fullname:         Struct_file,
	FieldMask_message_fullname:      FieldMask_file,
	Api_message_fullname:            Api_file,
	Method_message_fullname:         Api_file,
	Mixin_message_fullname:          Api_file,
	Syntax_enum_fullname:            Type_file,
	Type_message_fullname:           Type_file,
	Field_message_fullname:          Type_file,
	Field_Kind_enum_fullname:        Type_file,
	Field_Cardinality_enum_fullname: Type_file,
	Enum_message_fullname:           Type_file,
	EnumValue_message_fullname:      Type_file,
	Option_message_fullname:         Type_file,
	SourceContext_message_fullname:  SourceContext_file,
	Empty_message_fullname:          Empty_file,
}

// WhichFile identifies the proto file that an enum or message belongs to.
// It currently only identifies well-known types.
func WhichFile(s protoreflect.FullName) ProtoFile {
	return wellKnownTypes[s]
}
