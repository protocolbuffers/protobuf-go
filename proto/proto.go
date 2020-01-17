// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import (
	"google.golang.org/protobuf/internal/errors"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Message is the top-level interface that all messages must implement.
type Message = protoreflect.ProtoMessage

// Error matches all errors produced by packages in the protobuf module.
//
// That is, errors.Is(err, Error) reports whether an error is produced
// by this module.
var Error error

func init() {
	Error = errors.Error
}
