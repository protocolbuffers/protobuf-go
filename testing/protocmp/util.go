// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protocmp

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/google/go-cmp/cmp"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	enumReflectType    = reflect.TypeOf(Enum{})
	messageReflectType = reflect.TypeOf(Message{})
)

// IgnoreEnums ignores all enums of the specified types.
// See IgnoreDescriptors with regard to EnumDescriptors for more information.
//
// This must be used in conjunction with Transform.
func IgnoreEnums(enums ...protoreflect.Enum) cmp.Option {
	var ds []protoreflect.Descriptor
	for _, e := range enums {
		ds = append(ds, e.Descriptor())
	}
	return IgnoreDescriptors(ds...)
}

// IgnoreMessages ignores all messages of the specified types.
// See IgnoreDescriptors with regard to MessageDescriptors for more information.
//
// This must be used in conjunction with Transform.
func IgnoreMessages(messages ...proto.Message) cmp.Option {
	var ds []protoreflect.Descriptor
	for _, m := range messages {
		ds = append(ds, m.ProtoReflect().Descriptor())
	}
	return IgnoreDescriptors(ds...)
}

// IgnoreFields ignores the specified fields in messages of type m.
// This panics if a field of the given name does not exist.
// See IgnoreDescriptors with regard to FieldDescriptors for more information.
//
// This must be used in conjunction with Transform.
func IgnoreFields(message proto.Message, names ...protoreflect.Name) cmp.Option {
	var ds []protoreflect.Descriptor
	md := message.ProtoReflect().Descriptor()
	for _, s := range names {
		ds = append(ds, mustFindFieldDescriptor(md, s))
	}
	return IgnoreDescriptors(ds...)
}

// IgnoreOneofs ignores fields in the specified oneofs in messages of type m.
// This panics if a oneof of the given name does not exist.
// See IgnoreDescriptors with regard to OneofDescriptors for more information.
//
// This must be used in conjunction with Transform.
func IgnoreOneofs(message proto.Message, names ...protoreflect.Name) cmp.Option {
	var ds []protoreflect.Descriptor
	md := message.ProtoReflect().Descriptor()
	for _, s := range names {
		ds = append(ds, mustFindOneofDescriptor(md, s))
	}
	return IgnoreDescriptors(ds...)
}

// IgnoreDescriptors ignores the specified set of descriptors.
// The following descriptor types may be specified:
//
// • EnumDescriptor: Enums of this type or messages containing singular fields,
// list fields, or map fields with enum values of this type are ignored.
// Enums are matched based on their full name.
//
// • MessageDescriptor: Messages of this type or messages containing
// singular fields, list fields, or map fields with message values of this type
// are ignored. Messages are matched based on their full name.
//
// • ExtensionDescriptor: Extensions fields that match the given descriptor
// by full name are ignored.
//
// • FieldDescriptor: Message fields that match the given descriptor
// by full name are ignored.
//
// • OneofDescriptor: Message fields that match the set of fields in the given
// oneof descriptor by full name are ignored.
//
// This must be used in conjunction with Transform.
func IgnoreDescriptors(descs ...protoreflect.Descriptor) cmp.Option {
	return cmp.FilterPath(newNameFilters(descs...).Filter, cmp.Ignore())
}

func mustFindFieldDescriptor(md protoreflect.MessageDescriptor, s protoreflect.Name) protoreflect.FieldDescriptor {
	d := findDescriptor(md, s)
	if fd, ok := d.(protoreflect.FieldDescriptor); ok && fd.Name() == s {
		return fd
	}

	var suggestion string
	switch d.(type) {
	case protoreflect.FieldDescriptor:
		suggestion = fmt.Sprintf("; consider specifying field %q instead", d.Name())
	case protoreflect.OneofDescriptor:
		suggestion = fmt.Sprintf("; consider specifying oneof %q with IgnoreOneofs instead", d.Name())
	}
	panic(fmt.Sprintf("message %q has no field %q%s", md.FullName(), s, suggestion))
}

func mustFindOneofDescriptor(md protoreflect.MessageDescriptor, s protoreflect.Name) protoreflect.OneofDescriptor {
	d := findDescriptor(md, s)
	if od, ok := d.(protoreflect.OneofDescriptor); ok && d.Name() == s {
		return od
	}

	var suggestion string
	switch d.(type) {
	case protoreflect.OneofDescriptor:
		suggestion = fmt.Sprintf("; consider specifying oneof %q instead", d.Name())
	case protoreflect.FieldDescriptor:
		suggestion = fmt.Sprintf("; consider specifying field %q with IgnoreFields instead", d.Name())
	}
	panic(fmt.Sprintf("message %q has no oneof %q%s", md.FullName(), s, suggestion))
}

func findDescriptor(md protoreflect.MessageDescriptor, s protoreflect.Name) protoreflect.Descriptor {
	// Exact match.
	if fd := md.Fields().ByName(s); fd != nil {
		return fd
	}
	if od := md.Oneofs().ByName(s); od != nil {
		return od
	}

	// Best-effort match.
	//
	// It's a common user mistake to use the CameCased field name as it appears
	// in the generated Go struct. Instead of complaining that it doesn't exist,
	// suggest the real protobuf name that the user may have desired.
	normalize := func(s protoreflect.Name) string {
		return strings.Replace(strings.ToLower(string(s)), "_", "", -1)
	}
	for i := 0; i < md.Fields().Len(); i++ {
		if fd := md.Fields().Get(i); normalize(fd.Name()) == normalize(s) {
			return fd
		}
	}
	for i := 0; i < md.Oneofs().Len(); i++ {
		if od := md.Oneofs().Get(i); normalize(od.Name()) == normalize(s) {
			return od
		}
	}
	return nil
}

type nameFilters struct {
	names map[protoreflect.FullName]bool
}

func newNameFilters(descs ...protoreflect.Descriptor) *nameFilters {
	f := &nameFilters{names: make(map[protoreflect.FullName]bool)}
	for _, d := range descs {
		switch d := d.(type) {
		case protoreflect.EnumDescriptor:
			f.names[d.FullName()] = true
		case protoreflect.MessageDescriptor:
			f.names[d.FullName()] = true
		case protoreflect.FieldDescriptor:
			f.names[d.FullName()] = true
		case protoreflect.OneofDescriptor:
			for i := 0; i < d.Fields().Len(); i++ {
				f.names[d.Fields().Get(i).FullName()] = true
			}
		default:
			panic("invalid descriptor type")
		}
	}
	return f
}

func (f *nameFilters) Filter(p cmp.Path) bool {
	vx, vy := p.Last().Values()
	return (f.filterValue(vx) && f.filterValue(vy)) || f.filterFields(p)
}

func (f *nameFilters) filterFields(p cmp.Path) bool {
	// Filter for Message maps.
	mi, ok := p.Index(-1).(cmp.MapIndex)
	if !ok {
		return false
	}
	ps := p.Index(-2)
	if ps.Type() != messageReflectType {
		return false
	}

	// Check field name.
	vx, vy := ps.Values()
	mx := vx.Interface().(Message)
	my := vy.Interface().(Message)
	k := mi.Key().String()
	if f.filterFieldName(mx, k) && f.filterFieldName(my, k) {
		return true
	}

	// Check field value.
	vx, vy = mi.Values()
	if f.filterFieldValue(vx) && f.filterFieldValue(vy) {
		return true
	}

	return false
}

func (f *nameFilters) filterFieldName(m Message, k string) bool {
	if md := m.Descriptor(); md != nil {
		switch {
		case protoreflect.Name(k).IsValid():
			return f.names[md.Fields().ByName(protoreflect.Name(k)).FullName()]
		case strings.HasPrefix(k, "[") && strings.HasSuffix(k, "]"):
			return f.names[protoreflect.FullName(k[1:len(k)-1])]
		}
	}
	return false
}

func (f *nameFilters) filterFieldValue(v reflect.Value) bool {
	if !v.IsValid() {
		return true // implies missing slice element or map entry
	}
	v = v.Elem() // map entries are always populated values
	switch t := v.Type(); {
	case t == enumReflectType || t == messageReflectType:
		// Check for singular message or enum field.
		return f.filterValue(v)
	case t.Kind() == reflect.Slice && (t.Elem() == enumReflectType || t.Elem() == messageReflectType):
		// Check for list field of enum or message type.
		return f.filterValue(v.Index(0))
	case t.Kind() == reflect.Map && (t.Elem() == enumReflectType || t.Elem() == messageReflectType):
		// Check for map field of enum or message type.
		return f.filterValue(v.MapIndex(v.MapKeys()[0]))
	}
	return false
}

func (f *nameFilters) filterValue(v reflect.Value) bool {
	if !v.IsValid() {
		return true // implies missing slice element or map entry
	}
	if !v.CanInterface() {
		return false // implies unexported struct field
	}
	switch v := v.Interface().(type) {
	case Enum:
		return v.Descriptor() != nil && f.names[v.Descriptor().FullName()]
	case Message:
		return v.Descriptor() != nil && f.names[v.Descriptor().FullName()]
	}
	return false
}

// IgnoreDefaultScalars ignores singular scalars that are unpopulated or
// explicitly set to the default value.
// This option does not effect elements in a list or entries in a map.
//
// This must be used in conjunction with Transform.
func IgnoreDefaultScalars() cmp.Option {
	return cmp.FilterPath(func(p cmp.Path) bool {
		// Filter for Message maps.
		mi, ok := p.Index(-1).(cmp.MapIndex)
		if !ok {
			return false
		}
		ps := p.Index(-2)
		if ps.Type() != messageReflectType {
			return false
		}

		// Check whether both fields are default or unpopulated scalars.
		vx, vy := ps.Values()
		mx := vx.Interface().(Message)
		my := vy.Interface().(Message)
		k := mi.Key().String()
		return isDefaultScalar(mx, k) && isDefaultScalar(my, k)
	}, cmp.Ignore())
}

func isDefaultScalar(m Message, k string) bool {
	if _, ok := m[k]; !ok {
		return true
	}

	var fd protoreflect.FieldDescriptor
	switch mt := m[messageTypeKey].(messageType); {
	case protoreflect.Name(k).IsValid():
		fd = mt.md.Fields().ByName(protoreflect.Name(k))
	case strings.HasPrefix(k, "[") && strings.HasSuffix(k, "]"):
		fd = mt.xds[protoreflect.FullName(k[1:len(k)-1])]
	}
	if fd == nil || !fd.Default().IsValid() {
		return false
	}
	switch fd.Kind() {
	case protoreflect.BytesKind:
		v, ok := m[k].([]byte)
		return ok && bytes.Equal(fd.Default().Bytes(), v)
	case protoreflect.FloatKind:
		v, ok := m[k].(float32)
		return ok && equalFloat64(fd.Default().Float(), float64(v))
	case protoreflect.DoubleKind:
		v, ok := m[k].(float64)
		return ok && equalFloat64(fd.Default().Float(), float64(v))
	case protoreflect.EnumKind:
		v, ok := m[k].(Enum)
		return ok && fd.Default().Enum() == v.Number()
	default:
		return reflect.DeepEqual(fd.Default().Interface(), m[k])
	}
}

func equalFloat64(x, y float64) bool {
	return x == y || (math.IsNaN(x) && math.IsNaN(y))
}

// IgnoreEmptyMessages ignores messages that are empty or unpopulated.
// It applies to standalone Messages, singular message fields,
// list fields of messages, and map fields of message values.
//
// This must be used in conjunction with Transform.
func IgnoreEmptyMessages() cmp.Option {
	return cmp.FilterPath(func(p cmp.Path) bool {
		vx, vy := p.Last().Values()
		return (isEmptyMessage(vx) && isEmptyMessage(vy)) || isEmptyMessageFields(p)
	}, cmp.Ignore())
}

func isEmptyMessageFields(p cmp.Path) bool {
	// Filter for Message maps.
	mi, ok := p.Index(-1).(cmp.MapIndex)
	if !ok {
		return false
	}
	ps := p.Index(-2)
	if ps.Type() != messageReflectType {
		return false
	}

	// Check field value.
	vx, vy := mi.Values()
	if isEmptyMessageFieldValue(vx) && isEmptyMessageFieldValue(vy) {
		return true
	}

	return false
}

func isEmptyMessageFieldValue(v reflect.Value) bool {
	if !v.IsValid() {
		return true // implies missing slice element or map entry
	}
	v = v.Elem() // map entries are always populated values
	switch t := v.Type(); {
	case t == messageReflectType:
		// Check singular field for empty message.
		if !isEmptyMessage(v) {
			return false
		}
	case t.Kind() == reflect.Slice && t.Elem() == messageReflectType:
		// Check list field for all empty message elements.
		for i := 0; i < v.Len(); i++ {
			if !isEmptyMessage(v.Index(i)) {
				return false
			}
		}
	case t.Kind() == reflect.Map && t.Elem() == messageReflectType:
		// Check map field for all empty message values.
		for _, k := range v.MapKeys() {
			if !isEmptyMessage(v.MapIndex(k)) {
				return false
			}
		}
	default:
		return false
	}
	return true
}

func isEmptyMessage(v reflect.Value) bool {
	if !v.IsValid() {
		return true // implies missing slice element or map entry
	}
	if !v.CanInterface() {
		return false // implies unexported struct field
	}
	if m, ok := v.Interface().(Message); ok {
		return len(m) == 0 || (len(m) == 1 && m[messageTypeKey] != nil)
	}
	return false
}

// IgnoreUnknown ignores unknown fields in all messages.
//
// This must be used in conjunction with Transform.
func IgnoreUnknown() cmp.Option {
	return cmp.FilterPath(func(p cmp.Path) bool {
		// Filter for Message maps.
		mi, ok := p.Index(-1).(cmp.MapIndex)
		if !ok {
			return false
		}
		ps := p.Index(-2)
		if ps.Type() != messageReflectType {
			return false
		}

		// Filter for unknown fields (which always have a numeric map key).
		return strings.Trim(mi.Key().String(), "0123456789") == ""
	}, cmp.Ignore())
}
