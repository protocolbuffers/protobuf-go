// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"reflect"

	"google.golang.org/protobuf/internal/pragma"
	pvalue "google.golang.org/protobuf/internal/value"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	piface "google.golang.org/protobuf/runtime/protoiface"
)

// MessageState is a data structure that is nested as the first field in a
// concrete message. It provides a way to implement the ProtoReflect method
// in an allocation-free way without needing to have a shadow Go type generated
// for every message type. This technique only works using unsafe.
//
//
// Example generated code:
//
//	type M struct {
//		state protoimpl.MessageState
//
//		Field1 int32
//		Field2 string
//		Field3 *BarMessage
//		...
//	}
//
//	func (m *M) ProtoReflect() protoreflect.Message {
//		mi := &file_fizz_buzz_proto_msgInfos[5]
//		if protoimpl.UnsafeEnabled && m != nil {
//			ms := protoimpl.X.MessageStateOf(Pointer(m))
//			if ms.LoadMessageInfo() == nil {
//				ms.StoreMessageInfo(mi)
//			}
//			return ms
//		}
//		return mi.MessageOf(m)
//	}
//
// The MessageState type holds a *MessageInfo, which must be atomically set to
// the message info associated with a given message instance.
// By unsafely converting a *M into a *MessageState, the MessageState object
// has access to all the information needed to implement protobuf reflection.
// It has access to the message info as its first field, and a pointer to the
// MessageState is identical to a pointer to the concrete message value.
//
//
// Requirements:
//	• The type M must implement protoreflect.ProtoMessage.
//	• The address of m must not be nil.
//	• The address of m and the address of m.state must be equal,
//	even though they are different Go types.
type MessageState struct {
	pragma.NoUnkeyedLiterals
	pragma.DoNotCompare
	pragma.DoNotCopy

	mi *MessageInfo
}

type messageState MessageState

var (
	_ pref.Message     = (*messageState)(nil)
	_ pvalue.Unwrapper = (*messageState)(nil)
)

// messageDataType is a tuple of a pointer to the message data and
// a pointer to the message type. It is a generalized way of providing a
// reflective view over a message instance. The disadvantage of this approach
// is the need to allocate this tuple of 16B.
type messageDataType struct {
	p  pointer
	mi *MessageInfo
}

type (
	messageIfaceWrapper   messageDataType
	messageReflectWrapper messageDataType
)

var (
	_ pref.Message      = (*messageReflectWrapper)(nil)
	_ pvalue.Unwrapper  = (*messageReflectWrapper)(nil)
	_ pref.ProtoMessage = (*messageIfaceWrapper)(nil)
	_ pvalue.Unwrapper  = (*messageIfaceWrapper)(nil)
)

// MessageOf returns a reflective view over a message. The input must be a
// pointer to a named Go struct. If the provided type has a ProtoReflect method,
// it must be implemented by calling this method.
func (mi *MessageInfo) MessageOf(m interface{}) pref.Message {
	// TODO: Switch the input to be an opaque Pointer.
	if reflect.TypeOf(m) != mi.GoType {
		panic(fmt.Sprintf("type mismatch: got %T, want %v", m, mi.GoType))
	}
	p := pointerOfIface(m)
	if p.IsNil() {
		return mi.nilMessage.Init(mi)
	}
	return &messageReflectWrapper{p, mi}
}

func (m *messageReflectWrapper) pointer() pointer { return m.p }

func (m *messageIfaceWrapper) ProtoReflect() pref.Message {
	return (*messageReflectWrapper)(m)
}
func (m *messageIfaceWrapper) XXX_Methods() *piface.Methods {
	// TODO: Consider not recreating this on every call.
	m.mi.init()
	return &piface.Methods{
		Flags:         piface.MethodFlagDeterministicMarshal,
		MarshalAppend: m.marshalAppend,
		Unmarshal:     m.unmarshal,
		Size:          m.size,
		IsInitialized: m.isInitialized,
	}
}
func (m *messageIfaceWrapper) ProtoUnwrap() interface{} {
	return m.p.AsIfaceOf(m.mi.GoType.Elem())
}
func (m *messageIfaceWrapper) marshalAppend(b []byte, _ pref.ProtoMessage, opts piface.MarshalOptions) ([]byte, error) {
	return m.mi.marshalAppendPointer(b, m.p, newMarshalOptions(opts))
}
func (m *messageIfaceWrapper) unmarshal(b []byte, _ pref.ProtoMessage, opts piface.UnmarshalOptions) error {
	_, err := m.mi.unmarshalPointer(b, m.p, 0, newUnmarshalOptions(opts))
	return err
}
func (m *messageIfaceWrapper) size(msg pref.ProtoMessage) (size int) {
	return m.mi.sizePointer(m.p, 0)
}
func (m *messageIfaceWrapper) isInitialized(_ pref.ProtoMessage) error {
	return m.mi.isInitializedPointer(m.p)
}

type extensionMap map[int32]ExtensionField

func (m *extensionMap) Range(f func(pref.FieldDescriptor, pref.Value) bool) {
	if m != nil {
		for _, x := range *m {
			xt := x.GetType()
			if !f(xt, xt.ValueOf(x.GetValue())) {
				return
			}
		}
	}
}
func (m *extensionMap) Has(xt pref.ExtensionType) (ok bool) {
	if m != nil {
		_, ok = (*m)[int32(xt.Number())]
	}
	return ok
}
func (m *extensionMap) Clear(xt pref.ExtensionType) {
	delete(*m, int32(xt.Number()))
}
func (m *extensionMap) Get(xt pref.ExtensionType) pref.Value {
	if m != nil {
		if x, ok := (*m)[int32(xt.Number())]; ok {
			return xt.ValueOf(x.GetValue())
		}
	}
	if !isComposite(xt) {
		return defaultValueOf(xt)
	}
	return frozenValueOf(xt.New())
}
func (m *extensionMap) Set(xt pref.ExtensionType, v pref.Value) {
	if *m == nil {
		*m = make(map[int32]ExtensionField)
	}
	var x ExtensionField
	x.SetType(xt)
	x.SetEagerValue(xt.InterfaceOf(v))
	(*m)[int32(xt.Number())] = x
}
func (m *extensionMap) Mutable(xt pref.ExtensionType) pref.Value {
	if !isComposite(xt) {
		panic("invalid Mutable on field with non-composite type")
	}
	if x, ok := (*m)[int32(xt.Number())]; ok {
		return xt.ValueOf(x.GetValue())
	}
	v := xt.New()
	m.Set(xt, v)
	return v
}

func isComposite(fd pref.FieldDescriptor) bool {
	return fd.Kind() == pref.MessageKind || fd.Kind() == pref.GroupKind || fd.IsList() || fd.IsMap()
}

// checkField verifies that the provided field descriptor is valid.
// Exactly one of the returned values is populated.
func (mi *MessageInfo) checkField(fd pref.FieldDescriptor) (*fieldInfo, pref.ExtensionType) {
	if fi := mi.fields[fd.Number()]; fi != nil {
		if fi.fieldDesc != fd {
			panic("mismatching field descriptor")
		}
		return fi, nil
	}
	if fd.IsExtension() {
		if fd.ContainingMessage().FullName() != mi.PBType.FullName() {
			// TODO: Should this be exact containing message descriptor match?
			panic("mismatching containing message")
		}
		if !mi.PBType.ExtensionRanges().Has(fd.Number()) {
			panic("invalid extension field")
		}
		return nil, fd.(pref.ExtensionType)
	}
	panic("invalid field descriptor")
}
