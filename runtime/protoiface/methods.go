// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package protoiface contains types referenced or implemented by messages.
//
// WARNING: This package should only be imported by message implementations.
// The functionality found in this package should be accessed through
// higher-level abstractions provided by the proto package.
package protoiface

import (
	"google.golang.org/protobuf/internal/pragma"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// Methoder is an optional interface implemented by protoreflect.Message to
// provide fast-path implementations of various operations.
// The returned Methods struct must not be mutated.
type Methoder interface {
	ProtoMethods() *Methods // may return nil
}

// Methods is a set of optional fast-path implementations of various operations.
type Methods struct {
	pragma.NoUnkeyedLiterals

	// Flags indicate support for optional features.
	Flags SupportFlags

	// Size returns the size in bytes of the wire-format encoding of m.
	// MarshalAppend must be provided if a custom Size is provided.
	Size func(m protoreflect.Message, opts MarshalOptions) int

	// MarshalAppend appends the wire-format encoding of m to b, returning the result.
	// Size must be provided if a custom MarshalAppend is provided.
	// It must not perform required field checks.
	MarshalAppend func(b []byte, m protoreflect.Message, opts MarshalOptions) ([]byte, error)

	// Unmarshal parses the wire-format message in b and merges the result in m.
	// It must not reset m or perform required field checks.
	Unmarshal func(b []byte, m protoreflect.Message, opts UnmarshalOptions) error

	// IsInitialized returns an error if any required fields in m are not set.
	IsInitialized func(m protoreflect.Message) error
}

type SupportFlags uint64

const (
	// SupportMarshalDeterministic reports whether MarshalOptions.Deterministic is supported.
	SupportMarshalDeterministic SupportFlags = 1 << iota

	// SupportUnmarshalDiscardUnknown reports whether UnmarshalOptions.DiscardUnknown is supported.
	SupportUnmarshalDiscardUnknown
)

// MarshalOptions configure the marshaler.
//
// This type is identical to the one in package proto.
type MarshalOptions struct {
	pragma.NoUnkeyedLiterals

	AllowPartial  bool // must be treated as true by method implementations
	Deterministic bool
	UseCachedSize bool
}

// UnmarshalOptions configures the unmarshaler.
//
// This type is identical to the one in package proto.
type UnmarshalOptions struct {
	pragma.NoUnkeyedLiterals

	AllowPartial   bool // must be treated as true by method implementations
	DiscardUnknown bool
	Resolver       interface {
		protoregistry.ExtensionTypeResolver
	}
}
