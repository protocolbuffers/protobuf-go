// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import (
	"google.golang.org/protobuf/reflect/protoreflect"
)

// HasExtension reports whether an extension field is populated.
// It panics if ext does not extend m.
func HasExtension(m Message, ext protoreflect.ExtensionType) bool {
	// Treat nil message interface as an empty message; no populated fields.
	if m == nil {
		return false
	}

	return m.ProtoReflect().Has(ext.TypeDescriptor())
}

// ClearExtension clears an extension field such that subsequent
// HasExtension calls return false.
// It panics if ext does not extend m.
func ClearExtension(m Message, ext protoreflect.ExtensionType) {
	m.ProtoReflect().Clear(ext.TypeDescriptor())
}

// GetExtension retrieves the value for an extension field.
// If the field is unpopulated, it returns the default value for
// scalars and an immutable, empty value for lists or messages.
// It panics if ext does not extend m.
func GetExtension(m Message, ext protoreflect.ExtensionType) interface{} {
	// Treat nil message interface as an empty message; return the default.
	if m == nil {
		return ext.InterfaceOf(ext.Zero())
	}

	return ext.InterfaceOf(m.ProtoReflect().Get(ext.TypeDescriptor()))
}

// SetExtension stores the value of an extension field.
// It panics if ext does not extend m or if value type is invalid for the field.
func SetExtension(m Message, ext protoreflect.ExtensionType, value interface{}) {
	m.ProtoReflect().Set(ext.TypeDescriptor(), ext.ValueOf(value))
}

// RangeExtensions iterates over every populated extension field in m in an
// undefined order, calling f for each extension type and value encountered.
// It returns immediately if f returns false.
// While iterating, mutating operations may only be performed
// on the current extension field.
func RangeExtensions(m Message, f func(protoreflect.ExtensionType, interface{}) bool) {
	// Treat nil message interface as an empty message; nothing to range over.
	if m == nil {
		return
	}

	m.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if fd.IsExtension() {
			xt := fd.(protoreflect.ExtensionTypeDescriptor).Type()
			vi := xt.InterfaceOf(v)
			return f(xt, vi)
		}
		return true
	})
}
