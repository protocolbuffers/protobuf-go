// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protoreflect

import (
	"google.golang.org/protobuf/internal/pragma"
)

// The following types are used by the fast-path Message.ProtoMethods method.
//
// To avoid polluting the public protoreflect API with types used only by
// low-level implementations, the canonical definitions of these types are
// in the runtime/protoiface package. The definitions here and in protoiface
// must be kept in sync.
type (
	methods = struct {
		pragma.NoUnkeyedLiterals
		Flags         supportFlags
		Size          func(Message, marshalOptions) int
		Marshal       func(Message, marshalInput, marshalOptions) (marshalOutput, error)
		Unmarshal     func(Message, unmarshalInput, unmarshalOptions) (unmarshalOutput, error)
		IsInitialized func(Message) error
		Merge         func(Message, Message, mergeInput, mergeOptions) mergeOutput
	}
	supportFlags = uint64
	marshalInput = struct {
		pragma.NoUnkeyedLiterals
		Buf []byte
	}
	marshalOutput = struct {
		pragma.NoUnkeyedLiterals
		Buf []byte
	}
	marshalOptions = struct {
		pragma.NoUnkeyedLiterals
		Flags uint8
	}
	unmarshalInput = struct {
		pragma.NoUnkeyedLiterals
		Buf []byte
	}
	unmarshalOutput = struct {
		pragma.NoUnkeyedLiterals
		Initialized bool
	}
	unmarshalOptions = struct {
		pragma.NoUnkeyedLiterals
		Flags    uint8
		Resolver interface {
			FindExtensionByName(field FullName) (ExtensionType, error)
			FindExtensionByNumber(message FullName, field FieldNumber) (ExtensionType, error)
		}
	}
	mergeInput = struct {
		pragma.NoUnkeyedLiterals
	}
	mergeOutput = struct {
		pragma.NoUnkeyedLiterals
		Merged bool
	}
	mergeOptions = struct {
		pragma.NoUnkeyedLiterals
	}
)
