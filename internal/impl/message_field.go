// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"math"
	"reflect"
	"sync"

	"google.golang.org/protobuf/internal/flags"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	preg "google.golang.org/protobuf/reflect/protoregistry"
	piface "google.golang.org/protobuf/runtime/protoiface"
)

type fieldInfo struct {
	fieldDesc pref.FieldDescriptor

	// These fields are used for protobuf reflection support.
	has        func(pointer) bool
	clear      func(pointer)
	get        func(pointer) pref.Value
	set        func(pointer, pref.Value)
	mutable    func(pointer) pref.Value
	newMessage func() pref.Message
}

func fieldInfoForOneof(fd pref.FieldDescriptor, fs reflect.StructField, x exporter, ot reflect.Type) fieldInfo {
	ft := fs.Type
	if ft.Kind() != reflect.Interface {
		panic(fmt.Sprintf("invalid type: got %v, want interface kind", ft))
	}
	if ot.Kind() != reflect.Struct {
		panic(fmt.Sprintf("invalid type: got %v, want struct kind", ot))
	}
	if !reflect.PtrTo(ot).Implements(ft) {
		panic(fmt.Sprintf("invalid type: %v does not implement %v", ot, ft))
	}
	conv := NewConverter(ot.Field(0).Type, fd)
	isMessage := fd.Message() != nil
	var frozenEmpty pref.Value
	if isMessage {
		frozenEmpty = pref.ValueOf(frozenMessage{conv.New().Message()})
	}

	// TODO: Implement unsafe fast path?
	fieldOffset := offsetOf(fs, x)
	return fieldInfo{
		// NOTE: The logic below intentionally assumes that oneof fields are
		// well-formatted. That is, the oneof interface never contains a
		// typed nil pointer to one of the wrapper structs.

		fieldDesc: fd,
		has: func(p pointer) bool {
			if p.IsNil() {
				return false
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if rv.IsNil() || rv.Elem().Type().Elem() != ot {
				return false
			}
			return true
		},
		clear: func(p pointer) {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if rv.IsNil() || rv.Elem().Type().Elem() != ot {
				return
			}
			rv.Set(reflect.Zero(rv.Type()))
		},
		get: func(p pointer) pref.Value {
			if p.IsNil() {
				if frozenEmpty.IsValid() {
					return frozenEmpty
				}
				return defaultValueOf(fd)
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if rv.IsNil() || rv.Elem().Type().Elem() != ot {
				if frozenEmpty.IsValid() {
					return frozenEmpty
				}
				return defaultValueOf(fd)
			}
			rv = rv.Elem().Elem().Field(0)
			return conv.PBValueOf(rv)
		},
		set: func(p pointer, v pref.Value) {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if rv.IsNil() || rv.Elem().Type().Elem() != ot {
				rv.Set(reflect.New(ot))
			}
			rv = rv.Elem().Elem().Field(0)
			rv.Set(conv.GoValueOf(v))
		},
		mutable: func(p pointer) pref.Value {
			if !isMessage {
				panic("invalid Mutable on field with non-composite type")
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if rv.IsNil() || rv.Elem().Type().Elem() != ot {
				rv.Set(reflect.New(ot))
			}
			rv = rv.Elem().Elem().Field(0)
			if rv.IsNil() {
				rv.Set(conv.GoValueOf(pref.ValueOf(conv.New().Message())))
			}
			return conv.PBValueOf(rv)
		},
		newMessage: func() pref.Message {
			return conv.New().Message()
		},
	}
}

func fieldInfoForMap(fd pref.FieldDescriptor, fs reflect.StructField, x exporter) fieldInfo {
	ft := fs.Type
	if ft.Kind() != reflect.Map {
		panic(fmt.Sprintf("invalid type: got %v, want map kind", ft))
	}
	conv := NewConverter(ft, fd)
	frozenEmpty := pref.ValueOf(frozenMap{conv.New().Map()})

	// TODO: Implement unsafe fast path?
	fieldOffset := offsetOf(fs, x)
	return fieldInfo{
		fieldDesc: fd,
		has: func(p pointer) bool {
			if p.IsNil() {
				return false
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			return rv.Len() > 0
		},
		clear: func(p pointer) {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			rv.Set(reflect.Zero(rv.Type()))
		},
		get: func(p pointer) pref.Value {
			if p.IsNil() {
				return frozenEmpty
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if rv.Len() == 0 {
				return frozenEmpty
			}
			return conv.PBValueOf(rv)
		},
		set: func(p pointer, v pref.Value) {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			rv.Set(conv.GoValueOf(v))
		},
		mutable: func(p pointer) pref.Value {
			v := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if v.IsNil() {
				v.Set(reflect.MakeMap(fs.Type))
			}
			return conv.PBValueOf(v)
		},
	}
}

func fieldInfoForList(fd pref.FieldDescriptor, fs reflect.StructField, x exporter) fieldInfo {
	ft := fs.Type
	if ft.Kind() != reflect.Slice {
		panic(fmt.Sprintf("invalid type: got %v, want slice kind", ft))
	}
	conv := NewConverter(reflect.PtrTo(ft), fd)
	frozenEmpty := pref.ValueOf(frozenList{conv.New().List()})

	// TODO: Implement unsafe fast path?
	fieldOffset := offsetOf(fs, x)
	return fieldInfo{
		fieldDesc: fd,
		has: func(p pointer) bool {
			if p.IsNil() {
				return false
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			return rv.Len() > 0
		},
		clear: func(p pointer) {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			rv.Set(reflect.Zero(rv.Type()))
		},
		get: func(p pointer) pref.Value {
			if p.IsNil() {
				return frozenEmpty
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type)
			if rv.Elem().Len() == 0 {
				return frozenEmpty
			}
			return conv.PBValueOf(rv)
		},
		set: func(p pointer, v pref.Value) {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			rv.Set(reflect.ValueOf(v.List().(Unwrapper).ProtoUnwrap()).Elem())
		},
		mutable: func(p pointer) pref.Value {
			v := p.Apply(fieldOffset).AsValueOf(fs.Type)
			return conv.PBValueOf(v)
		},
	}
}

var (
	nilBytes   = reflect.ValueOf([]byte(nil))
	emptyBytes = reflect.ValueOf([]byte{})
)

func fieldInfoForScalar(fd pref.FieldDescriptor, fs reflect.StructField, x exporter) fieldInfo {
	ft := fs.Type
	nullable := fd.Syntax() == pref.Proto2
	isBytes := ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Uint8
	if nullable {
		if ft.Kind() != reflect.Ptr && ft.Kind() != reflect.Slice {
			panic(fmt.Sprintf("invalid type: got %v, want pointer", ft))
		}
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
	}
	conv := NewConverter(ft, fd)

	// TODO: Implement unsafe fast path?
	fieldOffset := offsetOf(fs, x)
	return fieldInfo{
		fieldDesc: fd,
		has: func(p pointer) bool {
			if p.IsNil() {
				return false
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if nullable {
				return !rv.IsNil()
			}
			switch rv.Kind() {
			case reflect.Bool:
				return rv.Bool()
			case reflect.Int32, reflect.Int64:
				return rv.Int() != 0
			case reflect.Uint32, reflect.Uint64:
				return rv.Uint() != 0
			case reflect.Float32, reflect.Float64:
				return rv.Float() != 0 || math.Signbit(rv.Float())
			case reflect.String, reflect.Slice:
				return rv.Len() > 0
			default:
				panic(fmt.Sprintf("invalid type: %v", rv.Type())) // should never happen
			}
		},
		clear: func(p pointer) {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			rv.Set(reflect.Zero(rv.Type()))
		},
		get: func(p pointer) pref.Value {
			if p.IsNil() {
				return defaultValueOf(fd)
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if nullable {
				if rv.IsNil() {
					return defaultValueOf(fd)
				}
				if rv.Kind() == reflect.Ptr {
					rv = rv.Elem()
				}
			}
			return conv.PBValueOf(rv)
		},
		set: func(p pointer, v pref.Value) {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if nullable && rv.Kind() == reflect.Ptr {
				if rv.IsNil() {
					rv.Set(reflect.New(ft))
				}
				rv = rv.Elem()
			}
			rv.Set(conv.GoValueOf(v))
			if isBytes && rv.Len() == 0 {
				if nullable {
					rv.Set(emptyBytes) // preserve presence in proto2
				} else {
					rv.Set(nilBytes) // do not preserve presence in proto3
				}
			}
		},
	}
}

func fieldInfoForWeakMessage(fd pref.FieldDescriptor, weakOffset offset) fieldInfo {
	if !flags.Proto1Legacy {
		panic("no support for proto1 weak fields")
	}

	var once sync.Once
	var messageType pref.MessageType
	var frozenEmpty pref.Value
	lazyInit := func() {
		once.Do(func() {
			messageName := fd.Message().FullName()
			messageType, _ = preg.GlobalTypes.FindMessageByName(messageName)
			if messageType == nil {
				panic(fmt.Sprintf("weak message %v is not linked in", messageName))
			}
			frozenEmpty = pref.ValueOf(frozenMessage{messageType.New()})
		})
	}

	num := int32(fd.Number())
	return fieldInfo{
		fieldDesc: fd,
		has: func(p pointer) bool {
			if p.IsNil() {
				return false
			}
			fs := p.Apply(weakOffset).WeakFields()
			_, ok := (*fs)[num]
			return ok
		},
		clear: func(p pointer) {
			fs := p.Apply(weakOffset).WeakFields()
			delete(*fs, num)
		},
		get: func(p pointer) pref.Value {
			lazyInit()
			if p.IsNil() {
				return frozenEmpty
			}
			fs := p.Apply(weakOffset).WeakFields()
			m, ok := (*fs)[num]
			if !ok {
				return frozenEmpty
			}
			return pref.ValueOf(m.(pref.ProtoMessage).ProtoReflect())
		},
		set: func(p pointer, v pref.Value) {
			lazyInit()
			m := v.Message()
			if m.Descriptor() != messageType.Descriptor() {
				panic("mismatching message descriptor")
			}
			fs := p.Apply(weakOffset).WeakFields()
			if *fs == nil {
				*fs = make(WeakFields)
			}
			(*fs)[num] = m.Interface().(piface.MessageV1)
		},
		mutable: func(p pointer) pref.Value {
			lazyInit()
			fs := p.Apply(weakOffset).WeakFields()
			if *fs == nil {
				*fs = make(WeakFields)
			}
			m, ok := (*fs)[num]
			if !ok {
				m = messageType.New().Interface().(piface.MessageV1)
				(*fs)[num] = m
			}
			return pref.ValueOf(m.(pref.ProtoMessage).ProtoReflect())
		},
		newMessage: func() pref.Message {
			lazyInit()
			return messageType.New()
		},
	}
}

func fieldInfoForMessage(fd pref.FieldDescriptor, fs reflect.StructField, x exporter) fieldInfo {
	ft := fs.Type
	conv := NewConverter(ft, fd)
	frozenEmpty := pref.ValueOf(frozenMessage{conv.New().Message()})

	// TODO: Implement unsafe fast path?
	fieldOffset := offsetOf(fs, x)
	return fieldInfo{
		fieldDesc: fd,
		has: func(p pointer) bool {
			if p.IsNil() {
				return false
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			return !rv.IsNil()
		},
		clear: func(p pointer) {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			rv.Set(reflect.Zero(rv.Type()))
		},
		get: func(p pointer) pref.Value {
			if p.IsNil() {
				return frozenEmpty
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if rv.IsNil() {
				return frozenEmpty
			}
			return conv.PBValueOf(rv)
		},
		set: func(p pointer, v pref.Value) {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			rv.Set(conv.GoValueOf(v))
			if rv.IsNil() {
				panic("invalid nil pointer")
			}
		},
		mutable: func(p pointer) pref.Value {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if rv.IsNil() {
				rv.Set(conv.GoValueOf(conv.New()))
			}
			return conv.PBValueOf(rv)
		},
		newMessage: func() pref.Message {
			return conv.New().Message()
		},
	}
}

type oneofInfo struct {
	oneofDesc pref.OneofDescriptor
	which     func(pointer) pref.FieldNumber
}

func makeOneofInfo(od pref.OneofDescriptor, fs reflect.StructField, x exporter, wrappersByType map[reflect.Type]pref.FieldNumber) *oneofInfo {
	fieldOffset := offsetOf(fs, x)
	return &oneofInfo{
		oneofDesc: od,
		which: func(p pointer) pref.FieldNumber {
			if p.IsNil() {
				return 0
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if rv.IsNil() {
				return 0
			}
			return wrappersByType[rv.Elem().Type().Elem()]
		},
	}
}

// defaultValueOf returns the default value for the field.
func defaultValueOf(fd pref.FieldDescriptor) pref.Value {
	if fd == nil {
		return pref.Value{}
	}
	pv := fd.Default() // invalid Value for messages and repeated fields
	if fd.Kind() == pref.BytesKind && pv.IsValid() && len(pv.Bytes()) > 0 {
		return pref.ValueOf(append([]byte(nil), pv.Bytes()...)) // copy default bytes for safety
	}
	return pv
}

// frozenValueOf returns a frozen version of any composite value.
func frozenValueOf(v pref.Value) pref.Value {
	switch v := v.Interface().(type) {
	case pref.Message:
		if _, ok := v.(frozenMessage); !ok {
			return pref.ValueOf(frozenMessage{v})
		}
	case pref.List:
		if _, ok := v.(frozenList); !ok {
			return pref.ValueOf(frozenList{v})
		}
	case pref.Map:
		if _, ok := v.(frozenMap); !ok {
			return pref.ValueOf(frozenMap{v})
		}
	}
	return v
}

type frozenMessage struct{ pref.Message }

func (m frozenMessage) ProtoReflect() pref.Message   { return m }
func (m frozenMessage) Interface() pref.ProtoMessage { return m }
func (m frozenMessage) Range(f func(pref.FieldDescriptor, pref.Value) bool) {
	m.Message.Range(func(fd pref.FieldDescriptor, v pref.Value) bool {
		return f(fd, frozenValueOf(v))
	})
}
func (m frozenMessage) Get(fd pref.FieldDescriptor) pref.Value {
	v := m.Message.Get(fd)
	return frozenValueOf(v)
}
func (frozenMessage) Clear(pref.FieldDescriptor)              { panic("invalid on read-only Message") }
func (frozenMessage) Set(pref.FieldDescriptor, pref.Value)    { panic("invalid on read-only Message") }
func (frozenMessage) Mutable(pref.FieldDescriptor) pref.Value { panic("invalid on read-only Message") }
func (frozenMessage) SetUnknown(pref.RawFields)               { panic("invalid on read-only Message") }

type frozenList struct{ pref.List }

func (ls frozenList) Get(i int) pref.Value {
	v := ls.List.Get(i)
	return frozenValueOf(v)
}
func (frozenList) Set(i int, v pref.Value) { panic("invalid on read-only List") }
func (frozenList) Append(v pref.Value)     { panic("invalid on read-only List") }
func (frozenList) Truncate(i int)          { panic("invalid on read-only List") }

type frozenMap struct{ pref.Map }

func (ms frozenMap) Get(k pref.MapKey) pref.Value {
	v := ms.Map.Get(k)
	return frozenValueOf(v)
}
func (ms frozenMap) Range(f func(pref.MapKey, pref.Value) bool) {
	ms.Map.Range(func(k pref.MapKey, v pref.Value) bool {
		return f(k, frozenValueOf(v))
	})
}
func (frozenMap) Set(k pref.MapKey, v pref.Value) { panic("invalid n read-only Map") }
func (frozenMap) Clear(k pref.MapKey)             { panic("invalid on read-only Map") }
