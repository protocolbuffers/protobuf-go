// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package proto provides functions operating on protocol buffer messages.
//
// For documentation on protocol buffers in general, see:
//
//   https://developers.google.com/protocol-buffers
//
// For a tutorial on using protocol buffers with Go, see:
//
//   https://developers.google.com/protocol-buffers/docs/gotutorial
//
// For a guide to generated Go protocol buffer code, see:
//
//   https://developers.google.com/protocol-buffers/docs/reference/go-generated
//
//
// Binary serialization
//
// This package contains functions to convert to and from the wire format,
// an efficient binary serialization of protocol buffers.
//
// • Marshal converts a message to the wire format.
// The MarshalOptions type provides more control over wire marshaling.
//
// • Unmarshal converts a message from the wire format.
// The UnmarshalOptions type provides more control over wire unmarshaling.
//
// • Size reports the size of a message in the wire format.
//
//
// Basic message operations
//
// • Clone makes a deep copy of a message.
//
// • Equal compares two messages. For more control over comparisons
// and detailed reporting of differences, see package
// "google.golang.org/protobuf/testing/protocmp".
//
// • IsInitialized reports whether all required fields in a message are set.
//
// • Merge combines two messages. The MergeOptions type provides more
// control over merge operations.
//
// • Reset clears the contents of a message.
//
//
// Optional scalar constructors
//
// The API for some generated messages represents optional scalar fields
// as pointers to a value. For example, an optional string field has the
// Go type *string.
//
// • Bool, Float32, Float64, Int32, Int64, String, Uint32, and Uint64
// take a value and return a pointer to a new instance of it, to
// simplify construction of optional field values.
//
// Generated enum types usually have an Enum method which performs the
// same operation.
//
// Optional values are only supported in proto2.
//
//
// Extension accessors
//
// • GetExtension, SetExtension, ClearExtension, and HasExtension access
// extension field values in generated messages.
//
// Extensions are only supported in proto2.
//
//
// Related packages
//
// • Package "google.golang.org/protobuf/encoding/protojson" converts messages to
// and from JSON.
//
// • Package "google.golang.org/protobuf/encoding/prototext" converts messages to
// and from the text format.
//
// • Package "google.golang.org/protobuf/reflect/protoreflect" provides a
// reflection interface for protocol buffer data types.
//
// • Package "google.golang.org/protobuf/testing/protocmp" provides features
// to compare protocol buffer messages with the "github.com/google/go-cmp/cmp"
// package.
//
// • Package "google.golang.org/protobuf/types/dynamicpb" provides a dynamic
// message type, suitable for working with messages where the protocol buffer
// type is only known at runtime.
//
// This module contains additional packages for more specialized use cases.
// Consult the individual package documentation for details.
package proto
