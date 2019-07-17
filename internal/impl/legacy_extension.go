// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"reflect"
	"sync"

	"google.golang.org/protobuf/internal/descfmt"
	ptag "google.golang.org/protobuf/internal/encoding/tag"
	"google.golang.org/protobuf/internal/filedesc"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	preg "google.golang.org/protobuf/reflect/protoregistry"
	piface "google.golang.org/protobuf/runtime/protoiface"
)

// legacyExtensionDescKey is a comparable version of protoiface.ExtensionDescV1
// suitable for use as a key in a map.
type legacyExtensionDescKey struct {
	typeV2        pref.ExtensionType
	extendedType  reflect.Type
	extensionType reflect.Type
	field         int32
	name          string
	tag           string
	filename      string
}

func legacyExtensionDescKeyOf(d *piface.ExtensionDescV1) legacyExtensionDescKey {
	return legacyExtensionDescKey{
		d.Type,
		reflect.TypeOf(d.ExtendedType),
		reflect.TypeOf(d.ExtensionType),
		d.Field, d.Name, d.Tag, d.Filename,
	}
}

var (
	legacyExtensionTypeCache sync.Map // map[legacyExtensionDescKey]protoreflect.ExtensionType
	legacyExtensionDescCache sync.Map // map[protoreflect.ExtensionType]*protoiface.ExtensionDescV1
)

// legacyExtensionDescFromType converts a v2 protoreflect.ExtensionType to a
// protoiface.ExtensionDescV1. The returned ExtensionDesc must not be mutated.
func legacyExtensionDescFromType(xt pref.ExtensionType) *piface.ExtensionDescV1 {
	// Fast-path: check whether an extension desc is already nested within.
	if xt, ok := xt.(interface {
		ProtoLegacyExtensionDesc() *piface.ExtensionDescV1
	}); ok {
		if d := xt.ProtoLegacyExtensionDesc(); d != nil {
			return d
		}
	}

	// Fast-path: check the cache for whether this ExtensionType has already
	// been converted to a legacy descriptor.
	if d, ok := legacyExtensionDescCache.Load(xt); ok {
		return d.(*piface.ExtensionDescV1)
	}

	// Determine the parent type if possible.
	var parent piface.MessageV1
	messageName := xt.ContainingMessage().FullName()
	if mt, _ := preg.GlobalTypes.FindMessageByName(messageName); mt != nil {
		// Create a new parent message and unwrap it if possible.
		mv := mt.New().Interface()
		t := reflect.TypeOf(mv)
		if mv, ok := mv.(Unwrapper); ok {
			t = reflect.TypeOf(mv.ProtoUnwrap())
		}

		// Check whether the message implements the legacy v1 Message interface.
		mz := reflect.Zero(t).Interface()
		if mz, ok := mz.(piface.MessageV1); ok {
			parent = mz
		}
	}

	// Determine the v1 extension type, which is unfortunately not the same as
	// the v2 ExtensionType.GoType.
	extType := xt.GoType()
	switch extType.Kind() {
	case reflect.Bool, reflect.Int32, reflect.Int64, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64, reflect.String:
		extType = reflect.PtrTo(extType) // T -> *T for singular scalar fields
	case reflect.Ptr:
		if extType.Elem().Kind() == reflect.Slice {
			extType = extType.Elem() // *[]T -> []T for repeated fields
		}
	}

	// Reconstruct the legacy enum full name, which is an odd mixture of the
	// proto package name with the Go type name.
	var enumName string
	if xt.Kind() == pref.EnumKind {
		// Derive Go type name.
		t := extType
		if t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice {
			t = t.Elem()
		}
		enumName = t.Name()

		// Derive the proto package name.
		// For legacy enums, obtain the proto package from the raw descriptor.
		var protoPkg string
		if fd := xt.Enum().ParentFile(); fd != nil {
			protoPkg = string(fd.Package())
		}
		if ed, ok := reflect.Zero(t).Interface().(enumV1); ok && protoPkg == "" {
			b, _ := ed.EnumDescriptor()
			protoPkg = string(legacyLoadFileDesc(b).Package())
		}

		if protoPkg != "" {
			enumName = protoPkg + "." + enumName
		}
	}

	// Derive the proto file that the extension was declared within.
	var filename string
	if fd := xt.ParentFile(); fd != nil {
		filename = fd.Path()
	}

	// Construct and return a ExtensionDescV1.
	d := &piface.ExtensionDescV1{
		Type:          xt,
		ExtendedType:  parent,
		ExtensionType: reflect.Zero(extType).Interface(),
		Field:         int32(xt.Number()),
		Name:          string(xt.FullName()),
		Tag:           ptag.Marshal(xt, enumName),
		Filename:      filename,
	}
	if d, ok := legacyExtensionDescCache.LoadOrStore(xt, d); ok {
		return d.(*piface.ExtensionDescV1)
	}
	return d
}

// legacyExtensionTypeFromDesc converts a protoiface.ExtensionDescV1 to a
// v2 protoreflect.ExtensionType. The returned descriptor type takes ownership
// of the input extension desc. The input must not be mutated so long as the
// returned type is still in use.
func legacyExtensionTypeFromDesc(d *piface.ExtensionDescV1) pref.ExtensionType {
	// Fast-path: check whether an extension type is already nested within.
	if d.Type != nil {
		return d.Type
	}

	// Fast-path: check the cache for whether this ExtensionType has already
	// been converted from a legacy descriptor.
	dk := legacyExtensionDescKeyOf(d)
	if t, ok := legacyExtensionTypeCache.Load(dk); ok {
		return t.(pref.ExtensionType)
	}

	// Resolve enum or message dependencies.
	var ed pref.EnumDescriptor
	var md pref.MessageDescriptor
	t := reflect.TypeOf(d.ExtensionType)
	isOptional := t.Kind() == reflect.Ptr && t.Elem().Kind() != reflect.Struct
	isRepeated := t.Kind() == reflect.Slice && t.Elem().Kind() != reflect.Uint8
	if isOptional || isRepeated {
		t = t.Elem()
	}
	switch v := reflect.Zero(t).Interface().(type) {
	case pref.Enum:
		ed = v.Descriptor()
	case enumV1:
		ed = LegacyLoadEnumDesc(t)
	case pref.ProtoMessage:
		md = v.ProtoReflect().Descriptor()
	case messageV1:
		md = LegacyLoadMessageDesc(t)
	}

	// Derive basic field information from the struct tag.
	var evs pref.EnumValueDescriptors
	if ed != nil {
		evs = ed.Values()
	}
	fd := ptag.Unmarshal(d.Tag, t, evs).(*filedesc.Field)

	// Construct a v2 ExtensionType.
	xd := &filedesc.Extension{L2: new(filedesc.ExtensionL2)}
	xd.L0.ParentFile = filedesc.SurrogateProto2
	xd.L0.FullName = pref.FullName(d.Name)
	xd.L1.Number = pref.FieldNumber(d.Field)
	xd.L2.Cardinality = fd.L1.Cardinality
	xd.L1.Kind = fd.L1.Kind
	xd.L2.IsPacked = fd.L1.IsPacked
	xd.L2.Default = fd.L1.Default
	xd.L1.Extendee = Export{}.MessageDescriptorOf(d.ExtendedType)
	xd.L2.Enum = ed
	xd.L2.Message = md
	tt := reflect.TypeOf(d.ExtensionType)
	if isOptional {
		tt = tt.Elem()
	} else if isRepeated {
		tt = reflect.PtrTo(tt)
	}
	xt := LegacyExtensionTypeOf(xd, tt)

	// Cache the conversion for both directions.
	legacyExtensionDescCache.LoadOrStore(xt, d)
	if xt, ok := legacyExtensionTypeCache.LoadOrStore(dk, xt); ok {
		return xt.(pref.ExtensionType)
	}
	return xt
}

// LegacyExtensionTypeOf returns a protoreflect.ExtensionType where the
// element type of the field is t.
//
// This is exported for testing purposes.
func LegacyExtensionTypeOf(xd pref.ExtensionDescriptor, t reflect.Type) pref.ExtensionType {
	return &legacyExtensionType{
		ExtensionDescriptor: xd,
		typ:                 t,
		conv:                NewConverter(t, xd),
	}
}

type legacyExtensionType struct {
	pref.ExtensionDescriptor
	typ  reflect.Type
	conv Converter
}

func (x *legacyExtensionType) GoType() reflect.Type { return x.typ }
func (x *legacyExtensionType) New() pref.Value      { return x.conv.New() }
func (x *legacyExtensionType) ValueOf(v interface{}) pref.Value {
	return x.conv.PBValueOf(reflect.ValueOf(v))
}
func (x *legacyExtensionType) InterfaceOf(v pref.Value) interface{} {
	return x.conv.GoValueOf(v).Interface()
}
func (x *legacyExtensionType) Descriptor() pref.ExtensionDescriptor { return x.ExtensionDescriptor }
func (x *legacyExtensionType) Format(s fmt.State, r rune)           { descfmt.FormatDesc(s, r, x) }
