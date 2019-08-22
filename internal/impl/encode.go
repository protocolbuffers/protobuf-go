// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"sort"
	"sync/atomic"

	proto "google.golang.org/protobuf/proto"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	piface "google.golang.org/protobuf/runtime/protoiface"
)

// marshalOptions is a more efficient representation of MarshalOptions.
//
// We don't preserve the AllowPartial flag, because fast-path (un)marshal
// operations always allow partial messages.
type marshalOptions uint

const (
	marshalDeterministic marshalOptions = 1 << iota
	marshalUseCachedSize
)

func newMarshalOptions(opts piface.MarshalOptions) marshalOptions {
	var o marshalOptions
	if opts.Deterministic {
		o |= marshalDeterministic
	}
	if opts.UseCachedSize {
		o |= marshalUseCachedSize
	}
	return o
}

func (o marshalOptions) Options() proto.MarshalOptions {
	return proto.MarshalOptions{
		AllowPartial:  true,
		Deterministic: o.Deterministic(),
		UseCachedSize: o.UseCachedSize(),
	}
}

func (o marshalOptions) Deterministic() bool { return o&marshalDeterministic != 0 }
func (o marshalOptions) UseCachedSize() bool { return o&marshalUseCachedSize != 0 }

// size is protoreflect.Methods.Size.
func (mi *MessageInfo) size(m pref.Message, opts piface.MarshalOptions) (size int) {
	var p pointer
	if ms, ok := m.(*messageState); ok {
		p = ms.pointer()
	} else {
		p = m.(*messageReflectWrapper).pointer()
	}
	return mi.sizePointer(p, newMarshalOptions(opts))
}

func (mi *MessageInfo) sizePointer(p pointer, opts marshalOptions) (size int) {
	mi.init()
	if p.IsNil() {
		return 0
	}
	if opts.UseCachedSize() && mi.sizecacheOffset.IsValid() {
		return int(atomic.LoadInt32(p.Apply(mi.sizecacheOffset).Int32()))
	}
	return mi.sizePointerSlow(p, opts)
}

func (mi *MessageInfo) sizePointerSlow(p pointer, opts marshalOptions) (size int) {
	if mi.extensionOffset.IsValid() {
		e := p.Apply(mi.extensionOffset).Extensions()
		size += mi.sizeExtensions(e, opts)
	}
	for _, f := range mi.orderedCoderFields {
		fptr := p.Apply(f.offset)
		if f.isPointer && fptr.Elem().IsNil() {
			continue
		}
		if f.funcs.size == nil {
			continue
		}
		size += f.funcs.size(fptr, f.tagsize, opts)
	}
	if mi.unknownOffset.IsValid() {
		u := *p.Apply(mi.unknownOffset).Bytes()
		size += len(u)
	}
	if mi.sizecacheOffset.IsValid() {
		atomic.StoreInt32(p.Apply(mi.sizecacheOffset).Int32(), int32(size))
	}
	return size
}

// marshalAppend is protoreflect.Methods.MarshalAppend.
func (mi *MessageInfo) marshalAppend(b []byte, m pref.Message, opts piface.MarshalOptions) ([]byte, error) {
	var p pointer
	if ms, ok := m.(*messageState); ok {
		p = ms.pointer()
	} else {
		p = m.(*messageReflectWrapper).pointer()
	}
	return mi.marshalAppendPointer(b, p, newMarshalOptions(opts))
}

func (mi *MessageInfo) marshalAppendPointer(b []byte, p pointer, opts marshalOptions) ([]byte, error) {
	mi.init()
	if p.IsNil() {
		return b, nil
	}
	var err error
	// The old marshaler encodes extensions at beginning.
	if mi.extensionOffset.IsValid() {
		e := p.Apply(mi.extensionOffset).Extensions()
		// TODO: Special handling for MessageSet?
		b, err = mi.appendExtensions(b, e, opts)
		if err != nil {
			return b, err
		}
	}
	for _, f := range mi.orderedCoderFields {
		fptr := p.Apply(f.offset)
		if f.isPointer && fptr.Elem().IsNil() {
			continue
		}
		if f.funcs.marshal == nil {
			continue
		}
		b, err = f.funcs.marshal(b, fptr, f.wiretag, opts)
		if err != nil {
			return b, err
		}
	}
	if mi.unknownOffset.IsValid() {
		u := *p.Apply(mi.unknownOffset).Bytes()
		b = append(b, u...)
	}
	return b, nil
}

func (mi *MessageInfo) sizeExtensions(ext *map[int32]ExtensionField, opts marshalOptions) (n int) {
	if ext == nil {
		return 0
	}
	for _, x := range *ext {
		xi := mi.extensionFieldInfo(x.Type())
		if xi.funcs.size == nil {
			continue
		}
		n += xi.funcs.size(x.Value(), xi.tagsize, opts)
	}
	return n
}

func (mi *MessageInfo) appendExtensions(b []byte, ext *map[int32]ExtensionField, opts marshalOptions) ([]byte, error) {
	if ext == nil {
		return b, nil
	}

	switch len(*ext) {
	case 0:
		return b, nil
	case 1:
		// Fast-path for one extension: Don't bother sorting the keys.
		var err error
		for _, x := range *ext {
			xi := mi.extensionFieldInfo(x.Type())
			b, err = xi.funcs.marshal(b, x.Value(), xi.wiretag, opts)
		}
		return b, err
	default:
		// Sort the keys to provide a deterministic encoding.
		// Not sure this is required, but the old code does it.
		keys := make([]int, 0, len(*ext))
		for k := range *ext {
			keys = append(keys, int(k))
		}
		sort.Ints(keys)
		var err error
		for _, k := range keys {
			x := (*ext)[int32(k)]
			xi := mi.extensionFieldInfo(x.Type())
			b, err = xi.funcs.marshal(b, x.Value(), xi.wiretag, opts)
			if err != nil {
				return b, err
			}
		}
		return b, nil
	}
}
