// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"reflect"

	"google.golang.org/protobuf/internal/encoding/wire"
	"google.golang.org/protobuf/internal/mapsort"
	pref "google.golang.org/protobuf/reflect/protoreflect"
)

type mapInfo struct {
	goType     reflect.Type
	keyWiretag uint64
	valWiretag uint64
	keyFuncs   valueCoderFuncs
	valFuncs   valueCoderFuncs
	keyZero    pref.Value
	keyKind    pref.Kind
}

func encoderFuncsForMap(fd pref.FieldDescriptor, ft reflect.Type) (funcs pointerCoderFuncs) {
	// TODO: Consider generating specialized map coders.
	keyField := fd.MapKey()
	valField := fd.MapValue()
	keyWiretag := wire.EncodeTag(1, wireTypes[keyField.Kind()])
	valWiretag := wire.EncodeTag(2, wireTypes[valField.Kind()])
	keyFuncs := encoderFuncsForValue(keyField)
	valFuncs := encoderFuncsForValue(valField)
	conv := NewConverter(ft, fd)

	mapi := &mapInfo{
		goType:     ft,
		keyWiretag: keyWiretag,
		valWiretag: valWiretag,
		keyFuncs:   keyFuncs,
		valFuncs:   valFuncs,
		keyZero:    keyField.Default(),
		keyKind:    keyField.Kind(),
	}

	funcs = pointerCoderFuncs{
		size: func(p pointer, tagsize int, opts marshalOptions) int {
			mapv := conv.PBValueOf(p.AsValueOf(ft).Elem()).Map()
			return sizeMap(mapv, tagsize, mapi, opts)
		},
		marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
			mapv := conv.PBValueOf(p.AsValueOf(ft).Elem()).Map()
			return appendMap(b, mapv, wiretag, mapi, opts)
		},
		unmarshal: func(b []byte, p pointer, wtyp wire.Type, opts unmarshalOptions) (int, error) {
			mp := p.AsValueOf(ft)
			if mp.Elem().IsNil() {
				mp.Elem().Set(reflect.MakeMap(mapi.goType))
			}
			mapv := conv.PBValueOf(mp.Elem()).Map()
			return consumeMap(b, mapv, wtyp, mapi, opts)
		},
	}
	if valFuncs.isInit != nil {
		funcs.isInit = func(p pointer) error {
			mapv := conv.PBValueOf(p.AsValueOf(ft).Elem()).Map()
			return isInitMap(mapv, mapi)
		}
	}
	return funcs
}

const (
	mapKeyTagSize = 1 // field 1, tag size 1.
	mapValTagSize = 1 // field 2, tag size 2.
)

func sizeMap(mapv pref.Map, tagsize int, mapi *mapInfo, opts marshalOptions) int {
	if mapv.Len() == 0 {
		return 0
	}
	n := 0
	mapv.Range(func(key pref.MapKey, value pref.Value) bool {
		n += tagsize + wire.SizeBytes(
			mapi.keyFuncs.size(key.Value(), mapKeyTagSize, opts)+
				mapi.valFuncs.size(value, mapValTagSize, opts))
		return true
	})
	return n
}

func consumeMap(b []byte, mapv pref.Map, wtyp wire.Type, mapi *mapInfo, opts unmarshalOptions) (int, error) {
	if wtyp != wire.BytesType {
		return 0, errUnknown
	}
	b, n := wire.ConsumeBytes(b)
	if n < 0 {
		return 0, wire.ParseError(n)
	}
	var (
		key = mapi.keyZero
		val = mapv.NewValue()
	)
	for len(b) > 0 {
		num, wtyp, n := wire.ConsumeTag(b)
		if n < 0 {
			return 0, wire.ParseError(n)
		}
		b = b[n:]
		err := errUnknown
		switch num {
		case 1:
			var v pref.Value
			v, n, err = mapi.keyFuncs.unmarshal(b, key, num, wtyp, opts)
			if err != nil {
				break
			}
			key = v
		case 2:
			var v pref.Value
			v, n, err = mapi.valFuncs.unmarshal(b, val, num, wtyp, opts)
			if err != nil {
				break
			}
			val = v
		}
		if err == errUnknown {
			n = wire.ConsumeFieldValue(num, wtyp, b)
			if n < 0 {
				return 0, wire.ParseError(n)
			}
		} else if err != nil {
			return 0, err
		}
		b = b[n:]
	}
	mapv.Set(key.MapKey(), val)
	return n, nil
}

func appendMap(b []byte, mapv pref.Map, wiretag uint64, mapi *mapInfo, opts marshalOptions) ([]byte, error) {
	if mapv.Len() == 0 {
		return b, nil
	}
	var err error
	fn := func(key pref.MapKey, value pref.Value) bool {
		b = wire.AppendVarint(b, wiretag)
		size := 0
		size += mapi.keyFuncs.size(key.Value(), mapKeyTagSize, opts)
		size += mapi.valFuncs.size(value, mapValTagSize, opts)
		b = wire.AppendVarint(b, uint64(size))
		b, err = mapi.keyFuncs.marshal(b, key.Value(), mapi.keyWiretag, opts)
		if err != nil {
			return false
		}
		b, err = mapi.valFuncs.marshal(b, value, mapi.valWiretag, opts)
		if err != nil {
			return false
		}
		return true
	}
	if opts.Deterministic() {
		mapsort.Range(mapv, mapi.keyKind, fn)
	} else {
		mapv.Range(fn)
	}
	return b, err
}

func isInitMap(mapv pref.Map, mapi *mapInfo) error {
	var err error
	mapv.Range(func(_ pref.MapKey, value pref.Value) bool {
		err = mapi.valFuncs.isInit(value)
		return err == nil
	})
	return err
}
