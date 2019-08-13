// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"reflect"
	"sort"
	"sync"

	"google.golang.org/protobuf/internal/encoding/messageset"
	"google.golang.org/protobuf/internal/encoding/wire"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	piface "google.golang.org/protobuf/runtime/protoiface"
)

// coderMessageInfo contains per-message information used by the fast-path functions.
// This is a different type from MessageInfo to keep MessageInfo as general-purpose as
// possible.
type coderMessageInfo struct {
	methods piface.Methods

	orderedCoderFields []*coderFieldInfo
	denseCoderFields   []*coderFieldInfo
	coderFields        map[wire.Number]*coderFieldInfo
	sizecacheOffset    offset
	unknownOffset      offset
	extensionOffset    offset
	needsInitCheck     bool

	extensionFieldInfosMu sync.RWMutex
	extensionFieldInfos   map[pref.ExtensionType]*extensionFieldInfo
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

func (mi *MessageInfo) makeCoderMethods(t reflect.Type, si structInfo) {
	mi.sizecacheOffset = si.sizecacheOffset
	mi.unknownOffset = si.unknownOffset
	mi.extensionOffset = si.extensionOffset

	mi.coderFields = make(map[wire.Number]*coderFieldInfo)
	fields := mi.Desc.Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)

		fs := si.fieldsByNumber[fd.Number()]
		if fd.ContainingOneof() != nil {
			fs = si.oneofsByName[fd.ContainingOneof().Name()]
		}
		ft := fs.Type
		var wiretag uint64
		if !fd.IsPacked() {
			wiretag = wire.EncodeTag(fd.Number(), wireTypes[fd.Kind()])
		} else {
			wiretag = wire.EncodeTag(fd.Number(), wire.BytesType)
		}
		var funcs pointerCoderFuncs
		if fd.ContainingOneof() != nil {
			funcs = makeOneofFieldCoder(si, fd)
		} else {
			funcs = fieldCoder(fd, ft)
		}
		cf := &coderFieldInfo{
			num:     fd.Number(),
			offset:  offsetOf(fs, mi.Exporter),
			wiretag: wiretag,
			tagsize: wire.SizeVarint(wiretag),
			funcs:   funcs,
			isPointer: (fd.Cardinality() == pref.Repeated ||
				fd.Kind() == pref.MessageKind ||
				fd.Kind() == pref.GroupKind ||
				fd.Syntax() != pref.Proto3),
			isRequired: fd.Cardinality() == pref.Required,
		}
		mi.orderedCoderFields = append(mi.orderedCoderFields, cf)
		mi.coderFields[cf.num] = cf
	}
	if messageset.IsMessageSet(mi.Desc) {
		if !mi.extensionOffset.IsValid() {
			panic(fmt.Sprintf("%v: MessageSet with no extensions field", mi.Desc.FullName()))
		}
		cf := &coderFieldInfo{
			num:       messageset.FieldItem,
			offset:    si.extensionOffset,
			isPointer: true,
			funcs:     makeMessageSetFieldCoder(mi),
		}
		mi.orderedCoderFields = append(mi.orderedCoderFields, cf)
		mi.coderFields[cf.num] = cf
		// Invalidate the extension offset, since the field codec handles extensions.
		mi.extensionOffset = invalidOffset
	}
	sort.Slice(mi.orderedCoderFields, func(i, j int) bool {
		return mi.orderedCoderFields[i].num < mi.orderedCoderFields[j].num
	})

	var maxDense pref.FieldNumber
	for _, cf := range mi.orderedCoderFields {
		if cf.num >= 16 && cf.num >= 2*maxDense {
			break
		}
		maxDense = cf.num
	}
	mi.denseCoderFields = make([]*coderFieldInfo, maxDense+1)
	for _, cf := range mi.orderedCoderFields {
		if int(cf.num) > len(mi.denseCoderFields) {
			break
		}
		mi.denseCoderFields[cf.num] = cf
	}

	mi.needsInitCheck = needsInitCheck(mi.Desc)
	mi.methods = piface.Methods{
		Flags:         piface.SupportMarshalDeterministic | piface.SupportUnmarshalDiscardUnknown,
		MarshalAppend: mi.marshalAppend,
		Unmarshal:     mi.unmarshal,
		Size:          mi.size,
		IsInitialized: mi.isInitialized,
	}
}
