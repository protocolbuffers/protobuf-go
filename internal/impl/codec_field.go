// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"reflect"
	"sync"

	"google.golang.org/protobuf/internal/encoding/wire"
	"google.golang.org/protobuf/proto"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	preg "google.golang.org/protobuf/reflect/protoregistry"
)

type errInvalidUTF8 struct{}

func (errInvalidUTF8) Error() string     { return "string field contains invalid UTF-8" }
func (errInvalidUTF8) InvalidUTF8() bool { return true }

// initOneofFieldCoders initializes the fast-path functions for the fields in a oneof.
//
// For size, marshal, and isInit operations, functions are set only on the first field
// in the oneof. The functions are called when the oneof is non-nil, and will dispatch
// to the appropriate field-specific function as necessary.
//
// The unmarshal function is set on each field individually as usual.
func (mi *MessageInfo) initOneofFieldCoders(od pref.OneofDescriptor, si structInfo) {
	type oneofFieldInfo struct {
		wiretag uint64 // field tag (number + wire type)
		tagsize int    // size of the varint-encoded tag
		funcs   pointerCoderFuncs
	}
	fs := si.oneofsByName[od.Name()]
	ft := fs.Type
	oneofFields := make(map[reflect.Type]*oneofFieldInfo)
	needIsInit := false
	fields := od.Fields()
	for i, lim := 0, fields.Len(); i < lim; i++ {
		fd := od.Fields().Get(i)
		num := fd.Number()
		cf := mi.coderFields[num]
		ot := si.oneofWrappersByNumber[num]
		funcs := fieldCoder(fd, ot.Field(0).Type)
		oneofFields[ot] = &oneofFieldInfo{
			wiretag: cf.wiretag,
			tagsize: cf.tagsize,
			funcs:   funcs,
		}
		if funcs.isInit != nil {
			needIsInit = true
		}
		cf.funcs.unmarshal = func(b []byte, p pointer, wtyp wire.Type, opts unmarshalOptions) (unmarshalOutput, error) {
			var vw reflect.Value         // pointer to wrapper type
			vi := p.AsValueOf(ft).Elem() // oneof field value of interface kind
			if !vi.IsNil() && !vi.Elem().IsNil() && vi.Elem().Elem().Type() == ot {
				vw = vi.Elem()
			} else {
				vw = reflect.New(ot)
			}
			out, err := funcs.unmarshal(b, pointerOfValue(vw).Apply(zeroOffset), wtyp, opts)
			if err != nil {
				return out, err
			}
			vi.Set(vw)
			return out, nil
		}
	}
	getInfo := func(p pointer) (pointer, *oneofFieldInfo) {
		v := p.AsValueOf(ft).Elem()
		if v.IsNil() {
			return pointer{}, nil
		}
		v = v.Elem() // interface -> *struct
		if v.IsNil() {
			return pointer{}, nil
		}
		return pointerOfValue(v).Apply(zeroOffset), oneofFields[v.Elem().Type()]
	}
	first := mi.coderFields[od.Fields().Get(0).Number()]
	first.funcs.size = func(p pointer, tagsize int, opts marshalOptions) int {
		p, info := getInfo(p)
		if info == nil || info.funcs.size == nil {
			return 0
		}
		return info.funcs.size(p, info.tagsize, opts)
	}
	first.funcs.marshal = func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
		p, info := getInfo(p)
		if info == nil || info.funcs.marshal == nil {
			return b, nil
		}
		return info.funcs.marshal(b, p, info.wiretag, opts)
	}
	if needIsInit {
		first.funcs.isInit = func(p pointer) error {
			p, info := getInfo(p)
			if info == nil || info.funcs.isInit == nil {
				return nil
			}
			return info.funcs.isInit(p)
		}
	}
}

func makeWeakMessageFieldCoder(fd pref.FieldDescriptor) pointerCoderFuncs {
	var once sync.Once
	var messageType pref.MessageType
	lazyInit := func() {
		once.Do(func() {
			messageName := fd.Message().FullName()
			messageType, _ = preg.GlobalTypes.FindMessageByName(messageName)
		})
	}

	num := fd.Number()
	return pointerCoderFuncs{
		size: func(p pointer, tagsize int, opts marshalOptions) int {
			m, ok := p.WeakFields().get(num)
			if !ok {
				return 0
			}
			lazyInit()
			if messageType == nil {
				panic(fmt.Sprintf("weak message %v is not linked in", fd.Message().FullName()))
			}
			return sizeMessage(m, tagsize, opts)
		},
		marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
			m, ok := p.WeakFields().get(num)
			if !ok {
				return b, nil
			}
			lazyInit()
			if messageType == nil {
				panic(fmt.Sprintf("weak message %v is not linked in", fd.Message().FullName()))
			}
			return appendMessage(b, m, wiretag, opts)
		},
		unmarshal: func(b []byte, p pointer, wtyp wire.Type, opts unmarshalOptions) (unmarshalOutput, error) {
			fs := p.WeakFields()
			m, ok := fs.get(num)
			if !ok {
				lazyInit()
				if messageType == nil {
					return unmarshalOutput{}, errUnknown
				}
				m = messageType.New().Interface()
				fs.set(num, m)
			}
			return consumeMessage(b, m, wtyp, opts)
		},
		isInit: func(p pointer) error {
			m, ok := p.WeakFields().get(num)
			if !ok {
				return nil
			}
			return proto.IsInitialized(m)
		},
	}
}

func makeMessageFieldCoder(fd pref.FieldDescriptor, ft reflect.Type) pointerCoderFuncs {
	if mi := getMessageInfo(ft); mi != nil {
		funcs := pointerCoderFuncs{
			size: func(p pointer, tagsize int, opts marshalOptions) int {
				return sizeMessageInfo(p, mi, tagsize, opts)
			},
			marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
				return appendMessageInfo(b, p, wiretag, mi, opts)
			},
			unmarshal: func(b []byte, p pointer, wtyp wire.Type, opts unmarshalOptions) (unmarshalOutput, error) {
				return consumeMessageInfo(b, p, mi, wtyp, opts)
			},
		}
		if needsInitCheck(mi.Desc) {
			funcs.isInit = func(p pointer) error {
				return mi.isInitializedPointer(p.Elem())
			}
		}
		return funcs
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
			unmarshal: func(b []byte, p pointer, wtyp wire.Type, opts unmarshalOptions) (unmarshalOutput, error) {
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

func consumeMessageInfo(b []byte, p pointer, mi *MessageInfo, wtyp wire.Type, opts unmarshalOptions) (out unmarshalOutput, err error) {
	if wtyp != wire.BytesType {
		return out, errUnknown
	}
	v, n := wire.ConsumeBytes(b)
	if n < 0 {
		return out, wire.ParseError(n)
	}
	if p.Elem().IsNil() {
		p.SetPointer(pointerOfValue(reflect.New(mi.GoReflectType.Elem())))
	}
	if _, err := mi.unmarshalPointer(v, p.Elem(), 0, opts); err != nil {
		return out, err
	}
	out.n = n
	return out, nil
}

func sizeMessage(m proto.Message, tagsize int, _ marshalOptions) int {
	return wire.SizeBytes(proto.Size(m)) + tagsize
}

func appendMessage(b []byte, m proto.Message, wiretag uint64, opts marshalOptions) ([]byte, error) {
	b = wire.AppendVarint(b, wiretag)
	b = wire.AppendVarint(b, uint64(proto.Size(m)))
	return opts.Options().MarshalAppend(b, m)
}

func consumeMessage(b []byte, m proto.Message, wtyp wire.Type, opts unmarshalOptions) (out unmarshalOutput, err error) {
	if wtyp != wire.BytesType {
		return out, errUnknown
	}
	v, n := wire.ConsumeBytes(b)
	if n < 0 {
		return out, wire.ParseError(n)
	}
	if err := opts.Options().Unmarshal(v, m); err != nil {
		return out, err
	}
	out.n = n
	return out, nil
}

func sizeMessageValue(v pref.Value, tagsize int, opts marshalOptions) int {
	m := v.Message().Interface()
	return sizeMessage(m, tagsize, opts)
}

func appendMessageValue(b []byte, v pref.Value, wiretag uint64, opts marshalOptions) ([]byte, error) {
	m := v.Message().Interface()
	return appendMessage(b, m, wiretag, opts)
}

func consumeMessageValue(b []byte, v pref.Value, _ wire.Number, wtyp wire.Type, opts unmarshalOptions) (pref.Value, unmarshalOutput, error) {
	m := v.Message().Interface()
	out, err := consumeMessage(b, m, wtyp, opts)
	return v, out, err
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

func consumeGroupValue(b []byte, v pref.Value, num wire.Number, wtyp wire.Type, opts unmarshalOptions) (pref.Value, unmarshalOutput, error) {
	m := v.Message().Interface()
	out, err := consumeGroup(b, m, num, wtyp, opts)
	return v, out, err
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
		funcs := pointerCoderFuncs{
			size: func(p pointer, tagsize int, opts marshalOptions) int {
				return sizeGroupType(p, mi, tagsize, opts)
			},
			marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
				return appendGroupType(b, p, wiretag, mi, opts)
			},
			unmarshal: func(b []byte, p pointer, wtyp wire.Type, opts unmarshalOptions) (unmarshalOutput, error) {
				return consumeGroupType(b, p, mi, num, wtyp, opts)
			},
		}
		if needsInitCheck(mi.Desc) {
			funcs.isInit = func(p pointer) error {
				return mi.isInitializedPointer(p.Elem())
			}
		}
		return funcs
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
			unmarshal: func(b []byte, p pointer, wtyp wire.Type, opts unmarshalOptions) (unmarshalOutput, error) {
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

func consumeGroupType(b []byte, p pointer, mi *MessageInfo, num wire.Number, wtyp wire.Type, opts unmarshalOptions) (out unmarshalOutput, err error) {
	if wtyp != wire.StartGroupType {
		return out, errUnknown
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

func consumeGroup(b []byte, m proto.Message, num wire.Number, wtyp wire.Type, opts unmarshalOptions) (out unmarshalOutput, err error) {
	if wtyp != wire.StartGroupType {
		return out, errUnknown
	}
	b, n := wire.ConsumeGroup(num, b)
	if n < 0 {
		return out, wire.ParseError(n)
	}
	out.n = n
	return out, opts.Options().Unmarshal(b, m)
}

func makeMessageSliceFieldCoder(fd pref.FieldDescriptor, ft reflect.Type) pointerCoderFuncs {
	if mi := getMessageInfo(ft); mi != nil {
		funcs := pointerCoderFuncs{
			size: func(p pointer, tagsize int, opts marshalOptions) int {
				return sizeMessageSliceInfo(p, mi, tagsize, opts)
			},
			marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
				return appendMessageSliceInfo(b, p, wiretag, mi, opts)
			},
			unmarshal: func(b []byte, p pointer, wtyp wire.Type, opts unmarshalOptions) (unmarshalOutput, error) {
				return consumeMessageSliceInfo(b, p, mi, wtyp, opts)
			},
		}
		if needsInitCheck(mi.Desc) {
			funcs.isInit = func(p pointer) error {
				return isInitMessageSliceInfo(p, mi)
			}
		}
		return funcs
	}
	return pointerCoderFuncs{
		size: func(p pointer, tagsize int, opts marshalOptions) int {
			return sizeMessageSlice(p, ft, tagsize, opts)
		},
		marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
			return appendMessageSlice(b, p, wiretag, ft, opts)
		},
		unmarshal: func(b []byte, p pointer, wtyp wire.Type, opts unmarshalOptions) (unmarshalOutput, error) {
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

func consumeMessageSliceInfo(b []byte, p pointer, mi *MessageInfo, wtyp wire.Type, opts unmarshalOptions) (out unmarshalOutput, err error) {
	if wtyp != wire.BytesType {
		return out, errUnknown
	}
	v, n := wire.ConsumeBytes(b)
	if n < 0 {
		return out, wire.ParseError(n)
	}
	m := reflect.New(mi.GoReflectType.Elem()).Interface()
	mp := pointerOfIface(m)
	if _, err := mi.unmarshalPointer(v, mp, 0, opts); err != nil {
		return out, err
	}
	p.AppendPointerSlice(mp)
	out.n = n
	return out, nil
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

func consumeMessageSlice(b []byte, p pointer, goType reflect.Type, wtyp wire.Type, opts unmarshalOptions) (out unmarshalOutput, err error) {
	if wtyp != wire.BytesType {
		return out, errUnknown
	}
	v, n := wire.ConsumeBytes(b)
	if n < 0 {
		return out, wire.ParseError(n)
	}
	mp := reflect.New(goType.Elem())
	if err := opts.Options().Unmarshal(v, asMessage(mp)); err != nil {
		return out, err
	}
	p.AppendPointerSlice(pointerOfValue(mp))
	out.n = n
	return out, nil
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

func consumeMessageSliceValue(b []byte, listv pref.Value, _ wire.Number, wtyp wire.Type, opts unmarshalOptions) (_ pref.Value, out unmarshalOutput, err error) {
	list := listv.List()
	if wtyp != wire.BytesType {
		return pref.Value{}, out, errUnknown
	}
	v, n := wire.ConsumeBytes(b)
	if n < 0 {
		return pref.Value{}, out, wire.ParseError(n)
	}
	m := list.NewElement()
	if err := opts.Options().Unmarshal(v, m.Message().Interface()); err != nil {
		return pref.Value{}, out, err
	}
	list.Append(m)
	out.n = n
	return listv, out, nil
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

func consumeGroupSliceValue(b []byte, listv pref.Value, num wire.Number, wtyp wire.Type, opts unmarshalOptions) (_ pref.Value, out unmarshalOutput, err error) {
	list := listv.List()
	if wtyp != wire.StartGroupType {
		return pref.Value{}, out, errUnknown
	}
	b, n := wire.ConsumeGroup(num, b)
	if n < 0 {
		return pref.Value{}, out, wire.ParseError(n)
	}
	m := list.NewElement()
	if err := opts.Options().Unmarshal(b, m.Message().Interface()); err != nil {
		return pref.Value{}, out, err
	}
	list.Append(m)
	out.n = n
	return listv, out, nil
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
		funcs := pointerCoderFuncs{
			size: func(p pointer, tagsize int, opts marshalOptions) int {
				return sizeGroupSliceInfo(p, mi, tagsize, opts)
			},
			marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
				return appendGroupSliceInfo(b, p, wiretag, mi, opts)
			},
			unmarshal: func(b []byte, p pointer, wtyp wire.Type, opts unmarshalOptions) (unmarshalOutput, error) {
				return consumeGroupSliceInfo(b, p, num, wtyp, mi, opts)
			},
		}
		if needsInitCheck(mi.Desc) {
			funcs.isInit = func(p pointer) error {
				return isInitMessageSliceInfo(p, mi)
			}
		}
		return funcs
	}
	return pointerCoderFuncs{
		size: func(p pointer, tagsize int, opts marshalOptions) int {
			return sizeGroupSlice(p, ft, tagsize, opts)
		},
		marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
			return appendGroupSlice(b, p, wiretag, ft, opts)
		},
		unmarshal: func(b []byte, p pointer, wtyp wire.Type, opts unmarshalOptions) (unmarshalOutput, error) {
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

func consumeGroupSlice(b []byte, p pointer, num wire.Number, wtyp wire.Type, goType reflect.Type, opts unmarshalOptions) (out unmarshalOutput, err error) {
	if wtyp != wire.StartGroupType {
		return out, errUnknown
	}
	b, n := wire.ConsumeGroup(num, b)
	if n < 0 {
		return out, wire.ParseError(n)
	}
	mp := reflect.New(goType.Elem())
	if err := opts.Options().Unmarshal(b, asMessage(mp)); err != nil {
		return out, err
	}
	p.AppendPointerSlice(pointerOfValue(mp))
	out.n = n
	return out, nil
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

func consumeGroupSliceInfo(b []byte, p pointer, num wire.Number, wtyp wire.Type, mi *MessageInfo, opts unmarshalOptions) (unmarshalOutput, error) {
	if wtyp != wire.StartGroupType {
		return unmarshalOutput{}, errUnknown
	}
	m := reflect.New(mi.GoReflectType.Elem()).Interface()
	mp := pointerOfIface(m)
	out, err := mi.unmarshalPointer(b, mp, num, opts)
	if err != nil {
		return out, err
	}
	p.AppendPointerSlice(mp)
	return out, nil
}

func asMessage(v reflect.Value) pref.ProtoMessage {
	if m, ok := v.Interface().(pref.ProtoMessage); ok {
		return m
	}
	return legacyWrapMessage(v)
}
