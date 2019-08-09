// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package internal_gengo

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/internal/encoding/wire"

	"google.golang.org/protobuf/types/descriptorpb"
)

// messageFlags provides flags that control the generated API.
type messageFlags struct {
	IsTracked bool
	HasWeak   bool
}

func loadMessageFlags(message *protogen.Message) messageFlags {
	var flags messageFlags
	flags.IsTracked = isTrackedMessage(message)
	for _, field := range message.Fields {
		if field.Desc.IsWeak() {
			flags.HasWeak = true
			break
		}
	}
	return flags
}

// isTrackedMessage reports whether field tracking is enabled on the message.
// It is a variable so that the behavior is easily overridden in another file.
var isTrackedMessage = func(message *protogen.Message) (tracked bool) {
	const trackFieldUse_fieldNumber = 37383685

	// Decode the option from unknown fields to avoid a dependency on the
	// annotation proto from protoc-gen-go.
	b := message.Desc.Options().(*descriptorpb.MessageOptions).ProtoReflect().GetUnknown()
	for len(b) > 0 {
		num, typ, n := wire.ConsumeTag(b)
		b = b[n:]
		if num == trackFieldUse_fieldNumber && typ == wire.VarintType {
			v, _ := wire.ConsumeVarint(b)
			tracked = wire.DecodeBool(v)
		}
		m := wire.ConsumeFieldValue(num, typ, b)
		b = b[m:]
	}
	return tracked
}
