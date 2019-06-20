// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"reflect"
	"sort"

	"google.golang.org/protobuf/internal/encoding/wire"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	piface "google.golang.org/protobuf/runtime/protoiface"
)

// coderMessageInfo contains per-message information used by the fast-path functions.
// This is a different type from MessageInfo to keep MessageInfo as general-purpose as
// possible.
type coderMessageInfo struct {
	orderedCoderFields []*coderFieldInfo
	sizecacheOffset    offset
	extensionOffset    offset
	unknownOffset      offset
	needsInitCheck     bool
}

type coderFieldInfo struct {
	funcs      pointerCoderFuncs // fast-path per-field functions
	num        pref.FieldNumber  // field number
	offset     offset            // struct field offset
	wiretag    uint64            // field tag (number + wire type)
	tagsize    int               // size of the varint-encoded tag
	isPointer  bool              // true if IsNil may be called on the struct field
	isRequired bool              // true if field is required
}

func (mi *MessageInfo) makeMethods(t reflect.Type, si structInfo) {
	mi.sizecacheOffset = invalidOffset
	if fx, _ := t.FieldByName("XXX_sizecache"); fx.Type == sizecacheType {
		mi.sizecacheOffset = offsetOf(fx)
	}
	mi.unknownOffset = invalidOffset
	if fx, _ := t.FieldByName("XXX_unrecognized"); fx.Type == unknownFieldsType {
		mi.unknownOffset = offsetOf(fx)
	}
	mi.extensionOffset = invalidOffset
	if fx, _ := t.FieldByName("XXX_InternalExtensions"); fx.Type == extensionFieldsType {
		mi.extensionOffset = offsetOf(fx)
	} else if fx, _ = t.FieldByName("XXX_extensions"); fx.Type == extensionFieldsType {
		mi.extensionOffset = offsetOf(fx)
	}

	for i := 0; i < mi.PBType.Descriptor().Fields().Len(); i++ {
		fd := mi.PBType.Descriptor().Fields().Get(i)
		if fd.ContainingOneof() != nil {
			continue
		}

		fs := si.fieldsByNumber[fd.Number()]
		ft := fs.Type
		var wiretag uint64
		if !fd.IsPacked() {
			wiretag = wire.EncodeTag(fd.Number(), wireTypes[fd.Kind()])
		} else {
			wiretag = wire.EncodeTag(fd.Number(), wire.BytesType)
		}
		mi.orderedCoderFields = append(mi.orderedCoderFields, &coderFieldInfo{
			num:     fd.Number(),
			offset:  offsetOf(fs),
			wiretag: wiretag,
			tagsize: wire.SizeVarint(wiretag),
			funcs:   fieldCoder(fd, ft),
			isPointer: (fd.Cardinality() == pref.Repeated ||
				fd.Kind() == pref.MessageKind ||
				fd.Kind() == pref.GroupKind ||
				fd.Syntax() != pref.Proto3),
			isRequired: fd.Cardinality() == pref.Required,
		})
	}
	for i := 0; i < mi.PBType.Descriptor().Oneofs().Len(); i++ {
		od := mi.PBType.Descriptor().Oneofs().Get(i)
		fs := si.oneofsByName[od.Name()]
		mi.orderedCoderFields = append(mi.orderedCoderFields, &coderFieldInfo{
			num:       od.Fields().Get(0).Number(),
			offset:    offsetOf(fs),
			funcs:     makeOneofFieldCoder(fs, od, si.fieldsByNumber, si.oneofWrappersByNumber),
			isPointer: true,
		})
	}
	sort.Slice(mi.orderedCoderFields, func(i, j int) bool {
		return mi.orderedCoderFields[i].num < mi.orderedCoderFields[j].num
	})

	mi.needsInitCheck = needsInitCheck(mi.PBType)
	mi.methods = piface.Methods{
		Flags:         piface.MethodFlagDeterministicMarshal,
		MarshalAppend: mi.marshalAppend,
		Size:          mi.size,
		IsInitialized: mi.isInitialized,
	}
}
