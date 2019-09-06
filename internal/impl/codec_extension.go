// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"sync"
	"sync/atomic"

	"google.golang.org/protobuf/internal/encoding/wire"
	pref "google.golang.org/protobuf/reflect/protoreflect"
)

type extensionFieldInfo struct {
	wiretag             uint64
	tagsize             int
	unmarshalNeedsValue bool
	funcs               valueCoderFuncs
}

func (mi *MessageInfo) extensionFieldInfo(xt pref.ExtensionType) *extensionFieldInfo {
	// As of this time (Go 1.12, linux/amd64), an RWMutex benchmarks as faster
	// than a sync.Map.
	mi.extensionFieldInfosMu.RLock()
	e, ok := mi.extensionFieldInfos[xt]
	mi.extensionFieldInfosMu.RUnlock()
	if ok {
		return e
	}

	xd := xt.TypeDescriptor()
	var wiretag uint64
	if !xd.IsPacked() {
		wiretag = wire.EncodeTag(xd.Number(), wireTypes[xd.Kind()])
	} else {
		wiretag = wire.EncodeTag(xd.Number(), wire.BytesType)
	}
	e = &extensionFieldInfo{
		wiretag: wiretag,
		tagsize: wire.SizeVarint(wiretag),
		funcs:   encoderFuncsForValue(xd),
	}
	// Does the unmarshal function need a value passed to it?
	// This is true for composite types, where we pass in a message, list, or map to fill in,
	// and for enums, where we pass in a prototype value to specify the concrete enum type.
	switch xd.Kind() {
	case pref.MessageKind, pref.GroupKind, pref.EnumKind:
		e.unmarshalNeedsValue = true
	default:
		if xd.Cardinality() == pref.Repeated {
			e.unmarshalNeedsValue = true
		}
	}
	mi.extensionFieldInfosMu.Lock()
	if mi.extensionFieldInfos == nil {
		mi.extensionFieldInfos = make(map[pref.ExtensionType]*extensionFieldInfo)
	}
	mi.extensionFieldInfos[xt] = e
	mi.extensionFieldInfosMu.Unlock()
	return e
}

type ExtensionField struct {
	typ pref.ExtensionType

	// value is either the value of GetValue,
	// or a *lazyExtensionValue that then returns the value of GetValue.
	value pref.Value
	lazy  *lazyExtensionValue
}

// Set sets the type and value of the extension field.
// This must not be called concurrently.
func (f *ExtensionField) Set(t pref.ExtensionType, v pref.Value) {
	f.typ = t
	f.value = v
}

// SetLazy sets the type and a value that is to be lazily evaluated upon first use.
// This must not be called concurrently.
func (f *ExtensionField) SetLazy(t pref.ExtensionType, fn func() pref.Value) {
	f.typ = t
	f.lazy = &lazyExtensionValue{value: fn}
}

// Value returns the value of the extension field.
// This may be called concurrently.
func (f *ExtensionField) Value() pref.Value {
	if f.lazy != nil {
		return f.lazy.GetValue()
	}
	return f.value
}

// Type returns the type of the extension field.
// This may be called concurrently.
func (f ExtensionField) Type() pref.ExtensionType {
	return f.typ
}

// IsSet returns whether the extension field is set.
// This may be called concurrently.
func (f ExtensionField) IsSet() bool {
	return f.typ != nil
}

// Deprecated: Do not use.
func (f ExtensionField) HasType() bool {
	return f.typ != nil
}

// Deprecated: Do not use.
func (f ExtensionField) GetType() pref.ExtensionType {
	return f.typ
}

// Deprecated: Do not use.
func (f *ExtensionField) SetType(t pref.ExtensionType) {
	f.typ = t
}

// Deprecated: Do not use.
func (f ExtensionField) HasValue() bool {
	return f.value.IsValid() || f.lazy != nil
}

// Deprecated: Do not use.
func (f ExtensionField) GetValue() interface{} {
	return f.typ.InterfaceOf(f.Value())
}

// Deprecated: Do not use.
func (f *ExtensionField) SetEagerValue(ival interface{}) {
	f.value = f.typ.ValueOf(ival)
}

// Deprecated: Do not use.
func (f *ExtensionField) SetLazyValue(fn func() interface{}) {
	f.lazy = &lazyExtensionValue{value: func() pref.Value {
		return f.typ.ValueOf(fn())
	}}
}

type lazyExtensionValue struct {
	once  uint32      // atomically set if value is valid
	mu    sync.Mutex  // protects value
	value interface{} // either a pref.Value itself or a func() pref.ValueOf
}

func (v *lazyExtensionValue) GetValue() pref.Value {
	if atomic.LoadUint32(&v.once) == 0 {
		v.mu.Lock()
		if f, ok := v.value.(func() pref.Value); ok {
			v.value = f()
		}
		atomic.StoreUint32(&v.once, 1)
		v.mu.Unlock()
	}
	return v.value.(pref.Value)
}
