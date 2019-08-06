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
		if v.Elem().Type() != ot {
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

func sizeMessageIface(ival interface{}, tagsize int, opts marshalOptions) int {
	m := Export{}.MessageOf(ival).Interface()
	return sizeMessage(m, tagsize, opts)
}

func appendMessageIface(b []byte, ival interface{}, wiretag uint64, opts marshalOptions) ([]byte, error) {
	m := Export{}.MessageOf(ival).Interface()
	return appendMessage(b, m, wiretag, opts)
}

func consumeMessageIface(b []byte, ival interface{}, _ wire.Number, wtyp wire.Type, opts unmarshalOptions) (interface{}, int, error) {
	m := Export{}.MessageOf(ival).Interface()
	n, err := consumeMessage(b, m, wtyp, opts)
	return ival, n, err
}

func isInitMessageIface(ival interface{}) error {
	m := Export{}.MessageOf(ival).Interface()
	return proto.IsInitialized(m)
}

var coderMessageIface = ifaceCoderFuncs{
	size:      sizeMessageIface,
	marshal:   appendMessageIface,
	unmarshal: consumeMessageIface,
	isInit:    isInitMessageIface,
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

func makeGroupValueCoder(fd pref.FieldDescriptor, ft reflect.Type) ifaceCoderFuncs {
	return ifaceCoderFuncs{
		size: func(ival interface{}, tagsize int, opts marshalOptions) int {
			m := Export{}.MessageOf(ival).Interface()
			return sizeGroup(m, tagsize, opts)
		},
		marshal: func(b []byte, ival interface{}, wiretag uint64, opts marshalOptions) ([]byte, error) {
			m := Export{}.MessageOf(ival).Interface()
			return appendGroup(b, m, wiretag, opts)
		},
		unmarshal: func(b []byte, ival interface{}, num wire.Number, wtyp wire.Type, opts unmarshalOptions) (interface{}, int, error) {
			m := Export{}.MessageOf(ival).Interface()
			n, err := consumeGroup(b, m, num, wtyp, opts)
			return ival, n, err
		},
		isInit: isInitMessageIface,
	}
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

func sizeMessageSliceIface(ival interface{}, tagsize int, opts marshalOptions) int {
	p := pointerOfIface(ival)
	return sizeMessageSlice(p, reflect.TypeOf(ival).Elem().Elem(), tagsize, opts)
}

func appendMessageSliceIface(b []byte, ival interface{}, wiretag uint64, opts marshalOptions) ([]byte, error) {
	p := pointerOfIface(ival)
	return appendMessageSlice(b, p, wiretag, reflect.TypeOf(ival).Elem().Elem(), opts)
}

func consumeMessageSliceIface(b []byte, ival interface{}, _ wire.Number, wtyp wire.Type, opts unmarshalOptions) (interface{}, int, error) {
	p := pointerOfIface(ival)
	n, err := consumeMessageSlice(b, p, reflect.TypeOf(ival).Elem().Elem(), wtyp, opts)
	return ival, n, err
}

func isInitMessageSliceIface(ival interface{}) error {
	p := pointerOfIface(ival)
	return isInitMessageSlice(p, reflect.TypeOf(ival).Elem().Elem())
}

var coderMessageSliceIface = ifaceCoderFuncs{
	size:      sizeMessageSliceIface,
	marshal:   appendMessageSliceIface,
	unmarshal: consumeMessageSliceIface,
	isInit:    isInitMessageSliceIface,
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

func sizeGroupSliceIface(ival interface{}, tagsize int, opts marshalOptions) int {
	p := pointerOfIface(ival)
	return sizeGroupSlice(p, reflect.TypeOf(ival).Elem().Elem(), tagsize, opts)
}

func appendGroupSliceIface(b []byte, ival interface{}, wiretag uint64, opts marshalOptions) ([]byte, error) {
	p := pointerOfIface(ival)
	return appendGroupSlice(b, p, wiretag, reflect.TypeOf(ival).Elem().Elem(), opts)
}

func consumeGroupSliceIface(b []byte, ival interface{}, num wire.Number, wtyp wire.Type, opts unmarshalOptions) (interface{}, int, error) {
	p := pointerOfIface(ival)
	n, err := consumeGroupSlice(b, p, num, wtyp, reflect.TypeOf(ival).Elem().Elem(), opts)
	return ival, n, err
}

var coderGroupSliceIface = ifaceCoderFuncs{
	size:      sizeGroupSliceIface,
	marshal:   appendGroupSliceIface,
	unmarshal: consumeGroupSliceIface,
	isInit:    isInitMessageSliceIface,
}

// Enums

func sizeEnumIface(ival interface{}, tagsize int, _ marshalOptions) (n int) {
	v := reflect.ValueOf(ival).Int()
	return wire.SizeVarint(uint64(v)) + tagsize
}

func appendEnumIface(b []byte, ival interface{}, wiretag uint64, _ marshalOptions) ([]byte, error) {
	v := reflect.ValueOf(ival).Int()
	b = wire.AppendVarint(b, wiretag)
	b = wire.AppendVarint(b, uint64(v))
	return b, nil
}

func consumeEnumIface(b []byte, ival interface{}, _ wire.Number, wtyp wire.Type, _ unmarshalOptions) (interface{}, int, error) {
	if wtyp != wire.VarintType {
		return nil, 0, errUnknown
	}
	v, n := wire.ConsumeVarint(b)
	if n < 0 {
		return nil, 0, wire.ParseError(n)
	}
	rv := reflect.New(reflect.TypeOf(ival)).Elem()
	rv.SetInt(int64(v))
	return rv.Interface(), n, nil
}

var coderEnumIface = ifaceCoderFuncs{
	size:      sizeEnumIface,
	marshal:   appendEnumIface,
	unmarshal: consumeEnumIface,
}

func sizeEnumSliceIface(ival interface{}, tagsize int, opts marshalOptions) (size int) {
	return sizeEnumSliceReflect(reflect.ValueOf(ival).Elem(), tagsize, opts)
}

func sizeEnumSliceReflect(s reflect.Value, tagsize int, _ marshalOptions) (size int) {
	for i, llen := 0, s.Len(); i < llen; i++ {
		size += wire.SizeVarint(uint64(s.Index(i).Int())) + tagsize
	}
	return size
}

func appendEnumSliceIface(b []byte, ival interface{}, wiretag uint64, opts marshalOptions) ([]byte, error) {
	return appendEnumSliceReflect(b, reflect.ValueOf(ival).Elem(), wiretag, opts)
}

func appendEnumSliceReflect(b []byte, s reflect.Value, wiretag uint64, opts marshalOptions) ([]byte, error) {
	for i, llen := 0, s.Len(); i < llen; i++ {
		b = wire.AppendVarint(b, wiretag)
		b = wire.AppendVarint(b, uint64(s.Index(i).Int()))
	}
	return b, nil
}

func consumeEnumSliceIface(b []byte, ival interface{}, _ wire.Number, wtyp wire.Type, opts unmarshalOptions) (interface{}, int, error) {
	n, err := consumeEnumSliceReflect(b, reflect.ValueOf(ival), wtyp, opts)
	return ival, n, err
}

func consumeEnumSliceReflect(b []byte, s reflect.Value, wtyp wire.Type, _ unmarshalOptions) (n int, err error) {
	s = s.Elem() // *[]E -> []E
	if wtyp == wire.BytesType {
		b, n = wire.ConsumeBytes(b)
		if n < 0 {
			return 0, wire.ParseError(n)
		}
		for len(b) > 0 {
			v, n := wire.ConsumeVarint(b)
			if n < 0 {
				return 0, wire.ParseError(n)
			}
			rv := reflect.New(s.Type().Elem()).Elem()
			rv.SetInt(int64(v))
			s.Set(reflect.Append(s, rv))
			b = b[n:]
		}
		return n, nil
	}
	if wtyp != wire.VarintType {
		return 0, errUnknown
	}
	v, n := wire.ConsumeVarint(b)
	if n < 0 {
		return 0, wire.ParseError(n)
	}
	rv := reflect.New(s.Type().Elem()).Elem()
	rv.SetInt(int64(v))
	s.Set(reflect.Append(s, rv))
	return n, nil
}

var coderEnumSliceIface = ifaceCoderFuncs{
	size:      sizeEnumSliceIface,
	marshal:   appendEnumSliceIface,
	unmarshal: consumeEnumSliceIface,
}

func sizeEnumPackedSliceIface(ival interface{}, tagsize int, opts marshalOptions) (size int) {
	return sizeEnumPackedSliceReflect(reflect.ValueOf(ival).Elem(), tagsize, opts)
}

func sizeEnumPackedSliceReflect(s reflect.Value, tagsize int, _ marshalOptions) (size int) {
	llen := s.Len()
	if llen == 0 {
		return 0
	}
	n := 0
	for i := 0; i < llen; i++ {
		n += wire.SizeVarint(uint64(s.Index(i).Int()))
	}
	return tagsize + wire.SizeBytes(n)
}

func appendEnumPackedSliceIface(b []byte, ival interface{}, wiretag uint64, opts marshalOptions) ([]byte, error) {
	return appendEnumPackedSliceReflect(b, reflect.ValueOf(ival).Elem(), wiretag, opts)
}

func appendEnumPackedSliceReflect(b []byte, s reflect.Value, wiretag uint64, opts marshalOptions) ([]byte, error) {
	llen := s.Len()
	if llen == 0 {
		return b, nil
	}
	b = wire.AppendVarint(b, wiretag)
	n := 0
	for i := 0; i < llen; i++ {
		n += wire.SizeVarint(uint64(s.Index(i).Int()))
	}
	b = wire.AppendVarint(b, uint64(n))
	for i := 0; i < llen; i++ {
		b = wire.AppendVarint(b, uint64(s.Index(i).Int()))
	}
	return b, nil
}

var coderEnumPackedSliceIface = ifaceCoderFuncs{
	size:      sizeEnumPackedSliceIface,
	marshal:   appendEnumPackedSliceIface,
	unmarshal: consumeEnumSliceIface,
}

func asMessage(v reflect.Value) pref.ProtoMessage {
	if m, ok := v.Interface().(pref.ProtoMessage); ok {
		return m
	}
	return legacyWrapMessage(v)
}
