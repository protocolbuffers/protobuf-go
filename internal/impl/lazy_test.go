// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl_test

import (
	"testing"

	"github.com/infiniteloopcloud/protoc-gen-go-types/internal/flags"
	"github.com/infiniteloopcloud/protoc-gen-go-types/internal/impl"
	"github.com/infiniteloopcloud/protoc-gen-go-types/internal/protobuild"
	"github.com/infiniteloopcloud/protoc-gen-go-types/proto"

	testpb "github.com/infiniteloopcloud/protoc-gen-go-types/internal/testprotos/test"
)

func TestLazyExtensions(t *testing.T) {
	checkLazy := func(when string, m *testpb.TestAllExtensions, want bool) {
		xd := testpb.E_OptionalNestedMessage.TypeDescriptor()
		if got := impl.IsLazy(m.ProtoReflect(), xd); got != want {
			t.Errorf("%v: m.optional_nested_message lazy=%v, want %v", when, got, want)
		}
		e := proto.GetExtension(m, testpb.E_OptionalNestedMessage).(*testpb.TestAllExtensions_NestedMessage).Corecursive
		if got := impl.IsLazy(e.ProtoReflect(), xd); got != want {
			t.Errorf("%v: m.optional_nested_message.corecursive.optional_nested_message lazy=%v, want %v", when, got, want)
		}
	}

	m1 := &testpb.TestAllExtensions{}
	protobuild.Message{
		"optional_nested_message": protobuild.Message{
			"a": 1,
			"corecursive": protobuild.Message{
				"optional_nested_message": protobuild.Message{
					"a": 2,
				},
			},
		},
	}.Build(m1.ProtoReflect())
	checkLazy("before unmarshal", m1, false)

	w, err := proto.Marshal(m1)
	if err != nil {
		t.Fatal(err)
	}
	m := &testpb.TestAllExtensions{}
	if err := proto.Unmarshal(w, m); err != nil {
		t.Fatal(err)
	}
	checkLazy("after unmarshal", m, flags.LazyUnmarshalExtensions)
}
