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
	// extensionInfoDescInit: The desc and tdesc fields have been
	// set, but the descriptor is not otherwise initialized. Legacy
	// exported fields may or may not be set. This is the starting state
	// for an ExtensionInfo in new generated code. Calling the Descriptor
	// method will not trigger lazy initialization, although any other
	// method will.
	//
	// extensionInfoFullInit: The ExtensionInfo is fully initialized.
	// This state is only entered after lazy initialization is complete.
	init uint32
	mu   sync.Mutex

	desc   pref.ExtensionDescriptor
	tdesc  extensionTypeDescriptor
	goType reflect.Type
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
	if xi.desc != nil {
		return
	}
	xi.desc = xd
	xi.goType = goType

	xi.tdesc.ExtensionDescriptor = xi.desc
	xi.tdesc.xi = xi
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
	if atomic.LoadUint32(&xi.init) == extensionInfoUninitialized {
		xi.lazyInitSlow()
	}
	return &xi.tdesc
}

func (xi *ExtensionInfo) lazyInit() Converter {
	if atomic.LoadUint32(&xi.init) != extensionInfoFullInit {
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

	if xi.desc == nil {
		xi.initFromLegacy()
	} else if xi.desc.Cardinality() == pref.Repeated {
		// Cardinality is initialized lazily, so we defer consulting it until here.
		xi.goType = reflect.SliceOf(xi.goType)
	}
	xi.conv = NewConverter(xi.goType, xi.desc)
	xi.tdesc.ExtensionDescriptor = xi.desc
	xi.tdesc.xi = xi

	if xi.ExtensionType == nil {
		xi.initToLegacy()
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
