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
)

// Methods is a set of optional fast-path implementations of various operations.
type Methods = struct {
	pragma.NoUnkeyedLiterals

	// Flags indicate support for optional features.
	Flags SupportFlags

	// Size returns the size in bytes of the wire-format encoding of m.
	// MarshalAppend must be provided if a custom Size is provided.
	Size func(m protoreflect.Message, opts MarshalOptions) int

	// Marshal writes the wire-format encoding of m to the provided buffer.
	// Size should be provided if a custom MarshalAppend is provided.
	// It must not perform required field checks.
	Marshal func(m protoreflect.Message, in MarshalInput) (MarshalOutput, error)

	// Unmarshal parses the wire-format encoding of a message and merges the result to m.
	// It must not reset m or perform required field checks.
	Unmarshal func(m protoreflect.Message, in UnmarshalInput) (UnmarshalOutput, error)

	// IsInitialized returns an error if any required fields in m are not set.
	IsInitialized func(m protoreflect.Message) error
}

type SupportFlags = uint64

const (
	// SupportMarshalDeterministic reports whether MarshalOptions.Deterministic is supported.
	SupportMarshalDeterministic SupportFlags = 1 << iota

	// SupportUnmarshalDiscardUnknown reports whether UnmarshalOptions.DiscardUnknown is supported.
	SupportUnmarshalDiscardUnknown
)

// MarshalInput is input to the marshaler.
type MarshalInput = struct {
	pragma.NoUnkeyedLiterals

	Buf     []byte // output is appended to this buffer
	Options MarshalOptions
}

// MarshalOutput is output from the marshaler.
type MarshalOutput = struct {
	pragma.NoUnkeyedLiterals

	Buf []byte // contains marshaled message
}

// MarshalOptions configure the marshaler.
//
// This type is identical to the one in package proto.
type MarshalOptions = struct {
	pragma.NoUnkeyedLiterals

	AllowPartial  bool // must be treated as true by method implementations
	Deterministic bool
	UseCachedSize bool
}

// UnmarshalInput is input to the unmarshaler.
type UnmarshalInput = struct {
	pragma.NoUnkeyedLiterals

	Buf     []byte // input buffer
	Options UnmarshalOptions
}

// UnmarshalOutput is output from the unmarshaler.
type UnmarshalOutput = struct {
	pragma.NoUnkeyedLiterals

	// Contents available for future expansion.
}

// UnmarshalOptions configures the unmarshaler.
//
// This type is identical to the one in package proto.
type UnmarshalOptions = struct {
	pragma.NoUnkeyedLiterals

	Merge          bool // must be treated as true by method implementations
	AllowPartial   bool // must be treated as true by method implementations
	DiscardUnknown bool
	Resolver       interface {
		FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error)
		FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error)
	}
}
