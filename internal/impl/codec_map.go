// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"errors"
	"reflect"
	"sort"

	"google.golang.org/protobuf/internal/encoding/wire"
	pref "google.golang.org/protobuf/reflect/protoreflect"
)

type mapInfo struct {
	goType         reflect.Type
	keyWiretag     uint64
	valWiretag     uint64
	keyFuncs       valueCoderFuncs
	valFuncs       valueCoderFuncs
	keyZero        pref.Value
	keyKind        pref.Kind
	valMessageInfo *MessageInfo
	conv           *mapConverter
}

func encoderFuncsForMap(fd pref.FieldDescriptor, ft reflect.Type) (funcs pointerCoderFuncs) {
	// TODO: Consider generating specialized map coders.
	keyField := fd.MapKey()
	valField := fd.MapValue()
	keyWiretag := wire.EncodeTag(1, wireTypes[keyField.Kind()])
	valWiretag := wire.EncodeTag(2, wireTypes[valField.Kind()])
	keyFuncs := encoderFuncsForValue(keyField)
	valFuncs := encoderFuncsForValue(valField)
	conv := newMapConverter(ft, fd)

	mapi := &mapInfo{
		goType:     ft,
		keyWiretag: keyWiretag,
		valWiretag: valWiretag,
		keyFuncs:   keyFuncs,
		valFuncs:   valFuncs,
		keyZero:    keyField.Default(),
		keyKind:    keyField.Kind(),
		conv:       conv,
	}
	if valField.Kind() == pref.MessageKind {
		mapi.valMessageInfo = getMessageInfo(ft.Elem())
	}

	funcs = pointerCoderFuncs{
		size: func(p pointer, tagsize int, opts marshalOptions) int {
			return sizeMap(p.AsValueOf(ft).Elem(), tagsize, mapi, opts)
		},
		marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
			return appendMap(b, p.AsValueOf(ft).Elem(), wiretag, mapi, opts)
		},
		unmarshal: func(b []byte, p pointer, wtyp wire.Type, opts unmarshalOptions) (int, error) {
			mp := p.AsValueOf(ft)
			if mp.Elem().IsNil() {
				mp.Elem().Set(reflect.MakeMap(mapi.goType))
			}
			if mapi.valMessageInfo == nil {
				return consumeMap(b, mp.Elem(), wtyp, mapi, opts)
			} else {
				return consumeMapOfMessage(b, mp.Elem(), wtyp, mapi, opts)
			}
		},
	}
	if valFuncs.isInit != nil {
		funcs.isInit = func(p pointer) error {
			return isInitMap(p.AsValueOf(ft).Elem(), mapi)
		}
	}
	return funcs
}

const (
	mapKeyTagSize = 1 // field 1, tag size 1.
	mapValTagSize = 1 // field 2, tag size 2.
)

func sizeMap(mapv reflect.Value, tagsize int, mapi *mapInfo, opts marshalOptions) int {
	if mapv.Len() == 0 {
		return 0
	}
	n := 0
	iter := mapRange(mapv)
	for iter.Next() {
		key := mapi.conv.keyConv.PBValueOf(iter.Key()).MapKey()
		keySize := mapi.keyFuncs.size(key.Value(), mapKeyTagSize, opts)
		var valSize int
		value := mapi.conv.valConv.PBValueOf(iter.Value())
		if mapi.valMessageInfo == nil {
			valSize = mapi.valFuncs.size(value, mapValTagSize, opts)
		} else {
			p := pointerOfValue(iter.Value())
			valSize += mapValTagSize
			valSize += wire.SizeBytes(mapi.valMessageInfo.sizePointer(p, opts))
		}
		n += tagsize + wire.SizeBytes(keySize+valSize)
	}
	return n
}

func consumeMap(b []byte, mapv reflect.Value, wtyp wire.Type, mapi *mapInfo, opts unmarshalOptions) (int, error) {
	if wtyp != wire.BytesType {
		return 0, errUnknown
	}
	b, n := wire.ConsumeBytes(b)
	if n < 0 {
		return 0, wire.ParseError(n)
	}
	var (
		key = mapi.keyZero
		val = mapi.conv.valConv.New()
	)
	for len(b) > 0 {
		num, wtyp, n := wire.ConsumeTag(b)
		if n < 0 {
			return 0, wire.ParseError(n)
		}
		if num > wire.MaxValidNumber {
			return 0, errors.New("invalid field number")
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
	mapv.SetMapIndex(mapi.conv.keyConv.GoValueOf(key), mapi.conv.valConv.GoValueOf(val))
	return n, nil
}

func consumeMapOfMessage(b []byte, mapv reflect.Value, wtyp wire.Type, mapi *mapInfo, opts unmarshalOptions) (int, error) {
	if wtyp != wire.BytesType {
		return 0, errUnknown
	}
	b, n := wire.ConsumeBytes(b)
	if n < 0 {
		return 0, wire.ParseError(n)
	}
	var (
		key = mapi.keyZero
		val = reflect.New(mapi.valMessageInfo.GoReflectType.Elem())
	)
	for len(b) > 0 {
		num, wtyp, n := wire.ConsumeTag(b)
		if n < 0 {
			return 0, wire.ParseError(n)
		}
		if num > wire.MaxValidNumber {
			return 0, errors.New("invalid field number")
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
			if wtyp != wire.BytesType {
				break
			}
			var v []byte
			v, n = wire.ConsumeBytes(b)
			if n < 0 {
				return 0, wire.ParseError(n)
			}
			_, err = mapi.valMessageInfo.unmarshalPointer(v, pointerOfValue(val), 0, opts)
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
	mapv.SetMapIndex(mapi.conv.keyConv.GoValueOf(key), val)
	return n, nil
}

func appendMapItem(b []byte, keyrv, valrv reflect.Value, mapi *mapInfo, opts marshalOptions) ([]byte, error) {
	if mapi.valMessageInfo == nil {
		key := mapi.conv.keyConv.PBValueOf(keyrv).MapKey()
		val := mapi.conv.valConv.PBValueOf(valrv)
		size := 0
		size += mapi.keyFuncs.size(key.Value(), mapKeyTagSize, opts)
		size += mapi.valFuncs.size(val, mapValTagSize, opts)
		b = wire.AppendVarint(b, uint64(size))
		b, err := mapi.keyFuncs.marshal(b, key.Value(), mapi.keyWiretag, opts)
		if err != nil {
			return nil, err
		}
		return mapi.valFuncs.marshal(b, val, mapi.valWiretag, opts)
	} else {
		key := mapi.conv.keyConv.PBValueOf(keyrv).MapKey()
		val := pointerOfValue(valrv)
		valSize := mapi.valMessageInfo.sizePointer(val, opts)
		size := 0
		size += mapi.keyFuncs.size(key.Value(), mapKeyTagSize, opts)
		size += mapValTagSize + wire.SizeBytes(valSize)
		b = wire.AppendVarint(b, uint64(size))
		b, err := mapi.keyFuncs.marshal(b, key.Value(), mapi.keyWiretag, opts)
		if err != nil {
			return nil, err
		}
		b = wire.AppendVarint(b, mapi.valWiretag)
		b = wire.AppendVarint(b, uint64(valSize))
		return mapi.valMessageInfo.marshalAppendPointer(b, val, opts)
	}
}

func appendMap(b []byte, mapv reflect.Value, wiretag uint64, mapi *mapInfo, opts marshalOptions) ([]byte, error) {
	if mapv.Len() == 0 {
		return b, nil
	}
	if opts.Deterministic() {
		return appendMapDeterministic(b, mapv, wiretag, mapi, opts)
	}
	iter := mapRange(mapv)
	for iter.Next() {
		var err error
		b = wire.AppendVarint(b, wiretag)
		b, err = appendMapItem(b, iter.Key(), iter.Value(), mapi, opts)
		if err != nil {
			return b, err
		}
	}
	return b, nil
}

func appendMapDeterministic(b []byte, mapv reflect.Value, wiretag uint64, mapi *mapInfo, opts marshalOptions) ([]byte, error) {
	keys := mapv.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		switch keys[i].Kind() {
		case reflect.Bool:
			return !keys[i].Bool() && keys[j].Bool()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return keys[i].Int() < keys[j].Int()
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			return keys[i].Uint() < keys[j].Uint()
		case reflect.Float32, reflect.Float64:
			return keys[i].Float() < keys[j].Float()
		case reflect.String:
			return keys[i].String() < keys[j].String()
		default:
			panic("invalid kind: " + keys[i].Kind().String())
		}
	})
	for _, key := range keys {
		var err error
		b = wire.AppendVarint(b, wiretag)
		b, err = appendMapItem(b, key, mapv.MapIndex(key), mapi, opts)
		if err != nil {
			return b, err
		}
	}
	return b, nil
}

func isInitMap(mapv reflect.Value, mapi *mapInfo) error {
	if mi := mapi.valMessageInfo; mi != nil {
		mi.init()
		if !mi.needsInitCheck {
			return nil
		}
		iter := mapRange(mapv)
		for iter.Next() {
			val := pointerOfValue(iter.Value())
			if err := mi.isInitializedPointer(val); err != nil {
				return err
			}
		}
	} else {
		iter := mapRange(mapv)
		for iter.Next() {
			val := mapi.conv.valConv.PBValueOf(iter.Value())
			if err := mapi.valFuncs.isInit(val); err != nil {
				return err
			}
		}
	}
	return nil
}
