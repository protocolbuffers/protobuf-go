// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"reflect"

	pref "google.golang.org/protobuf/reflect/protoreflect"
)

type listConverter struct {
	goType reflect.Type
	c      Converter
}

func newListConverter(t reflect.Type, fd pref.FieldDescriptor) Converter {
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Slice {
		panic(fmt.Sprintf("invalid Go type %v for field %v", t, fd.FullName()))
	}
	return &listConverter{t, newSingularConverter(t.Elem().Elem(), fd)}
}

func (c *listConverter) PBValueOf(v reflect.Value) pref.Value {
	if v.Type() != c.goType {
		panic(fmt.Sprintf("invalid type: got %v, want %v", v.Type(), c.goType))
	}
	return pref.ValueOf(&listReflect{v, c.c})
}

func (c *listConverter) GoValueOf(v pref.Value) reflect.Value {
	return v.List().(*listReflect).v
}

func (c *listConverter) New() pref.Value {
	return c.PBValueOf(reflect.New(c.goType.Elem()))
}

func (c *listConverter) Zero() pref.Value {
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
func (ls *listReflect) NewMessage() pref.Message {
	return ls.NewElement().Message()
}
func (ls *listReflect) NewElement() pref.Value {
	return ls.conv.New()
}
func (ls *listReflect) ProtoUnwrap() interface{} {
	return ls.v.Interface()
}
