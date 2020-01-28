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
	// It should not return an error for a partial message.
	Marshal func(m protoreflect.Message, in MarshalInput, opts MarshalOptions) (MarshalOutput, error)

	// Unmarshal parses the wire-format encoding of a message and merges the result to m.
	// It should not reset the target message or return an error for a partial message.
	Unmarshal func(m protoreflect.Message, in UnmarshalInput, opts UnmarshalOptions) (UnmarshalOutput, error)

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

	Buf []byte // output is appended to this buffer
}

// MarshalOutput is output from the marshaler.
type MarshalOutput = struct {
	pragma.NoUnkeyedLiterals

	Buf []byte // contains marshaled message
}

// MarshalOptions configure the marshaler.
type MarshalOptions = struct {
	pragma.NoUnkeyedLiterals

	Flags MarshalFlags
}

// MarshalFlags are configure the marshaler.
// Most flags correspond to fields in proto.MarshalOptions.
type MarshalFlags = uint8

const (
	MarshalDeterministic MarshalFlags = 1 << iota
	MarshalUseCachedSize
)

// UnmarshalInput is input to the unmarshaler.
type UnmarshalInput = struct {
	pragma.NoUnkeyedLiterals

	Buf []byte // input buffer
}

// UnmarshalOutput is output from the unmarshaler.
type UnmarshalOutput = struct {
	pragma.NoUnkeyedLiterals

	// Initialized may be set on return if all required fields are known to be set.
	// A value of false does not indicate that the message is uninitialized, only
	// that its status could not be confirmed.
	Initialized bool
}

// UnmarshalOptions configures the unmarshaler.
type UnmarshalOptions = struct {
	pragma.NoUnkeyedLiterals

	Flags    UnmarshalFlags
	Resolver interface {
		FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error)
		FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error)
	}
}

// UnmarshalFlags configure the unmarshaler.
// Most flags correspond to fields in proto.UnmarshalOptions.
type UnmarshalFlags = uint8

const (
	UnmarshalDiscardUnknown UnmarshalFlags = 1 << iota
)
