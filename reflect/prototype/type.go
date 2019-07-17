// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package prototype provides constructors for protoreflect.EnumType,
// protoreflect.MessageType, and protoreflect.ExtensionType.
package prototype

import (
	"fmt"
	"reflect"
	"sync"

	"google.golang.org/protobuf/internal/descfmt"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Enum is a protoreflect.EnumType which combines a
// protoreflect.EnumDescriptor with a constructor function.
//
// Both EnumDescriptor and NewEnum must be populated.
// Once constructed, the exported fields must not be modified.
type Enum struct {
	protoreflect.EnumDescriptor

	// NewEnum constructs a new protoreflect.Enum representing the provided
	// enum number. The returned Go type must be identical for every call.
	NewEnum func(protoreflect.EnumNumber) protoreflect.Enum

	once   sync.Once
	goType reflect.Type
}

func (t *Enum) New(n protoreflect.EnumNumber) protoreflect.Enum {
	e := t.NewEnum(n)
	t.once.Do(func() {
		t.goType = reflect.TypeOf(e)
		if e.Descriptor() != t.Descriptor() {
			panic(fmt.Sprintf("mismatching enum descriptor: got %v, want %v", e.Descriptor(), t.Descriptor()))
		}
		if e.Descriptor().IsPlaceholder() {
			panic("enum descriptor must not be a placeholder")
		}
	})
	if t.goType != reflect.TypeOf(e) {
		panic(fmt.Sprintf("mismatching types for enum: got %T, want %v", e, t.goType))
	}
	return e
}

func (t *Enum) GoType() reflect.Type {
	t.New(0) // initialize t.typ
	return t.goType
}

func (t *Enum) Descriptor() protoreflect.EnumDescriptor {
	return t.EnumDescriptor
}

func (t *Enum) Format(s fmt.State, r rune) {
	descfmt.FormatDesc(s, r, t)
}

// Message is a protoreflect.MessageType which combines a
// protoreflect.MessageDescriptor with a constructor function.
//
// Both MessageDescriptor and NewMessage must be populated.
// Once constructed, the exported fields must not be modified.
type Message struct {
	protoreflect.MessageDescriptor

	// NewMessage constructs an empty, newly allocated protoreflect.Message.
	// The returned Go type must be identical for every call.
	NewMessage func() protoreflect.Message

	once   sync.Once
	goType reflect.Type
}

func (t *Message) New() protoreflect.Message {
	m := t.NewMessage()
	mi := m.Interface()
	t.once.Do(func() {
		t.goType = reflect.TypeOf(mi)
		if m.Descriptor() != t.Descriptor() {
			panic(fmt.Sprintf("mismatching message descriptor: got %v, want %v", m.Descriptor(), t.Descriptor()))
		}
		if m.Descriptor().IsPlaceholder() {
			panic("message descriptor must not be a placeholder")
		}
	})
	if t.goType != reflect.TypeOf(mi) {
		panic(fmt.Sprintf("mismatching types for message: got %T, want %v", mi, t.goType))
	}
	return m
}

func (t *Message) GoType() reflect.Type {
	t.New() // initialize t.goType
	return t.goType
}

func (t *Message) Descriptor() protoreflect.MessageDescriptor {
	return t.MessageDescriptor
}

func (t *Message) Format(s fmt.State, r rune) {
	descfmt.FormatDesc(s, r, t)
}

var (
	_ protoreflect.EnumType    = (*Enum)(nil)
	_ protoreflect.MessageType = (*Message)(nil)
)
