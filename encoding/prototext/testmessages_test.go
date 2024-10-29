// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototext_test

import (
	"google.golang.org/protobuf/internal/protobuild"
	"google.golang.org/protobuf/proto"
)

func makeMessages(in protobuild.Message, messages ...proto.Message) []proto.Message {
	for _, m := range messages {
		in.Build(m.ProtoReflect())
	}
	return messages
}
