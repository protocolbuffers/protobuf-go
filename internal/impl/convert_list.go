// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"reflect"

	pref "google.golang.org/protobuf/reflect/protoreflect"
)

func newListConverter(t reflect.Type, fd pref.FieldDescriptor) Converter {
	switch {
	case t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Slice:
		return &listPtrConverter{t, newSingularConverter(t.Elem().Elem(), fd)}
	case t.Kind() == reflect.Slice:
		return &listConverter{t, newSingularConverter(t.Elem(), fd)}
	}
	panic(fmt.Sprintf("invalid Go type %v for field %v", t, fd.FullName()))
}

type listConverter struct {
	goType reflect.Type
	c      Converter
}

func (c *listConverter) PBValueOf(v reflect.Value) pref.Value {
	if v.Type() != c.goType {
		panic(fmt.Sprintf("invalid type: got %v, want %v", v.Type(), c.goType))
	}
	pv := reflect.New(c.goType)
	pv.Elem().Set(v)
	return pref.ValueOf(&listReflect{pv, c.c})
}

func (c *listConverter) GoValueOf(v pref.Value) reflect.Value {
	rv := v.List().(*listReflect).v
	if rv.IsNil() {
		return reflect.Zero(c.goType)
	}
	return rv.Elem()
}

func (c *listConverter) IsValidPB(v pref.Value) bool {
	list, ok := v.Interface().(*listReflect)
	if !ok {
		return false
	}
	return list.v.Type().Elem() == c.goType
}

func (c *listConverter) IsValidGo(v reflect.Value) bool {
	return v.Type() == c.goType
}

func (c *listConverter) New() pref.Value {
	return pref.ValueOf(&listReflect{reflect.New(c.goType), c.c})
}

func (c *listConverter) Zero() pref.Value {
	return pref.ValueOf(&listReflect{reflect.Zero(reflect.PtrTo(c.goType)), c.c})
}

type listPtrConverter struct {
	goType reflect.Type
	c      Converter
}

func (c *listPtrConverter) PBValueOf(v reflect.Value) pref.Value {
	if v.Type() != c.goType {
		panic(fmt.Sprintf("invalid type: got %v, want %v", v.Type(), c.goType))
	}
	return pref.ValueOf(&listReflect{v, c.c})
}

func (c *listPtrConverter) GoValueOf(v pref.Value) reflect.Value {
	return v.List().(*listReflect).v
}

func (c *listPtrConverter) IsValidPB(v pref.Value) bool {
	list, ok := v.Interface().(*listReflect)
	if !ok {
		return false
	}
	return list.v.Type() == c.goType
}

func (c *listPtrConverter) IsValidGo(v reflect.Value) bool {
	return v.Type() == c.goType
}

func (c *listPtrConverter) New() pref.Value {
	return c.PBValueOf(reflect.New(c.goType.Elem()))
}

func (c *listPtrConverter) Zero() pref.Value {
	return c.PBValueOf(reflect.Zero(c.goType))
}

type listReflect struct {
	v    reflect.Value // *[]T
	conv Converter
}

func (ls *listReflect) Len() int {
	if ls.v.IsNil() {
		return 0
	}
	return ls.v.Elem().Len()
}
func (ls *listReflect) Get(i int) pref.Value {
	return ls.conv.PBValueOf(ls.v.Elem().Index(i))
}
func (ls *listReflect) Set(i int, v pref.Value) {
	ls.v.Elem().Index(i).Set(ls.conv.GoValueOf(v))
}
func (ls *listReflect) Append(v pref.Value) {
	ls.v.Elem().Set(reflect.Append(ls.v.Elem(), ls.conv.GoValueOf(v)))
}
func (ls *listReflect) Truncate(i int) {
	ls.v.Elem().Set(ls.v.Elem().Slice(0, i))
}
func (ls *listReflect) NewElement() pref.Value {
	return ls.conv.New()
}
func (ls *listReflect) protoUnwrap() interface{} {
	return ls.v.Interface()
}
