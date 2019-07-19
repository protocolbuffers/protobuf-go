// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package internal_gengo

import (
	"sync"

	"google.golang.org/protobuf/compiler/protogen"
)

// messageAPIFlags provides flags that control the generated API.
type messageAPIFlags struct {
	WeakMapField bool
}

var messageAPIFlagsCache sync.Map

func loadMessageAPIFlags(message *protogen.Message) messageAPIFlags {
	if flags, ok := messageAPIFlagsCache.Load(message); ok {
		return flags.(messageAPIFlags)
	}

	var flags messageAPIFlags
	for _, field := range message.Fields {
		if field.Desc.IsWeak() {
			flags.WeakMapField = true
			break
		}
	}

	messageAPIFlagsCache.Store(message, flags)
	return flags
}
