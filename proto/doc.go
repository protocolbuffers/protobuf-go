// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package proto provides functions operating on protocol buffer messages.
//
// For documentation on protocol buffers in general, see:
//
//	https://developers.google.com/protocol-buffers
//
// For a tutorial on using protocol buffers with Go, see:
//
//	https://developers.google.com/protocol-buffers/docs/gotutorial
//
// For a guide to generated Go protocol buffer code, see:
//
//	https://developers.google.com/protocol-buffers/docs/reference/go-generated
//
// # Binary serialization
//
// This package contains functions to convert to and from the wire format,
// an efficient binary serialization of protocol buffers.
//
// • Size reports the size of a message in the wire format.
//
// • Marshal converts a message to the wire format.
// The MarshalOptions type provides more control over wire marshaling.
//
// • Unmarshal converts a message from the wire format.
// The UnmarshalOptions type provides more control over wire unmarshaling.
//
// # Basic message operations
//
// • Clone makes a deep copy of a message.
//
// • Merge merges the content of a message into another.
//
// • Equal compares two messages. For more control over comparisons
// and detailed reporting of differences, see package
// "github.com/infiniteloopcloud/protoc-gen-go-types/testing/protocmp".
//
// • Reset clears the content of a message.
//
// • CheckInitialized reports whether all required fields in a message are set.
//
// # Optional scalar constructors
//
// The API for some generated messages represents optional scalar fields
// as pointers to a value. For example, an optional string field has the
// Go type *string.
//
// • Bool, Int32, Int64, Uint32, Uint64, Float32, Float64, and String
// take a value and return a pointer to a new instance of it,
// to simplify construction of optional field values.
//
// Generated enum types usually have an Enum method which performs the
// same operation.
//
// Optional scalar fields are only supported in proto2.
//
// # Extension accessors
//
// • HasExtension, GetExtension, SetExtension, and ClearExtension
// access extension field values in a protocol buffer message.
//
// Extension fields are only supported in proto2.
//
// # Related packages
//
// • Package "github.com/infiniteloopcloud/protoc-gen-go-types/encoding/protojson" converts messages to
// and from JSON.
//
// • Package "github.com/infiniteloopcloud/protoc-gen-go-types/encoding/prototext" converts messages to
// and from the text format.
//
// • Package "github.com/infiniteloopcloud/protoc-gen-go-types/reflect/protoreflect" provides a
// reflection interface for protocol buffer data types.
//
// • Package "github.com/infiniteloopcloud/protoc-gen-go-types/testing/protocmp" provides features
// to compare protocol buffer messages with the "github.com/google/go-cmp/cmp"
// package.
//
// • Package "github.com/infiniteloopcloud/protoc-gen-go-types/types/dynamicpb" provides a dynamic
// message type, suitable for working with messages where the protocol buffer
// type is only known at runtime.
//
// This module contains additional packages for more specialized use cases.
// Consult the individual package documentation for details.
package proto
