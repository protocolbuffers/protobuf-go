// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"reflect"
	"sort"

	"google.golang.org/protobuf/internal/encoding/wire"
	"google.golang.org/protobuf/proto"
	pref "google.golang.org/protobuf/reflect/protoreflect"
)

var protoMessageType = reflect.TypeOf((*proto.Message)(nil)).Elem()

type mapInfo struct {
	goType     reflect.Type
	keyWiretag uint64
	valWiretag uint64
	keyFuncs   ifaceCoderFuncs
	valFuncs   ifaceCoderFuncs
	keyZero    interface{}
	valZero    interface{}
	newVal     func() interface{}
}

func encoderFuncsForMap(fd pref.FieldDescriptor, ft reflect.Type) (funcs pointerCoderFuncs) {
	// TODO: Consider generating specialized map coders.
	keyField := fd.MapKey()
	valField := fd.MapValue()
	keyWiretag := wire.EncodeTag(1, wireTypes[keyField.Kind()])
	valWiretag := wire.EncodeTag(2, wireTypes[valField.Kind()])
	keyFuncs := encoderFuncsForValue(keyField, ft.Key())
	valFuncs := encoderFuncsForValue(valField, ft.Elem())

	mapi := &mapInfo{
		goType:     ft,
		keyWiretag: keyWiretag,
		valWiretag: valWiretag,
		keyFuncs:   keyFuncs,
		valFuncs:   valFuncs,
		keyZero:    reflect.Zero(ft.Key()).Interface(),
		valZero:    reflect.Zero(ft.Elem()).Interface(),
	}
	switch valField.Kind() {
	case pref.GroupKind, pref.MessageKind:
		mapi.newVal = func() interface{} {
			return reflect.New(ft.Elem().Elem()).Interface()
		}
	}

	funcs = pointerCoderFuncs{
		size: func(p pointer, tagsize int, opts marshalOptions) int {
			return sizeMap(p, tagsize, ft, keyFuncs, valFuncs, opts)
		},
		marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
			return appendMap(b, p, wiretag, keyWiretag, valWiretag, ft, keyFuncs, valFuncs, opts)
		},
		unmarshal: func(b []byte, p pointer, wtyp wire.Type, opts unmarshalOptions) (int, error) {
			return consumeMap(b, p, wtyp, mapi, opts)
		},
	}
	if valFuncs.isInit != nil {
		funcs.isInit = func(p pointer) error {
			return isInitMap(p, ft, valFuncs.isInit)
		}
	}
	return funcs
}

const (
	mapKeyTagSize = 1 // field 1, tag size 1.
	mapValTagSize = 1 // field 2, tag size 2.
)

func consumeMap(b []byte, p pointer, wtyp wire.Type, mapi *mapInfo, opts unmarshalOptions) (int, error) {
	mp := p.AsValueOf(mapi.goType)
	if mp.Elem().IsNil() {
		mp.Elem().Set(reflect.MakeMap(mapi.goType))
	}
	m := mp.Elem()

	if wtyp != wire.BytesType {
		return 0, errUnknown
	}
	b, n := wire.ConsumeBytes(b)
	if n < 0 {
		return 0, wire.ParseError(n)
	}
	var (
		key = mapi.keyZero
		val = mapi.valZero
	)
	if mapi.newVal != nil {
		val = mapi.newVal()
	}
	for len(b) > 0 {
		num, wtyp, n := wire.ConsumeTag(b)
		if n < 0 {
			return 0, wire.ParseError(n)
		}
		b = b[n:]
		err := errUnknown
		switch num {
		case 1:
			var v interface{}
			v, n, err = mapi.keyFuncs.unmarshal(b, key, num, wtyp, opts)
			if err != nil {
				break
			}
			key = v
		case 2:
			var v interface{}
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
	m.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(val))
	return n, nil
}

func sizeMap(p pointer, tagsize int, goType reflect.Type, keyFuncs, valFuncs ifaceCoderFuncs, opts marshalOptions) int {
	m := p.AsValueOf(goType).Elem()
	n := 0
	if m.Len() == 0 {
		return 0
	}
	iter := mapRange(m)
	for iter.Next() {
		ki := iter.Key().Interface()
		vi := iter.Value().Interface()
		size := keyFuncs.size(ki, mapKeyTagSize, opts) + valFuncs.size(vi, mapValTagSize, opts)
		n += wire.SizeBytes(size) + tagsize
	}
	return n
}

func appendMap(b []byte, p pointer, wiretag, keyWiretag, valWiretag uint64, goType reflect.Type, keyFuncs, valFuncs ifaceCoderFuncs, opts marshalOptions) ([]byte, error) {
	m := p.AsValueOf(goType).Elem()
	var err error

	if m.Len() == 0 {
		return b, nil
	}

	if opts.Deterministic() {
		keys := m.MapKeys()
		sort.Sort(mapKeys(keys))
		for _, k := range keys {
			b, err = appendMapElement(b, k, m.MapIndex(k), wiretag, keyWiretag, valWiretag, keyFuncs, valFuncs, opts)
			if err != nil {
				return b, err
			}
		}
		return b, nil
	}

	iter := mapRange(m)
	for iter.Next() {
		b, err = appendMapElement(b, iter.Key(), iter.Value(), wiretag, keyWiretag, valWiretag, keyFuncs, valFuncs, opts)
		if err != nil {
			return b, err
		}
	}
	return b, nil
}

func appendMapElement(b []byte, key, value reflect.Value, wiretag, keyWiretag, valWiretag uint64, keyFuncs, valFuncs ifaceCoderFuncs, opts marshalOptions) ([]byte, error) {
	ki := key.Interface()
	vi := value.Interface()
	b = wire.AppendVarint(b, wiretag)
	size := keyFuncs.size(ki, mapKeyTagSize, opts) + valFuncs.size(vi, mapValTagSize, opts)
	b = wire.AppendVarint(b, uint64(size))
	b, err := keyFuncs.marshal(b, ki, keyWiretag, opts)
	if err != nil {
		return b, err
	}
	b, err = valFuncs.marshal(b, vi, valWiretag, opts)
	if err != nil {
		return b, err
	}
	return b, nil
}

func isInitMap(p pointer, goType reflect.Type, isInit func(interface{}) error) error {
	m := p.AsValueOf(goType).Elem()
	if m.Len() == 0 {
		return nil
	}
	iter := mapRange(m)
	for iter.Next() {
		if err := isInit(iter.Value().Interface()); err != nil {
			return err
		}
	}
	return nil
}

// mapKeys returns a sort.Interface to be used for sorting the map keys.
// Map fields may have key types of non-float scalars, strings and enums.
func mapKeys(vs []reflect.Value) sort.Interface {
	s := mapKeySorter{vs: vs}

	// Type specialization per https://developers.google.com/protocol-buffers/docs/proto#maps.
	if len(vs) == 0 {
		return s
	}
	switch vs[0].Kind() {
	case reflect.Int32, reflect.Int64:
		s.less = func(a, b reflect.Value) bool { return a.Int() < b.Int() }
	case reflect.Uint32, reflect.Uint64:
		s.less = func(a, b reflect.Value) bool { return a.Uint() < b.Uint() }
	case reflect.Bool:
		s.less = func(a, b reflect.Value) bool { return !a.Bool() && b.Bool() } // false < true
	case reflect.String:
		s.less = func(a, b reflect.Value) bool { return a.String() < b.String() }
	default:
		panic(fmt.Sprintf("unsupported map key type: %v", vs[0].Kind()))
	}

	return s
}

type mapKeySorter struct {
	vs   []reflect.Value
	less func(a, b reflect.Value) bool
}

func (s mapKeySorter) Len() int      { return len(s.vs) }
func (s mapKeySorter) Swap(i, j int) { s.vs[i], s.vs[j] = s.vs[j], s.vs[i] }
func (s mapKeySorter) Less(i, j int) bool {
	return s.less(s.vs[i], s.vs[j])
}
