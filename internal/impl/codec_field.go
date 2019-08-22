// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"reflect"

	"google.golang.org/protobuf/internal/encoding/wire"
	"google.golang.org/protobuf/proto"
	pref "google.golang.org/protobuf/reflect/protoreflect"
)

type errInvalidUTF8 struct{}

func (errInvalidUTF8) Error() string     { return "string field contains invalid UTF-8" }
func (errInvalidUTF8) InvalidUTF8() bool { return true }

func makeOneofFieldCoder(si structInfo, fd pref.FieldDescriptor) pointerCoderFuncs {
	ot := si.oneofWrappersByNumber[fd.Number()]
	funcs := fieldCoder(fd, ot.Field(0).Type)
	fs := si.oneofsByName[fd.ContainingOneof().Name()]
	ft := fs.Type
	wiretag := wire.EncodeTag(fd.Number(), wireTypes[fd.Kind()])
	tagsize := wire.SizeVarint(wiretag)
	getInfo := func(p pointer) (pointer, bool) {
		v := p.AsValueOf(ft).Elem()
		if v.IsNil() {
			return pointer{}, false
		}
		v = v.Elem() // interface -> *struct
		if v.IsNil() || v.Elem().Type() != ot {
			return pointer{}, false
		}
		return pointerOfValue(v).Apply(zeroOffset), true
	}
	pcf := pointerCoderFuncs{
		size: func(p pointer, _ int, opts marshalOptions) int {
			v, ok := getInfo(p)
			if !ok {
				return 0
			}
			return funcs.size(v, tagsize, opts)
		},
		marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
			v, ok := getInfo(p)
			if !ok {
				return b, nil
			}
			return funcs.marshal(b, v, wiretag, opts)
		},
		unmarshal: func(b []byte, p pointer, wtyp wire.Type, opts unmarshalOptions) (int, error) {
			var vw reflect.Value         // pointer to wrapper type
			vi := p.AsValueOf(ft).Elem() // oneof field value of interface kind
			if !vi.IsNil() && !vi.Elem().IsNil() && vi.Elem().Elem().Type() == ot {
				vw = vi.Elem()
			} else {
				vw = reflect.New(ot)
			}
			n, err := funcs.unmarshal(b, pointerOfValue(vw).Apply(zeroOffset), wtyp, opts)
			if err != nil {
				return 0, err
			}
			vi.Set(vw)
			return n, nil
		},
	}
	if funcs.isInit != nil {
		pcf.isInit = func(p pointer) error {
			v, ok := getInfo(p)
			if !ok {
				return nil
			}
			return funcs.isInit(v)
		}
	}
	return pcf
}

func makeMessageFieldCoder(fd pref.FieldDescriptor, ft reflect.Type) pointerCoderFuncs {
	if mi := getMessageInfo(ft); mi != nil {
		return pointerCoderFuncs{
			size: func(p pointer, tagsize int, opts marshalOptions) int {
				return sizeMessageInfo(p, mi, tagsize, opts)
			},
			marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
				return appendMessageInfo(b, p, wiretag, mi, opts)
			},
			unmarshal: func(b []byte, p pointer, wtyp wire.Type, opts unmarshalOptions) (int, error) {
				return consumeMessageInfo(b, p, mi, wtyp, opts)
			},
			isInit: func(p pointer) error {
				return mi.isInitializedPointer(p.Elem())
			},
		}
	} else {
		return pointerCoderFuncs{
			size: func(p pointer, tagsize int, opts marshalOptions) int {
				m := asMessage(p.AsValueOf(ft).Elem())
				return sizeMessage(m, tagsize, opts)
			},
			marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
				m := asMessage(p.AsValueOf(ft).Elem())
				return appendMessage(b, m, wiretag, opts)
			},
			unmarshal: func(b []byte, p pointer, wtyp wire.Type, opts unmarshalOptions) (int, error) {
				mp := p.AsValueOf(ft).Elem()
				if mp.IsNil() {
					mp.Set(reflect.New(ft.Elem()))
				}
				return consumeMessage(b, asMessage(mp), wtyp, opts)
			},
			isInit: func(p pointer) error {
				m := asMessage(p.AsValueOf(ft).Elem())
				return proto.IsInitialized(m)
			},
		}
	}
}

func sizeMessageInfo(p pointer, mi *MessageInfo, tagsize int, opts marshalOptions) int {
	return wire.SizeBytes(mi.sizePointer(p.Elem(), opts)) + tagsize
}

func appendMessageInfo(b []byte, p pointer, wiretag uint64, mi *MessageInfo, opts marshalOptions) ([]byte, error) {
	b = wire.AppendVarint(b, wiretag)
	b = wire.AppendVarint(b, uint64(mi.sizePointer(p.Elem(), opts)))
	return mi.marshalAppendPointer(b, p.Elem(), opts)
}

func consumeMessageInfo(b []byte, p pointer, mi *MessageInfo, wtyp wire.Type, opts unmarshalOptions) (int, error) {
	if wtyp != wire.BytesType {
		return 0, errUnknown
	}
	v, n := wire.ConsumeBytes(b)
	if n < 0 {
		return 0, wire.ParseError(n)
	}
	if p.Elem().IsNil() {
		p.SetPointer(pointerOfValue(reflect.New(mi.GoReflectType.Elem())))
	}
	if _, err := mi.unmarshalPointer(v, p.Elem(), 0, opts); err != nil {
		return 0, err
	}
	return n, nil
}

func sizeMessage(m proto.Message, tagsize int, _ marshalOptions) int {
	return wire.SizeBytes(proto.Size(m)) + tagsize
}

func appendMessage(b []byte, m proto.Message, wiretag uint64, opts marshalOptions) ([]byte, error) {
	b = wire.AppendVarint(b, wiretag)
	b = wire.AppendVarint(b, uint64(proto.Size(m)))
	return opts.Options().MarshalAppend(b, m)
}

func consumeMessage(b []byte, m proto.Message, wtyp wire.Type, opts unmarshalOptions) (int, error) {
	if wtyp != wire.BytesType {
		return 0, errUnknown
	}
	v, n := wire.ConsumeBytes(b)
	if n < 0 {
		return 0, wire.ParseError(n)
	}
	if err := opts.Options().Unmarshal(v, m); err != nil {
		return 0, err
	}
	return n, nil
}

func sizeMessageValue(v pref.Value, tagsize int, opts marshalOptions) int {
	m := v.Message().Interface()
	return sizeMessage(m, tagsize, opts)
}

func appendMessageValue(b []byte, v pref.Value, wiretag uint64, opts marshalOptions) ([]byte, error) {
	m := v.Message().Interface()
	return appendMessage(b, m, wiretag, opts)
}

func consumeMessageValue(b []byte, v pref.Value, _ wire.Number, wtyp wire.Type, opts unmarshalOptions) (pref.Value, int, error) {
	m := v.Message().Interface()
	n, err := consumeMessage(b, m, wtyp, opts)
	return v, n, err
}

func isInitMessageValue(v pref.Value) error {
	m := v.Message().Interface()
	return proto.IsInitialized(m)
}

var coderMessageValue = valueCoderFuncs{
	size:      sizeMessageValue,
	marshal:   appendMessageValue,
	unmarshal: consumeMessageValue,
	isInit:    isInitMessageValue,
}

func sizeGroupValue(v pref.Value, tagsize int, opts marshalOptions) int {
	m := v.Message().Interface()
	return sizeGroup(m, tagsize, opts)
}

func appendGroupValue(b []byte, v pref.Value, wiretag uint64, opts marshalOptions) ([]byte, error) {
	m := v.Message().Interface()
	return appendGroup(b, m, wiretag, opts)
}

func consumeGroupValue(b []byte, v pref.Value, num wire.Number, wtyp wire.Type, opts unmarshalOptions) (pref.Value, int, error) {
	m := v.Message().Interface()
	n, err := consumeGroup(b, m, num, wtyp, opts)
	return v, n, err
}

var coderGroupValue = valueCoderFuncs{
	size:      sizeGroupValue,
	marshal:   appendGroupValue,
	unmarshal: consumeGroupValue,
	isInit:    isInitMessageValue,
}

func makeGroupFieldCoder(fd pref.FieldDescriptor, ft reflect.Type) pointerCoderFuncs {
	num := fd.Number()
	if mi := getMessageInfo(ft); mi != nil {
		return pointerCoderFuncs{
			size: func(p pointer, tagsize int, opts marshalOptions) int {
				return sizeGroupType(p, mi, tagsize, opts)
			},
			marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
				return appendGroupType(b, p, wiretag, mi, opts)
			},
			unmarshal: func(b []byte, p pointer, wtyp wire.Type, opts unmarshalOptions) (int, error) {
				return consumeGroupType(b, p, mi, num, wtyp, opts)
			},
			isInit: func(p pointer) error {
				return mi.isInitializedPointer(p.Elem())
			},
		}
	} else {
		return pointerCoderFuncs{
			size: func(p pointer, tagsize int, opts marshalOptions) int {
				m := asMessage(p.AsValueOf(ft).Elem())
				return sizeGroup(m, tagsize, opts)
			},
			marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
				m := asMessage(p.AsValueOf(ft).Elem())
				return appendGroup(b, m, wiretag, opts)
			},
			unmarshal: func(b []byte, p pointer, wtyp wire.Type, opts unmarshalOptions) (int, error) {
				mp := p.AsValueOf(ft).Elem()
				if mp.IsNil() {
					mp.Set(reflect.New(ft.Elem()))
				}
				return consumeGroup(b, asMessage(mp), num, wtyp, opts)
			},
			isInit: func(p pointer) error {
				m := asMessage(p.AsValueOf(ft).Elem())
				return proto.IsInitialized(m)
			},
		}
	}
}

func sizeGroupType(p pointer, mi *MessageInfo, tagsize int, opts marshalOptions) int {
	return 2*tagsize + mi.sizePointer(p.Elem(), opts)
}

func appendGroupType(b []byte, p pointer, wiretag uint64, mi *MessageInfo, opts marshalOptions) ([]byte, error) {
	b = wire.AppendVarint(b, wiretag) // start group
	b, err := mi.marshalAppendPointer(b, p.Elem(), opts)
	b = wire.AppendVarint(b, wiretag+1) // end group
	return b, err
}

func consumeGroupType(b []byte, p pointer, mi *MessageInfo, num wire.Number, wtyp wire.Type, opts unmarshalOptions) (int, error) {
	if wtyp != wire.StartGroupType {
		return 0, errUnknown
	}
	if p.Elem().IsNil() {
		p.SetPointer(pointerOfValue(reflect.New(mi.GoReflectType.Elem())))
	}
	return mi.unmarshalPointer(b, p.Elem(), num, opts)
}

func sizeGroup(m proto.Message, tagsize int, _ marshalOptions) int {
	return 2*tagsize + proto.Size(m)
}

func appendGroup(b []byte, m proto.Message, wiretag uint64, opts marshalOptions) ([]byte, error) {
	b = wire.AppendVarint(b, wiretag) // start group
	b, err := opts.Options().MarshalAppend(b, m)
	b = wire.AppendVarint(b, wiretag+1) // end group
	return b, err
}

func consumeGroup(b []byte, m proto.Message, num wire.Number, wtyp wire.Type, opts unmarshalOptions) (int, error) {
	if wtyp != wire.StartGroupType {
		return 0, errUnknown
	}
	b, n := wire.ConsumeGroup(num, b)
	if n < 0 {
		return 0, wire.ParseError(n)
	}
	return n, opts.Options().Unmarshal(b, m)
}

func makeMessageSliceFieldCoder(fd pref.FieldDescriptor, ft reflect.Type) pointerCoderFuncs {
	if mi := getMessageInfo(ft); mi != nil {
		return pointerCoderFuncs{
			size: func(p pointer, tagsize int, opts marshalOptions) int {
				return sizeMessageSliceInfo(p, mi, tagsize, opts)
			},
			marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
				return appendMessageSliceInfo(b, p, wiretag, mi, opts)
			},
			unmarshal: func(b []byte, p pointer, wtyp wire.Type, opts unmarshalOptions) (int, error) {
				return consumeMessageSliceInfo(b, p, mi, wtyp, opts)
			},
			isInit: func(p pointer) error {
				return isInitMessageSliceInfo(p, mi)
			},
		}
	}
	return pointerCoderFuncs{
		size: func(p pointer, tagsize int, opts marshalOptions) int {
			return sizeMessageSlice(p, ft, tagsize, opts)
		},
		marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
			return appendMessageSlice(b, p, wiretag, ft, opts)
		},
		unmarshal: func(b []byte, p pointer, wtyp wire.Type, opts unmarshalOptions) (int, error) {
			return consumeMessageSlice(b, p, ft, wtyp, opts)
		},
		isInit: func(p pointer) error {
			return isInitMessageSlice(p, ft)
		},
	}
}

func sizeMessageSliceInfo(p pointer, mi *MessageInfo, tagsize int, opts marshalOptions) int {
	s := p.PointerSlice()
	n := 0
	for _, v := range s {
		n += wire.SizeBytes(mi.sizePointer(v, opts)) + tagsize
	}
	return n
}

func appendMessageSliceInfo(b []byte, p pointer, wiretag uint64, mi *MessageInfo, opts marshalOptions) ([]byte, error) {
	s := p.PointerSlice()
	var err error
	for _, v := range s {
		b = wire.AppendVarint(b, wiretag)
		siz := mi.sizePointer(v, opts)
		b = wire.AppendVarint(b, uint64(siz))
		b, err = mi.marshalAppendPointer(b, v, opts)
		if err != nil {
			return b, err
		}
	}
	return b, nil
}

func consumeMessageSliceInfo(b []byte, p pointer, mi *MessageInfo, wtyp wire.Type, opts unmarshalOptions) (int, error) {
	if wtyp != wire.BytesType {
		return 0, errUnknown
	}
	v, n := wire.ConsumeBytes(b)
	if n < 0 {
		return 0, wire.ParseError(n)
	}
	m := reflect.New(mi.GoReflectType.Elem()).Interface()
	mp := pointerOfIface(m)
	if _, err := mi.unmarshalPointer(v, mp, 0, opts); err != nil {
		return 0, err
	}
	p.AppendPointerSlice(mp)
	return n, nil
}

func isInitMessageSliceInfo(p pointer, mi *MessageInfo) error {
	s := p.PointerSlice()
	for _, v := range s {
		if err := mi.isInitializedPointer(v); err != nil {
			return err
		}
	}
	return nil
}

func sizeMessageSlice(p pointer, goType reflect.Type, tagsize int, _ marshalOptions) int {
	s := p.PointerSlice()
	n := 0
	for _, v := range s {
		m := asMessage(v.AsValueOf(goType.Elem()))
		n += wire.SizeBytes(proto.Size(m)) + tagsize
	}
	return n
}

func appendMessageSlice(b []byte, p pointer, wiretag uint64, goType reflect.Type, opts marshalOptions) ([]byte, error) {
	s := p.PointerSlice()
	var err error
	for _, v := range s {
		m := asMessage(v.AsValueOf(goType.Elem()))
		b = wire.AppendVarint(b, wiretag)
		siz := proto.Size(m)
		b = wire.AppendVarint(b, uint64(siz))
		b, err = opts.Options().MarshalAppend(b, m)
		if err != nil {
			return b, err
		}
	}
	return b, nil
}

func consumeMessageSlice(b []byte, p pointer, goType reflect.Type, wtyp wire.Type, opts unmarshalOptions) (int, error) {
	if wtyp != wire.BytesType {
		return 0, errUnknown
	}
	v, n := wire.ConsumeBytes(b)
	if n < 0 {
		return 0, wire.ParseError(n)
	}
	mp := reflect.New(goType.Elem())
	if err := opts.Options().Unmarshal(v, asMessage(mp)); err != nil {
		return 0, err
	}
	p.AppendPointerSlice(pointerOfValue(mp))
	return n, nil
}

func isInitMessageSlice(p pointer, goType reflect.Type) error {
	s := p.PointerSlice()
	for _, v := range s {
		m := asMessage(v.AsValueOf(goType.Elem()))
		if err := proto.IsInitialized(m); err != nil {
			return err
		}
	}
	return nil
}

// Slices of messages

func sizeMessageSliceValue(listv pref.Value, tagsize int, opts marshalOptions) int {
	list := listv.List()
	n := 0
	for i, llen := 0, list.Len(); i < llen; i++ {
		m := list.Get(i).Message().Interface()
		n += wire.SizeBytes(proto.Size(m)) + tagsize
	}
	return n
}

func appendMessageSliceValue(b []byte, listv pref.Value, wiretag uint64, opts marshalOptions) ([]byte, error) {
	list := listv.List()
	mopts := opts.Options()
	for i, llen := 0, list.Len(); i < llen; i++ {
		m := list.Get(i).Message().Interface()
		b = wire.AppendVarint(b, wiretag)
		siz := proto.Size(m)
		b = wire.AppendVarint(b, uint64(siz))
		var err error
		b, err = mopts.MarshalAppend(b, m)
		if err != nil {
			return b, err
		}
	}
	return b, nil
}

func consumeMessageSliceValue(b []byte, listv pref.Value, _ wire.Number, wtyp wire.Type, opts unmarshalOptions) (pref.Value, int, error) {
	list := listv.List()
	if wtyp != wire.BytesType {
		return pref.Value{}, 0, errUnknown
	}
	v, n := wire.ConsumeBytes(b)
	if n < 0 {
		return pref.Value{}, 0, wire.ParseError(n)
	}
	m := list.NewElement()
	if err := opts.Options().Unmarshal(v, m.Message().Interface()); err != nil {
		return pref.Value{}, 0, err
	}
	list.Append(m)
	return listv, n, nil
}

func isInitMessageSliceValue(listv pref.Value) error {
	list := listv.List()
	for i, llen := 0, list.Len(); i < llen; i++ {
		m := list.Get(i).Message().Interface()
		if err := proto.IsInitialized(m); err != nil {
			return err
		}
	}
	return nil
}

var coderMessageSliceValue = valueCoderFuncs{
	size:      sizeMessageSliceValue,
	marshal:   appendMessageSliceValue,
	unmarshal: consumeMessageSliceValue,
	isInit:    isInitMessageSliceValue,
}

func sizeGroupSliceValue(listv pref.Value, tagsize int, opts marshalOptions) int {
	list := listv.List()
	n := 0
	for i, llen := 0, list.Len(); i < llen; i++ {
		m := list.Get(i).Message().Interface()
		n += 2*tagsize + proto.Size(m)
	}
	return n
}

func appendGroupSliceValue(b []byte, listv pref.Value, wiretag uint64, opts marshalOptions) ([]byte, error) {
	list := listv.List()
	mopts := opts.Options()
	for i, llen := 0, list.Len(); i < llen; i++ {
		m := list.Get(i).Message().Interface()
		b = wire.AppendVarint(b, wiretag) // start group
		var err error
		b, err = mopts.MarshalAppend(b, m)
		if err != nil {
			return b, err
		}
		b = wire.AppendVarint(b, wiretag+1) // end group
	}
	return b, nil
}

func consumeGroupSliceValue(b []byte, listv pref.Value, num wire.Number, wtyp wire.Type, opts unmarshalOptions) (pref.Value, int, error) {
	list := listv.List()
	if wtyp != wire.StartGroupType {
		return pref.Value{}, 0, errUnknown
	}
	b, n := wire.ConsumeGroup(num, b)
	if n < 0 {
		return pref.Value{}, 0, wire.ParseError(n)
	}
	m := list.NewElement()
	if err := opts.Options().Unmarshal(b, m.Message().Interface()); err != nil {
		return pref.Value{}, 0, err
	}
	list.Append(m)
	return listv, n, nil
}

var coderGroupSliceValue = valueCoderFuncs{
	size:      sizeGroupSliceValue,
	marshal:   appendGroupSliceValue,
	unmarshal: consumeGroupSliceValue,
	isInit:    isInitMessageSliceValue,
}

func makeGroupSliceFieldCoder(fd pref.FieldDescriptor, ft reflect.Type) pointerCoderFuncs {
	num := fd.Number()
	if mi := getMessageInfo(ft); mi != nil {
		return pointerCoderFuncs{
			size: func(p pointer, tagsize int, opts marshalOptions) int {
				return sizeGroupSliceInfo(p, mi, tagsize, opts)
			},
			marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
				return appendGroupSliceInfo(b, p, wiretag, mi, opts)
			},
			unmarshal: func(b []byte, p pointer, wtyp wire.Type, opts unmarshalOptions) (int, error) {
				return consumeGroupSliceInfo(b, p, num, wtyp, mi, opts)
			},
			isInit: func(p pointer) error {
				return isInitMessageSliceInfo(p, mi)
			},
		}
	}
	return pointerCoderFuncs{
		size: func(p pointer, tagsize int, opts marshalOptions) int {
			return sizeGroupSlice(p, ft, tagsize, opts)
		},
		marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
			return appendGroupSlice(b, p, wiretag, ft, opts)
		},
		unmarshal: func(b []byte, p pointer, wtyp wire.Type, opts unmarshalOptions) (int, error) {
			return consumeGroupSlice(b, p, num, wtyp, ft, opts)
		},
		isInit: func(p pointer) error {
			return isInitMessageSlice(p, ft)
		},
	}
}

func sizeGroupSlice(p pointer, messageType reflect.Type, tagsize int, _ marshalOptions) int {
	s := p.PointerSlice()
	n := 0
	for _, v := range s {
		m := asMessage(v.AsValueOf(messageType.Elem()))
		n += 2*tagsize + proto.Size(m)
	}
	return n
}

func appendGroupSlice(b []byte, p pointer, wiretag uint64, messageType reflect.Type, opts marshalOptions) ([]byte, error) {
	s := p.PointerSlice()
	var err error
	for _, v := range s {
		m := asMessage(v.AsValueOf(messageType.Elem()))
		b = wire.AppendVarint(b, wiretag) // start group
		b, err = opts.Options().MarshalAppend(b, m)
		if err != nil {
			return b, err
		}
		b = wire.AppendVarint(b, wiretag+1) // end group
	}
	return b, nil
}

func consumeGroupSlice(b []byte, p pointer, num wire.Number, wtyp wire.Type, goType reflect.Type, opts unmarshalOptions) (int, error) {
	if wtyp != wire.StartGroupType {
		return 0, errUnknown
	}
	b, n := wire.ConsumeGroup(num, b)
	if n < 0 {
		return 0, wire.ParseError(n)
	}
	mp := reflect.New(goType.Elem())
	if err := opts.Options().Unmarshal(b, asMessage(mp)); err != nil {
		return 0, err
	}
	p.AppendPointerSlice(pointerOfValue(mp))
	return n, nil
}

func sizeGroupSliceInfo(p pointer, mi *MessageInfo, tagsize int, opts marshalOptions) int {
	s := p.PointerSlice()
	n := 0
	for _, v := range s {
		n += 2*tagsize + mi.sizePointer(v, opts)
	}
	return n
}

func appendGroupSliceInfo(b []byte, p pointer, wiretag uint64, mi *MessageInfo, opts marshalOptions) ([]byte, error) {
	s := p.PointerSlice()
	var err error
	for _, v := range s {
		b = wire.AppendVarint(b, wiretag) // start group
		b, err = mi.marshalAppendPointer(b, v, opts)
		if err != nil {
			return b, err
		}
		b = wire.AppendVarint(b, wiretag+1) // end group
	}
	return b, nil
}

func consumeGroupSliceInfo(b []byte, p pointer, num wire.Number, wtyp wire.Type, mi *MessageInfo, opts unmarshalOptions) (int, error) {
	if wtyp != wire.StartGroupType {
		return 0, errUnknown
	}
	m := reflect.New(mi.GoReflectType.Elem()).Interface()
	mp := pointerOfIface(m)
	n, err := mi.unmarshalPointer(b, mp, num, opts)
	if err != nil {
		return 0, err
	}
	p.AppendPointerSlice(mp)
	return n, nil
}

func asMessage(v reflect.Value) pref.ProtoMessage {
	if m, ok := v.Interface().(pref.ProtoMessage); ok {
		return m
	}
	return legacyWrapMessage(v)
}
