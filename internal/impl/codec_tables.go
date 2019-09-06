// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"reflect"

	"google.golang.org/protobuf/internal/encoding/wire"
	"google.golang.org/protobuf/internal/strs"
	pref "google.golang.org/protobuf/reflect/protoreflect"
)

// pointerCoderFuncs is a set of pointer encoding functions.
type pointerCoderFuncs struct {
	size      func(p pointer, tagsize int, opts marshalOptions) int
	marshal   func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error)
	unmarshal func(b []byte, p pointer, wtyp wire.Type, opts unmarshalOptions) (int, error)
	isInit    func(p pointer) error
}

// valueCoderFuncs is a set of protoreflect.Value encoding functions.
type valueCoderFuncs struct {
	size      func(v pref.Value, tagsize int, opts marshalOptions) int
	marshal   func(b []byte, v pref.Value, wiretag uint64, opts marshalOptions) ([]byte, error)
	unmarshal func(b []byte, v pref.Value, num wire.Number, wtyp wire.Type, opts unmarshalOptions) (pref.Value, int, error)
	isInit    func(v pref.Value) error
}

// fieldCoder returns pointer functions for a field, used for operating on
// struct fields.
func fieldCoder(fd pref.FieldDescriptor, ft reflect.Type) pointerCoderFuncs {
	switch {
	case fd.IsMap():
		return encoderFuncsForMap(fd, ft)
	case fd.Cardinality() == pref.Repeated && !fd.IsPacked():
		// Repeated fields (not packed).
		if ft.Kind() != reflect.Slice {
			break
		}
		ft := ft.Elem()
		switch fd.Kind() {
		case pref.BoolKind:
			if ft.Kind() == reflect.Bool {
				return coderBoolSlice
			}
		case pref.EnumKind:
			if ft.Kind() == reflect.Int32 {
				return coderEnumSlice
			}
		case pref.Int32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderInt32Slice
			}
		case pref.Sint32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderSint32Slice
			}
		case pref.Uint32Kind:
			if ft.Kind() == reflect.Uint32 {
				return coderUint32Slice
			}
		case pref.Int64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderInt64Slice
			}
		case pref.Sint64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderSint64Slice
			}
		case pref.Uint64Kind:
			if ft.Kind() == reflect.Uint64 {
				return coderUint64Slice
			}
		case pref.Sfixed32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderSfixed32Slice
			}
		case pref.Fixed32Kind:
			if ft.Kind() == reflect.Uint32 {
				return coderFixed32Slice
			}
		case pref.FloatKind:
			if ft.Kind() == reflect.Float32 {
				return coderFloatSlice
			}
		case pref.Sfixed64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderSfixed64Slice
			}
		case pref.Fixed64Kind:
			if ft.Kind() == reflect.Uint64 {
				return coderFixed64Slice
			}
		case pref.DoubleKind:
			if ft.Kind() == reflect.Float64 {
				return coderDoubleSlice
			}
		case pref.StringKind:
			if ft.Kind() == reflect.String && strs.EnforceUTF8(fd) {
				return coderStringSliceValidateUTF8
			}
			if ft.Kind() == reflect.String {
				return coderStringSlice
			}
			if ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Uint8 && strs.EnforceUTF8(fd) {
				return coderBytesSliceValidateUTF8
			}
			if ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Uint8 {
				return coderBytesSlice
			}
		case pref.BytesKind:
			if ft.Kind() == reflect.String {
				return coderStringSlice
			}
			if ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Uint8 {
				return coderBytesSlice
			}
		case pref.MessageKind:
			return makeMessageSliceFieldCoder(fd, ft)
		case pref.GroupKind:
			return makeGroupSliceFieldCoder(fd, ft)
		}
	case fd.Cardinality() == pref.Repeated && fd.IsPacked():
		// Packed repeated fields.
		//
		// Only repeated fields of primitive numeric types
		// (Varint, Fixed32, or Fixed64 wire type) can be packed.
		if ft.Kind() != reflect.Slice {
			break
		}
		ft := ft.Elem()
		switch fd.Kind() {
		case pref.BoolKind:
			if ft.Kind() == reflect.Bool {
				return coderBoolPackedSlice
			}
		case pref.EnumKind:
			if ft.Kind() == reflect.Int32 {
				return coderEnumPackedSlice
			}
		case pref.Int32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderInt32PackedSlice
			}
		case pref.Sint32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderSint32PackedSlice
			}
		case pref.Uint32Kind:
			if ft.Kind() == reflect.Uint32 {
				return coderUint32PackedSlice
			}
		case pref.Int64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderInt64PackedSlice
			}
		case pref.Sint64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderSint64PackedSlice
			}
		case pref.Uint64Kind:
			if ft.Kind() == reflect.Uint64 {
				return coderUint64PackedSlice
			}
		case pref.Sfixed32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderSfixed32PackedSlice
			}
		case pref.Fixed32Kind:
			if ft.Kind() == reflect.Uint32 {
				return coderFixed32PackedSlice
			}
		case pref.FloatKind:
			if ft.Kind() == reflect.Float32 {
				return coderFloatPackedSlice
			}
		case pref.Sfixed64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderSfixed64PackedSlice
			}
		case pref.Fixed64Kind:
			if ft.Kind() == reflect.Uint64 {
				return coderFixed64PackedSlice
			}
		case pref.DoubleKind:
			if ft.Kind() == reflect.Float64 {
				return coderDoublePackedSlice
			}
		}
	case fd.Kind() == pref.MessageKind:
		return makeMessageFieldCoder(fd, ft)
	case fd.Kind() == pref.GroupKind:
		return makeGroupFieldCoder(fd, ft)
	case fd.Syntax() == pref.Proto3 && fd.ContainingOneof() == nil:
		// Populated oneof fields always encode even if set to the zero value,
		// which normally are not encoded in proto3.
		switch fd.Kind() {
		case pref.BoolKind:
			if ft.Kind() == reflect.Bool {
				return coderBoolNoZero
			}
		case pref.EnumKind:
			if ft.Kind() == reflect.Int32 {
				return coderEnumNoZero
			}
		case pref.Int32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderInt32NoZero
			}
		case pref.Sint32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderSint32NoZero
			}
		case pref.Uint32Kind:
			if ft.Kind() == reflect.Uint32 {
				return coderUint32NoZero
			}
		case pref.Int64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderInt64NoZero
			}
		case pref.Sint64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderSint64NoZero
			}
		case pref.Uint64Kind:
			if ft.Kind() == reflect.Uint64 {
				return coderUint64NoZero
			}
		case pref.Sfixed32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderSfixed32NoZero
			}
		case pref.Fixed32Kind:
			if ft.Kind() == reflect.Uint32 {
				return coderFixed32NoZero
			}
		case pref.FloatKind:
			if ft.Kind() == reflect.Float32 {
				return coderFloatNoZero
			}
		case pref.Sfixed64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderSfixed64NoZero
			}
		case pref.Fixed64Kind:
			if ft.Kind() == reflect.Uint64 {
				return coderFixed64NoZero
			}
		case pref.DoubleKind:
			if ft.Kind() == reflect.Float64 {
				return coderDoubleNoZero
			}
		case pref.StringKind:
			if ft.Kind() == reflect.String && strs.EnforceUTF8(fd) {
				return coderStringNoZeroValidateUTF8
			}
			if ft.Kind() == reflect.String {
				return coderStringNoZero
			}
			if ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Uint8 && strs.EnforceUTF8(fd) {
				return coderBytesNoZeroValidateUTF8
			}
			if ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Uint8 {
				return coderBytesNoZero
			}
		case pref.BytesKind:
			if ft.Kind() == reflect.String {
				return coderStringNoZero
			}
			if ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Uint8 {
				return coderBytesNoZero
			}
		}
	case ft.Kind() == reflect.Ptr:
		ft := ft.Elem()
		switch fd.Kind() {
		case pref.BoolKind:
			if ft.Kind() == reflect.Bool {
				return coderBoolPtr
			}
		case pref.EnumKind:
			if ft.Kind() == reflect.Int32 {
				return coderEnumPtr
			}
		case pref.Int32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderInt32Ptr
			}
		case pref.Sint32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderSint32Ptr
			}
		case pref.Uint32Kind:
			if ft.Kind() == reflect.Uint32 {
				return coderUint32Ptr
			}
		case pref.Int64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderInt64Ptr
			}
		case pref.Sint64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderSint64Ptr
			}
		case pref.Uint64Kind:
			if ft.Kind() == reflect.Uint64 {
				return coderUint64Ptr
			}
		case pref.Sfixed32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderSfixed32Ptr
			}
		case pref.Fixed32Kind:
			if ft.Kind() == reflect.Uint32 {
				return coderFixed32Ptr
			}
		case pref.FloatKind:
			if ft.Kind() == reflect.Float32 {
				return coderFloatPtr
			}
		case pref.Sfixed64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderSfixed64Ptr
			}
		case pref.Fixed64Kind:
			if ft.Kind() == reflect.Uint64 {
				return coderFixed64Ptr
			}
		case pref.DoubleKind:
			if ft.Kind() == reflect.Float64 {
				return coderDoublePtr
			}
		case pref.StringKind:
			if ft.Kind() == reflect.String {
				return coderStringPtr
			}
		case pref.BytesKind:
			if ft.Kind() == reflect.String {
				return coderStringPtr
			}
		}
	default:
		switch fd.Kind() {
		case pref.BoolKind:
			if ft.Kind() == reflect.Bool {
				return coderBool
			}
		case pref.EnumKind:
			if ft.Kind() == reflect.Int32 {
				return coderEnum
			}
		case pref.Int32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderInt32
			}
		case pref.Sint32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderSint32
			}
		case pref.Uint32Kind:
			if ft.Kind() == reflect.Uint32 {
				return coderUint32
			}
		case pref.Int64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderInt64
			}
		case pref.Sint64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderSint64
			}
		case pref.Uint64Kind:
			if ft.Kind() == reflect.Uint64 {
				return coderUint64
			}
		case pref.Sfixed32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderSfixed32
			}
		case pref.Fixed32Kind:
			if ft.Kind() == reflect.Uint32 {
				return coderFixed32
			}
		case pref.FloatKind:
			if ft.Kind() == reflect.Float32 {
				return coderFloat
			}
		case pref.Sfixed64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderSfixed64
			}
		case pref.Fixed64Kind:
			if ft.Kind() == reflect.Uint64 {
				return coderFixed64
			}
		case pref.DoubleKind:
			if ft.Kind() == reflect.Float64 {
				return coderDouble
			}
		case pref.StringKind:
			if ft.Kind() == reflect.String && strs.EnforceUTF8(fd) {
				return coderStringValidateUTF8
			}
			if ft.Kind() == reflect.String {
				return coderString
			}
			if ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Uint8 && strs.EnforceUTF8(fd) {
				return coderBytesValidateUTF8
			}
			if ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Uint8 {
				return coderBytes
			}
		case pref.BytesKind:
			if ft.Kind() == reflect.String {
				return coderString
			}
			if ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Uint8 {
				return coderBytes
			}
		}
	}
	panic(fmt.Sprintf("invalid type: no encoder for %v %v %v/%v", fd.FullName(), fd.Cardinality(), fd.Kind(), ft))
}

// encoderFuncsForValue returns value functions for a field, used for
// extension values and map encoding.
func encoderFuncsForValue(fd pref.FieldDescriptor) valueCoderFuncs {
	switch {
	case fd.Cardinality() == pref.Repeated && !fd.IsPacked():
		switch fd.Kind() {
		case pref.BoolKind:
			return coderBoolSliceValue
		case pref.EnumKind:
			return coderEnumSliceValue
		case pref.Int32Kind:
			return coderInt32SliceValue
		case pref.Sint32Kind:
			return coderSint32SliceValue
		case pref.Uint32Kind:
			return coderUint32SliceValue
		case pref.Int64Kind:
			return coderInt64SliceValue
		case pref.Sint64Kind:
			return coderSint64SliceValue
		case pref.Uint64Kind:
			return coderUint64SliceValue
		case pref.Sfixed32Kind:
			return coderSfixed32SliceValue
		case pref.Fixed32Kind:
			return coderFixed32SliceValue
		case pref.FloatKind:
			return coderFloatSliceValue
		case pref.Sfixed64Kind:
			return coderSfixed64SliceValue
		case pref.Fixed64Kind:
			return coderFixed64SliceValue
		case pref.DoubleKind:
			return coderDoubleSliceValue
		case pref.StringKind:
			// We don't have a UTF-8 validating coder for repeated string fields.
			// Value coders are used for extensions and maps.
			// Extensions are never proto3, and maps never contain lists.
			return coderStringSliceValue
		case pref.BytesKind:
			return coderBytesSliceValue
		case pref.MessageKind:
			return coderMessageSliceValue
		case pref.GroupKind:
			return coderGroupSliceValue
		}
	case fd.Cardinality() == pref.Repeated && fd.IsPacked():
		switch fd.Kind() {
		case pref.BoolKind:
			return coderBoolPackedSliceValue
		case pref.EnumKind:
			return coderEnumPackedSliceValue
		case pref.Int32Kind:
			return coderInt32PackedSliceValue
		case pref.Sint32Kind:
			return coderSint32PackedSliceValue
		case pref.Uint32Kind:
			return coderUint32PackedSliceValue
		case pref.Int64Kind:
			return coderInt64PackedSliceValue
		case pref.Sint64Kind:
			return coderSint64PackedSliceValue
		case pref.Uint64Kind:
			return coderUint64PackedSliceValue
		case pref.Sfixed32Kind:
			return coderSfixed32PackedSliceValue
		case pref.Fixed32Kind:
			return coderFixed32PackedSliceValue
		case pref.FloatKind:
			return coderFloatPackedSliceValue
		case pref.Sfixed64Kind:
			return coderSfixed64PackedSliceValue
		case pref.Fixed64Kind:
			return coderFixed64PackedSliceValue
		case pref.DoubleKind:
			return coderDoublePackedSliceValue
		}
	default:
		switch fd.Kind() {
		default:
		case pref.BoolKind:
			return coderBoolValue
		case pref.EnumKind:
			return coderEnumValue
		case pref.Int32Kind:
			return coderInt32Value
		case pref.Sint32Kind:
			return coderSint32Value
		case pref.Uint32Kind:
			return coderUint32Value
		case pref.Int64Kind:
			return coderInt64Value
		case pref.Sint64Kind:
			return coderSint64Value
		case pref.Uint64Kind:
			return coderUint64Value
		case pref.Sfixed32Kind:
			return coderSfixed32Value
		case pref.Fixed32Kind:
			return coderFixed32Value
		case pref.FloatKind:
			return coderFloatValue
		case pref.Sfixed64Kind:
			return coderSfixed64Value
		case pref.Fixed64Kind:
			return coderFixed64Value
		case pref.DoubleKind:
			return coderDoubleValue
		case pref.StringKind:
			if strs.EnforceUTF8(fd) {
				return coderStringValueValidateUTF8
			}
			return coderStringValue
		case pref.BytesKind:
			return coderBytesValue
		case pref.MessageKind:
			return coderMessageValue
		case pref.GroupKind:
			return coderGroupValue
		}
	}
	panic(fmt.Sprintf("invalid field: no encoder for %v %v %v", fd.FullName(), fd.Cardinality(), fd.Kind()))
}
