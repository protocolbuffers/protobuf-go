// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build purego appengine

package impl

import (
	"reflect"

	"google.golang.org/protobuf/internal/encoding/wire"
)

func sizeEnum(p pointer, tagsize int, _ marshalOptions) (size int) {
	v := p.v.Elem().Int()
	return tagsize + wire.SizeVarint(uint64(v))
}

func appendEnum(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
	v := p.v.Elem().Int()
	b = wire.AppendVarint(b, wiretag)
	b = wire.AppendVarint(b, uint64(v))
	return b, nil
}

func consumeEnum(b []byte, p pointer, wtyp wire.Type, _ unmarshalOptions) (n int, err error) {
	if wtyp != wire.VarintType {
		return 0, errUnknown
	}
	v, n := wire.ConsumeVarint(b)
	if n < 0 {
		return 0, wire.ParseError(n)
	}
	p.v.Elem().SetInt(int64(v))
	return n, nil
}

var coderEnum = pointerCoderFuncs{
	size:      sizeEnum,
	marshal:   appendEnum,
	unmarshal: consumeEnum,
}

func sizeEnumNoZero(p pointer, tagsize int, opts marshalOptions) (size int) {
	if p.v.Elem().Int() == 0 {
		return 0
	}
	return sizeEnum(p, tagsize, opts)
}

func appendEnumNoZero(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
	if p.v.Elem().Int() == 0 {
		return b, nil
	}
	return appendEnum(b, p, wiretag, opts)
}

var coderEnumNoZero = pointerCoderFuncs{
	size:      sizeEnumNoZero,
	marshal:   appendEnumNoZero,
	unmarshal: consumeEnum,
}

func sizeEnumPtr(p pointer, tagsize int, opts marshalOptions) (size int) {
	return sizeEnum(pointer{p.v.Elem()}, tagsize, opts)
}

func appendEnumPtr(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
	return appendEnum(b, pointer{p.v.Elem()}, wiretag, opts)
}

func consumeEnumPtr(b []byte, p pointer, wtyp wire.Type, opts unmarshalOptions) (n int, err error) {
	if wtyp != wire.VarintType {
		return 0, errUnknown
	}
	if p.v.Elem().IsNil() {
		p.v.Elem().Set(reflect.New(p.v.Elem().Type().Elem()))
	}
	return consumeEnum(b, pointer{p.v.Elem()}, wtyp, opts)
}

var coderEnumPtr = pointerCoderFuncs{
	size:      sizeEnumPtr,
	marshal:   appendEnumPtr,
	unmarshal: consumeEnumPtr,
}

func sizeEnumSlice(p pointer, tagsize int, opts marshalOptions) (size int) {
	return sizeEnumSliceReflect(p.v.Elem(), tagsize, opts)
}

func appendEnumSlice(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
	return appendEnumSliceReflect(b, p.v.Elem(), wiretag, opts)
}

func consumeEnumSlice(b []byte, p pointer, wtyp wire.Type, opts unmarshalOptions) (n int, err error) {
	return consumeEnumSliceReflect(b, p.v, wtyp, opts)
}

var coderEnumSlice = pointerCoderFuncs{
	size:      sizeEnumSlice,
	marshal:   appendEnumSlice,
	unmarshal: consumeEnumSlice,
}

func sizeEnumPackedSlice(p pointer, tagsize int, _ marshalOptions) (size int) {
	s := p.v.Elem()
	slen := s.Len()
	if slen == 0 {
		return 0
	}
	n := 0
	for i := 0; i < slen; i++ {
		n += wire.SizeVarint(uint64(s.Index(i).Int()))
	}
	return tagsize + wire.SizeBytes(n)
}

func appendEnumPackedSlice(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
	s := p.v.Elem()
	slen := s.Len()
	if slen == 0 {
		return b, nil
	}
	b = wire.AppendVarint(b, wiretag)
	n := 0
	for i := 0; i < slen; i++ {
		n += wire.SizeVarint(uint64(s.Index(i).Int()))
	}
	b = wire.AppendVarint(b, uint64(n))
	for i := 0; i < slen; i++ {
		b = wire.AppendVarint(b, uint64(s.Index(i).Int()))
	}
	return b, nil
}

var coderEnumPackedSlice = pointerCoderFuncs{
	size:      sizeEnumPackedSlice,
	marshal:   appendEnumPackedSlice,
	unmarshal: consumeEnumSlice,
}
