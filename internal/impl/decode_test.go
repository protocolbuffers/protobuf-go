// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"testing"
	"unsafe"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/internal/protolazy"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func TestUnmarshalPointerGroupNotLazy(t *testing.T) {
	type msg struct {
		lazy     *protolazy.XXX_lazyUnmarshalInfo
		presence [1]uint32
	}
	mi := &MessageInfo{
		coderMessageInfo: coderMessageInfo{
			lazyOffset:     offset(unsafe.Offsetof(msg{}.lazy)),
			presenceOffset: offset(unsafe.Offsetof(msg{}.presence)),
			presenceSize:   1,
		},
		initDone: 1,
	}
	opts := unmarshalOptions{resolver: protoregistry.GlobalTypes, depth: 10}

	var groupMsg msg
	b := protowire.AppendTag(nil, 1, protowire.EndGroupType)
	if _, err := mi.unmarshalPointer(b, pointer{p: unsafe.Pointer(&groupMsg)}, 1, opts); err != nil {
		t.Fatalf("unmarshalPointer(group) failed: %v", err)
	}
	if groupMsg.lazy != nil {
		t.Fatalf("groupMsg.lazy = %+v, want nil for group", groupMsg.lazy)
	}

	var subMsg msg
	if _, err := mi.unmarshalPointer(nil, pointer{p: unsafe.Pointer(&subMsg)}, 0, opts); err != nil {
		t.Fatalf("unmarshalPointer(non-group) failed: %v", err)
	}
	if LazyEnabled() && subMsg.lazy == nil {
		t.Fatalf("subMsg.lazy = nil, want non-nil when LazyEnabled")
	}
}
