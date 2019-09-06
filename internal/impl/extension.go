// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"reflect"
	"sync"
	"sync/atomic"

	pref "google.golang.org/protobuf/reflect/protoreflect"
	piface "google.golang.org/protobuf/runtime/protoiface"
)

// ExtensionInfo implements ExtensionType.
//
// This type contains a number of exported fields for legacy compatibility.
// The only non-deprecated use of this type is through the methods of the
// ExtensionType interface.
type ExtensionInfo struct {
	// An ExtensionInfo may exist in several stages of initialization.
	//
	// extensionInfoUninitialized: Some or all of the legacy exported
	// fields may be set, but none of the unexported fields have been
	// initialized. This is the starting state for an ExtensionInfo
	// in legacy generated code.
	//
	// extensionInfoDescInit: The desc field is set, but other unexported fields
	// may not be initialized. Legacy exported fields may or may not be set.
	// This is the starting state for an ExtensionInfo in newly generated code.
	//
	// extensionInfoFullInit: The ExtensionInfo is fully initialized.
	// This state is only entered after lazy initialization is complete.
	init uint32
	mu   sync.Mutex

	goType reflect.Type
	desc   extensionTypeDescriptor
	conv   Converter

	// ExtendedType is a typed nil-pointer to the parent message type that
	// is being extended. It is possible for this to be unpopulated in v2
	// since the message may no longer implement the MessageV1 interface.
	//
	// Deprecated: Use the ExtendedType method instead.
	ExtendedType piface.MessageV1

	// ExtensionType is zero value of the extension type.
	//
	// For historical reasons, reflect.TypeOf(ExtensionType) and Type.GoType
	// may not be identical:
	//	* for scalars (except []byte), where ExtensionType uses *T,
	//	while Type.GoType uses T.
	//	* for repeated fields, where ExtensionType uses []T,
	//	while Type.GoType uses *[]T.
	//
	// Deprecated: Use the GoType method instead.
	ExtensionType interface{}

	// Field is the field number of the extension.
	//
	// Deprecated: Use the Descriptor().Number method instead.
	Field int32

	// Name is the fully qualified name of extension.
	//
	// Deprecated: Use the Descriptor().FullName method instead.
	Name string

	// Tag is the protobuf struct tag used in the v1 API.
	//
	// Deprecated: Do not use.
	Tag string

	// Filename is the proto filename in which the extension is defined.
	//
	// Deprecated: Use Descriptor().ParentFile().Path() instead.
	Filename string
}

// Stages of initialization: See the ExtensionInfo.init field.
const (
	extensionInfoUninitialized = 0
	extensionInfoDescInit      = 1
	extensionInfoFullInit      = 2
)

func InitExtensionInfo(xi *ExtensionInfo, xd pref.ExtensionDescriptor, goType reflect.Type) {
	xi.goType = goType
	xi.desc = extensionTypeDescriptor{xd, xi}
	xi.init = extensionInfoDescInit
}

func (xi *ExtensionInfo) New() pref.Value {
	return xi.lazyInit().New()
}
func (xi *ExtensionInfo) Zero() pref.Value {
	return xi.lazyInit().Zero()
}
func (xi *ExtensionInfo) ValueOf(v interface{}) pref.Value {
	return xi.lazyInit().PBValueOf(reflect.ValueOf(v))
}
func (xi *ExtensionInfo) InterfaceOf(v pref.Value) interface{} {
	return xi.lazyInit().GoValueOf(v).Interface()
}
func (xi *ExtensionInfo) IsValidValue(v pref.Value) bool {
	return xi.lazyInit().IsValidPB(v)
}
func (xi *ExtensionInfo) IsValidInterface(v interface{}) bool {
	return xi.lazyInit().IsValidGo(reflect.ValueOf(v))
}
func (xi *ExtensionInfo) GoType() reflect.Type {
	xi.lazyInit()
	return xi.goType
}
func (xi *ExtensionInfo) TypeDescriptor() pref.ExtensionTypeDescriptor {
	if atomic.LoadUint32(&xi.init) < extensionInfoDescInit {
		xi.lazyInitSlow()
	}
	return &xi.desc
}

func (xi *ExtensionInfo) lazyInit() Converter {
	if atomic.LoadUint32(&xi.init) < extensionInfoFullInit {
		xi.lazyInitSlow()
	}
	return xi.conv
}

func (xi *ExtensionInfo) lazyInitSlow() {
	xi.mu.Lock()
	defer xi.mu.Unlock()

	if xi.init == extensionInfoFullInit {
		return
	}
	defer atomic.StoreUint32(&xi.init, extensionInfoFullInit)

	if xi.desc.ExtensionDescriptor == nil {
		xi.initFromLegacy()
	}
	if !xi.desc.ExtensionDescriptor.IsPlaceholder() {
		if xi.ExtensionType == nil {
			xi.initToLegacy()
		}
		xi.conv = NewConverter(xi.goType, xi.desc)
	}
}

type extensionTypeDescriptor struct {
	pref.ExtensionDescriptor
	xi *ExtensionInfo
}

func (xtd *extensionTypeDescriptor) Type() pref.ExtensionType {
	return xtd.xi
}
func (xtd *extensionTypeDescriptor) Descriptor() pref.ExtensionDescriptor {
	return xtd.ExtensionDescriptor
}
