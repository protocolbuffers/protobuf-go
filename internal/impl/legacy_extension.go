// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"reflect"
	"sync"

	"google.golang.org/protobuf/internal/encoding/messageset"
	ptag "google.golang.org/protobuf/internal/encoding/tag"
	"google.golang.org/protobuf/internal/filedesc"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	preg "google.golang.org/protobuf/reflect/protoregistry"
	piface "google.golang.org/protobuf/runtime/protoiface"
)

var legacyExtensionInfoCache sync.Map // map[protoreflect.ExtensionType]*ExtensionInfo

// legacyExtensionDescFromType converts a protoreflect.ExtensionType to an
// ExtensionInfo. The returned ExtensionInfo must not be mutated.
func legacyExtensionDescFromType(xt pref.ExtensionType) *ExtensionInfo {
	// Fast-path: check whether this is an ExtensionInfo.
	if xt, ok := xt.(*ExtensionInfo); ok {
		return xt
	}

	// Fast-path: check the cache for whether this ExtensionType has already
	// been converted to an ExtensionInfo.
	if d, ok := legacyExtensionInfoCache.Load(xt); ok {
		return d.(*ExtensionInfo)
	}

	tt := xt.GoType()
	if xt.TypeDescriptor().Cardinality() == pref.Repeated {
		tt = tt.Elem().Elem()
	}
	xi := &ExtensionInfo{}
	InitExtensionInfo(xi, xt.TypeDescriptor().Descriptor(), tt)
	xi.lazyInit() // populate legacy fields

	if xi, ok := legacyExtensionInfoCache.LoadOrStore(xt, xi); ok {
		return xi.(*ExtensionInfo)
	}
	return xi
}

func (xi *ExtensionInfo) initToLegacy() {
	xd := xi.desc
	var parent piface.MessageV1
	messageName := xd.ContainingMessage().FullName()
	if mt, _ := preg.GlobalTypes.FindMessageByName(messageName); mt != nil {
		// Create a new parent message and unwrap it if possible.
		mv := mt.New().Interface()
		t := reflect.TypeOf(mv)
		if mv, ok := mv.(unwrapper); ok {
			t = reflect.TypeOf(mv.protoUnwrap())
		}

		// Check whether the message implements the legacy v1 Message interface.
		mz := reflect.Zero(t).Interface()
		if mz, ok := mz.(piface.MessageV1); ok {
			parent = mz
		}
	}

	// Determine the v1 extension type, which is unfortunately not the same as
	// the v2 ExtensionType.GoType.
	extType := xi.goType
	switch extType.Kind() {
	case reflect.Bool, reflect.Int32, reflect.Int64, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64, reflect.String:
		extType = reflect.PtrTo(extType) // T -> *T for singular scalar fields
	}

	// Reconstruct the legacy enum full name.
	var enumName string
	if xd.Kind() == pref.EnumKind {
		enumName = legacyEnumName(xd.Enum())
	}

	// Derive the proto file that the extension was declared within.
	var filename string
	if fd := xd.ParentFile(); fd != nil {
		filename = fd.Path()
	}

	// For MessageSet extensions, the name used is the parent message.
	name := xd.FullName()
	if messageset.IsMessageSetExtension(xd) {
		name = name.Parent()
	}

	xi.ExtendedType = parent
	xi.ExtensionType = reflect.Zero(extType).Interface()
	xi.Field = int32(xd.Number())
	xi.Name = string(name)
	xi.Tag = ptag.Marshal(xd, enumName)
	xi.Filename = filename
}

// initFromLegacy initializes an ExtensionInfo from
// the contents of the deprecated exported fields of the type.
func (xi *ExtensionInfo) initFromLegacy() {
	// Resolve enum or message dependencies.
	var ed pref.EnumDescriptor
	var md pref.MessageDescriptor
	t := reflect.TypeOf(xi.ExtensionType)
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
	fd := ptag.Unmarshal(xi.Tag, t, evs).(*filedesc.Field)

	// Construct a v2 ExtensionType.
	xd := &filedesc.Extension{L2: new(filedesc.ExtensionL2)}
	xd.L0.ParentFile = filedesc.SurrogateProto2
	xd.L0.FullName = pref.FullName(xi.Name)
	xd.L1.Number = pref.FieldNumber(xi.Field)
	xd.L2.Cardinality = fd.L1.Cardinality
	xd.L1.Kind = fd.L1.Kind
	xd.L2.IsPacked = fd.L1.IsPacked
	xd.L2.Default = fd.L1.Default
	xd.L1.Extendee = Export{}.MessageDescriptorOf(xi.ExtendedType)
	xd.L2.Enum = ed
	xd.L2.Message = md

	// Derive real extension field name for MessageSets.
	if messageset.IsMessageSet(xd.L1.Extendee) && md.FullName() == xd.L0.FullName {
		xd.L0.FullName = xd.L0.FullName.Append(messageset.ExtensionName)
	}

	tt := reflect.TypeOf(xi.ExtensionType)
	if isOptional {
		tt = tt.Elem()
	}
	xi.desc = xd
	xi.goType = tt
}
