// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	pref "google.golang.org/protobuf/reflect/protoreflect"
	piface "google.golang.org/protobuf/runtime/protoiface"
)

// MessageInfo provides protobuf related functionality for a given Go type
// that represents a message. A given instance of MessageInfo is tied to
// exactly one Go type, which must be a pointer to a struct type.
type MessageInfo struct {
	// GoType is the underlying message Go type and must be populated.
	// Once set, this field must never be mutated.
	GoType reflect.Type // pointer to struct

	// PBType is the underlying message descriptor type and must be populated.
	// Once set, this field must never be mutated.
	PBType pref.MessageType

	// Exporter must be provided in a purego environment in order to provide
	// access to unexported fields.
	Exporter exporter

	// OneofWrappers is list of pointers to oneof wrapper struct types.
	OneofWrappers []interface{}

	initMu   sync.Mutex // protects all unexported fields
	initDone uint32

	reflectMessageInfo

	// Information used by the fast-path methods.
	methods piface.Methods
	coderMessageInfo

	extensionFieldInfosMu sync.RWMutex
	extensionFieldInfos   map[pref.ExtensionType]*extensionFieldInfo
}

type reflectMessageInfo struct {
	fields map[pref.FieldNumber]*fieldInfo
	oneofs map[pref.Name]*oneofInfo

	getUnknown   func(pointer) pref.RawFields
	setUnknown   func(pointer, pref.RawFields)
	extensionMap func(pointer) *extensionMap

	nilMessage atomicNilMessage
}

// exporter is a function that returns a reference to the ith field of v,
// where v is a pointer to a struct. It returns nil if it does not support
// exporting the requested field (e.g., already exported).
type exporter func(v interface{}, i int) interface{}

var prefMessageType = reflect.TypeOf((*pref.Message)(nil)).Elem()

// getMessageInfo returns the MessageInfo (if any) for a type.
//
// We find the MessageInfo by calling the ProtoReflect method on the type's
// zero value and looking at the returned type to see if it is a
// messageReflectWrapper. Note that the MessageInfo may still be uninitialized
// at this point.
func getMessageInfo(mt reflect.Type) (mi *MessageInfo, ok bool) {
	method, ok := mt.MethodByName("ProtoReflect")
	if !ok {
		return nil, false
	}
	if method.Type.NumIn() != 1 || method.Type.NumOut() != 1 || method.Type.Out(0) != prefMessageType {
		return nil, false
	}
	ret := reflect.Zero(mt).Method(method.Index).Call(nil)
	m, ok := ret[0].Elem().Interface().(*messageReflectWrapper)
	if !ok {
		return nil, ok
	}
	return m.mi, true
}

func (mi *MessageInfo) init() {
	// This function is called in the hot path. Inline the sync.Once
	// logic, since allocating a closure for Once.Do is expensive.
	// Keep init small to ensure that it can be inlined.
	if atomic.LoadUint32(&mi.initDone) == 0 {
		mi.initOnce()
	}
}

func (mi *MessageInfo) initOnce() {
	mi.initMu.Lock()
	defer mi.initMu.Unlock()
	if mi.initDone == 1 {
		return
	}

	t := mi.GoType
	if t.Kind() != reflect.Ptr && t.Elem().Kind() != reflect.Struct {
		panic(fmt.Sprintf("got %v, want *struct kind", t))
	}

	si := mi.makeStructInfo(t.Elem())
	mi.makeKnownFieldsFunc(si)
	mi.makeUnknownFieldsFunc(t.Elem(), si)
	mi.makeExtensionFieldsFunc(t.Elem(), si)
	mi.makeMethods(t.Elem(), si)

	atomic.StoreUint32(&mi.initDone, 1)
}

type (
	SizeCache       = int32
	UnknownFields   = []byte
	ExtensionFields = map[int32]ExtensionField
)

var (
	sizecacheType       = reflect.TypeOf(SizeCache(0))
	unknownFieldsType   = reflect.TypeOf(UnknownFields(nil))
	extensionFieldsType = reflect.TypeOf(ExtensionFields(nil))
)

type structInfo struct {
	sizecacheOffset offset
	unknownOffset   offset
	extensionOffset offset

	fieldsByNumber        map[pref.FieldNumber]reflect.StructField
	oneofsByName          map[pref.Name]reflect.StructField
	oneofWrappersByType   map[reflect.Type]pref.FieldNumber
	oneofWrappersByNumber map[pref.FieldNumber]reflect.Type
}

func (mi *MessageInfo) makeStructInfo(t reflect.Type) structInfo {
	si := structInfo{
		sizecacheOffset: invalidOffset,
		unknownOffset:   invalidOffset,
		extensionOffset: invalidOffset,

		fieldsByNumber:        map[pref.FieldNumber]reflect.StructField{},
		oneofsByName:          map[pref.Name]reflect.StructField{},
		oneofWrappersByType:   map[reflect.Type]pref.FieldNumber{},
		oneofWrappersByNumber: map[pref.FieldNumber]reflect.Type{},
	}

	if f, _ := t.FieldByName("sizeCache"); f.Type == sizecacheType {
		si.sizecacheOffset = offsetOf(f, mi.Exporter)
	}
	if f, _ := t.FieldByName("XXX_sizecache"); f.Type == sizecacheType {
		si.sizecacheOffset = offsetOf(f, mi.Exporter)
	}
	if f, _ := t.FieldByName("unknownFields"); f.Type == unknownFieldsType {
		si.unknownOffset = offsetOf(f, mi.Exporter)
	}
	if f, _ := t.FieldByName("XXX_unrecognized"); f.Type == unknownFieldsType {
		si.unknownOffset = offsetOf(f, mi.Exporter)
	}
	if f, _ := t.FieldByName("extensionFields"); f.Type == extensionFieldsType {
		si.extensionOffset = offsetOf(f, mi.Exporter)
	}
	if f, _ := t.FieldByName("XXX_InternalExtensions"); f.Type == extensionFieldsType {
		si.extensionOffset = offsetOf(f, mi.Exporter)
	}
	if f, _ := t.FieldByName("XXX_extensions"); f.Type == extensionFieldsType {
		si.extensionOffset = offsetOf(f, mi.Exporter)
	}

	// Generate a mapping of field numbers and names to Go struct field or type.
fieldLoop:
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		for _, s := range strings.Split(f.Tag.Get("protobuf"), ",") {
			if len(s) > 0 && strings.Trim(s, "0123456789") == "" {
				n, _ := strconv.ParseUint(s, 10, 64)
				si.fieldsByNumber[pref.FieldNumber(n)] = f
				continue fieldLoop
			}
		}
		if s := f.Tag.Get("protobuf_oneof"); len(s) > 0 {
			si.oneofsByName[pref.Name(s)] = f
			continue fieldLoop
		}
	}

	// Derive a mapping of oneof wrappers to fields.
	oneofWrappers := mi.OneofWrappers
	if fn, ok := reflect.PtrTo(t).MethodByName("XXX_OneofFuncs"); ok {
		oneofWrappers = fn.Func.Call([]reflect.Value{reflect.Zero(fn.Type.In(0))})[3].Interface().([]interface{})
	}
	if fn, ok := reflect.PtrTo(t).MethodByName("XXX_OneofWrappers"); ok {
		oneofWrappers = fn.Func.Call([]reflect.Value{reflect.Zero(fn.Type.In(0))})[0].Interface().([]interface{})
	}
	for _, v := range oneofWrappers {
		tf := reflect.TypeOf(v).Elem()
		f := tf.Field(0)
		for _, s := range strings.Split(f.Tag.Get("protobuf"), ",") {
			if len(s) > 0 && strings.Trim(s, "0123456789") == "" {
				n, _ := strconv.ParseUint(s, 10, 64)
				si.oneofWrappersByType[tf] = pref.FieldNumber(n)
				si.oneofWrappersByNumber[pref.FieldNumber(n)] = tf
				break
			}
		}
	}

	return si
}

// makeKnownFieldsFunc generates functions for operations that can be performed
// on each protobuf message field. It takes in a reflect.Type representing the
// Go struct and matches message fields with struct fields.
//
// This code assumes that the struct is well-formed and panics if there are
// any discrepancies.
func (mi *MessageInfo) makeKnownFieldsFunc(si structInfo) {
	mi.fields = map[pref.FieldNumber]*fieldInfo{}
	for i := 0; i < mi.PBType.Descriptor().Fields().Len(); i++ {
		fd := mi.PBType.Descriptor().Fields().Get(i)
		fs := si.fieldsByNumber[fd.Number()]
		var fi fieldInfo
		switch {
		case fd.ContainingOneof() != nil:
			fi = fieldInfoForOneof(fd, si.oneofsByName[fd.ContainingOneof().Name()], mi.Exporter, si.oneofWrappersByNumber[fd.Number()])
		case fd.IsMap():
			fi = fieldInfoForMap(fd, fs, mi.Exporter)
		case fd.IsList():
			fi = fieldInfoForList(fd, fs, mi.Exporter)
		case fd.Kind() == pref.MessageKind || fd.Kind() == pref.GroupKind:
			fi = fieldInfoForMessage(fd, fs, mi.Exporter)
		default:
			fi = fieldInfoForScalar(fd, fs, mi.Exporter)
		}
		mi.fields[fd.Number()] = &fi
	}

	mi.oneofs = map[pref.Name]*oneofInfo{}
	for i := 0; i < mi.PBType.Descriptor().Oneofs().Len(); i++ {
		od := mi.PBType.Descriptor().Oneofs().Get(i)
		mi.oneofs[od.Name()] = makeOneofInfo(od, si.oneofsByName[od.Name()], mi.Exporter, si.oneofWrappersByType)
	}
}

func (mi *MessageInfo) makeUnknownFieldsFunc(t reflect.Type, si structInfo) {
	mi.getUnknown = func(pointer) pref.RawFields { return nil }
	mi.setUnknown = func(pointer, pref.RawFields) { return }
	if si.unknownOffset.IsValid() {
		mi.getUnknown = func(p pointer) pref.RawFields {
			if p.IsNil() {
				return nil
			}
			rv := p.Apply(si.unknownOffset).AsValueOf(unknownFieldsType)
			return pref.RawFields(*rv.Interface().(*[]byte))
		}
		mi.setUnknown = func(p pointer, b pref.RawFields) {
			if p.IsNil() {
				panic("invalid SetUnknown on nil Message")
			}
			rv := p.Apply(si.unknownOffset).AsValueOf(unknownFieldsType)
			*rv.Interface().(*[]byte) = []byte(b)
		}
	} else {
		mi.getUnknown = func(pointer) pref.RawFields {
			return nil
		}
		mi.setUnknown = func(p pointer, _ pref.RawFields) {
			if p.IsNil() {
				panic("invalid SetUnknown on nil Message")
			}
		}
	}
}

func (mi *MessageInfo) makeExtensionFieldsFunc(t reflect.Type, si structInfo) {
	if si.extensionOffset.IsValid() {
		mi.extensionMap = func(p pointer) *extensionMap {
			if p.IsNil() {
				return (*extensionMap)(nil)
			}
			v := p.Apply(si.extensionOffset).AsValueOf(extensionFieldsType)
			return (*extensionMap)(v.Interface().(*map[int32]ExtensionField))
		}
	} else {
		mi.extensionMap = func(pointer) *extensionMap {
			return (*extensionMap)(nil)
		}
	}
}

// TODO: Move this to be on the reflect message instance.
func (mi *MessageInfo) Methods() *piface.Methods {
	mi.init()
	return &mi.methods
}
