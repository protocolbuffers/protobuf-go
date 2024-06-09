// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protoreflect

import (
	"bytes"
	"cmp"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"google.golang.org/protobuf/encoding/protowire"
)

// Compare compares the elements of v1 and v2, using [cmp.Compare] recursively
// on each field of elements. For unequal types we break the tie by the name of the
// type.
//
// This function implements the 3 properties required to be used as a Less function
// by Go's "sort" package
//
//   - Deterministic: less(x, y) == less(x, y)
//   - Irreflexive: !less(x, x)
//   - Transitive: if !less(x, y) and !less(y, z), then !less(x, z)
func (v1 Value) Compare(v2 Value) int {
	return valueCompare(v1, v2)
}

func valueCompare(x, y Value) int {
	if !x.IsValid() || !y.IsValid() {
		return boolCompare(x.IsValid(), y.IsValid())
	}

	if typCmp := stringCompare(x.typeName(), y.typeName()); typCmp != 0 {
		return typCmp
	}

	switch x.typ {
	case nilType:
		return 0
	case boolType:
		return boolCompare(x.Bool(), y.Bool())
	case int32Type, int64Type:
		return cmp.Compare(x.Int(), y.Int())
	case uint32Type, uint64Type:
		return cmp.Compare(x.Uint(), y.Uint())
	case float32Type, float64Type:
		return cmp.Compare(x.Float(), y.Float())
	case stringType:
		return stringCompare(x.String(), y.String())
	case bytesType:
		return bytes.Compare(x.Bytes(), y.Bytes())
	case enumType:
		return cmp.Compare(x.Enum(), y.Enum())
	default:
		switch xv := x.Interface().(type) {
		case Message:
			yv, ok := y.Interface().(Message)
			if !ok {
				return goTypeCompare(xv, y.Interface())
			}

			return messageCompare(xv, yv)
		case List:
			yv, ok := y.Interface().(List)
			if !ok {
				return goTypeCompare(xv, y.Interface())
			}

			return listCompare(xv, yv)
		case Map:
			yv, ok := y.Interface().(Map)
			if !ok {
				return goTypeCompare(xv, y.Interface())
			}

			return mapCompare(xv, yv)
		default:
			panic(fmt.Sprintf("unknown type: %T", x))
		}
	}
}

func goTypeCompare(x, y any) int {
	xType := reflect.TypeOf(x)
	yType := reflect.TypeOf(y)

	return stringCompare(xType.String(), yType.String())
}

// messageCompare reports whether mx is less than my.
func messageCompare(mx, my Message) int {
	if mx.Descriptor() != my.Descriptor() {
		return stringCompare(mx.Descriptor().FullName(), my.Descriptor().FullName())
	}

	xFields := getFieldVals(mx)
	yFields := getFieldVals(my)

	if fieldsCmp := sortAndCompareSlice(xFields, yFields, valueCompare); fieldsCmp != 0 {
		return fieldsCmp
	}

	return unknownCompare(mx.GetUnknown(), my.GetUnknown())
}

func getFieldVals(m Message) []Value {
	var out []Value
	m.Range(func(_ FieldDescriptor, val Value) bool {
		out = append(out, val)
		return true
	})

	return out
}

// listCompare compares two lists.
func listCompare(x, y List) int {
	if lengthCmp := cmp.Compare(x.Len(), y.Len()); lengthCmp != 0 {
		return lengthCmp
	}

	for i := 0; i < x.Len(); i++ {
		if valCmp := valueCompare(x.Get(i), y.Get(i)); valCmp != 0 {
			return valCmp
		}
	}

	return 0
}

// mapCompare return true if x is smaller than y.
// A map is considered smaller than another map if one of the following cases is true
//
//  1. It has less elements
//  2. If the keys we organized into lists, the map would have the smaller list.
//  3. When iterating in sorted key order, the map has a smaller value.
func mapCompare(x, y Map) int {
	if lengthCmp := cmp.Compare(x.Len(), y.Len()); lengthCmp != 0 {
		return lengthCmp
	}

	// In Go iterating over maps has undetermined order. So, we need to collect the keys
	// and sort them to ensure the LessThan function is deterministic.
	xKeys := getMapKeys(x)
	yKeys := getMapKeys(y)

	if keyCmp := sortAndCompareSlice(xKeys, yKeys, mapKeyCompare); keyCmp != 0 {
		// If key slices aren't the same, we return the smaller one.
		return keyCmp
	}

	for _, k := range xKeys {
		vx := x.Get(k)
		vy := y.Get(k)

		if valCmp := valueCompare(vx, vy); valCmp != 0 {
			// For the smallest key with value mismatch return the first not matching value.
			return valCmp
		}
	}

	return 0
}

// unknownCompare compares two unknown fields.
func unknownCompare(x, y []byte) int {
	if len(x) != len(y) {
		return cmp.Compare(len(x), len(y))
	}

	// If we one has a smaller byte array we are done
	if byteCmp := bytes.Compare(x, y); byteCmp != 0 {
		return byteCmp
	}

	// Before saying the two fields are equal we need to ensure each byte
	// is read to the same field.

	mx := make(map[FieldNumber][]byte)
	my := make(map[FieldNumber][]byte)
	for len(x) > 0 {
		fnum, _, n := protowire.ConsumeField(x)
		mx[fnum] = append(mx[fnum], x[:n]...)
		x = x[n:]
	}
	for len(y) > 0 {
		fnum, _, n := protowire.ConsumeField(y)
		my[fnum] = append(my[fnum], y[:n]...)
		y = y[n:]
	}

	// In Go iterating over maps has undetermined order. So, we need to collect the keys
	// and sort them to ensure the LessThan function is deterministic.
	xKeys := make([]FieldNumber, 0, len(mx))
	for fnum := range mx {
		xKeys = append(xKeys, fnum)
	}
	slices.Sort(xKeys)

	yKeys := make([]FieldNumber, 0, len(my))
	for fnum := range my {
		yKeys = append(yKeys, fnum)
	}
	slices.Sort(yKeys)

	if sliceCmp := slices.Compare(xKeys, yKeys); sliceCmp != 0 {
		return sliceCmp
	}

	for _, k := range xKeys {
		if byteCmp := bytes.Compare(mx[k], my[k]); byteCmp != 0 {
			return byteCmp
		}
	}

	return 0
}

func sortAndCompareSlice[S ~[]E, E any](s1, s2 S, compare func(E, E) int) int {
	if len(s1) != len(s2) {
		// Avoid sorting if we can determine comparison by just length.
		return cmp.Compare(len(s1), len(s2))
	}

	slices.SortFunc(s1, compare)
	slices.SortFunc(s2, compare)

	return slices.CompareFunc(s1, s2, compare)
}

func getMapKeys(m Map) []MapKey {
	if m.Len() == 0 {
		return nil
	}

	out := make([]MapKey, 0, m.Len())
	m.Range(func(k MapKey, _ Value) bool {
		out = append(out, k)
		return true
	})

	return out
}

func mapKeyCompare(x, y MapKey) int {
	return valueCompare(x.Value(), y.Value())
}

func stringCompare[T ~string](x, y T) int {
	return strings.Compare(string(x), string(y))
}

func boolCompare[T ~bool](x, y T) int {
	if x == y {
		return 0
	}

	if !x {
		return -1
	}

	return 1
}
