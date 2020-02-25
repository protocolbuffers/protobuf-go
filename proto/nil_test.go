// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto_test

import (
	"testing"

	"google.golang.org/protobuf/proto"

	testpb "google.golang.org/protobuf/internal/testprotos/test"
)

// TestNil tests for boundary conditions when nil and typed-nil messages
// are passed to various top-level functions.
// These tests are not necessarily a statement of proper behavior,
// but exist to detect accidental changes in behavior.
func TestNil(t *testing.T) {
	nilMsg := (*testpb.TestAllExtensions)(nil)
	extType := testpb.E_OptionalBool

	tests := []struct {
		label string
		test  func()
		panic bool
	}{{
		label: "Size",
		test:  func() { proto.Size(nil) },
		panic: true,
	}, {
		label: "Size",
		test:  func() { proto.Size(nilMsg) },
	}, {
		label: "Marshal",
		test:  func() { proto.Marshal(nil) },
		panic: true,
	}, {
		label: "Marshal",
		test:  func() { proto.Marshal(nilMsg) },
	}, {
		label: "Unmarshal",
		test:  func() { proto.Unmarshal(nil, nil) },
		panic: true,
	}, {
		label: "Unmarshal",
		test:  func() { proto.Unmarshal(nil, nilMsg) },
		panic: true,
	}, {
		label: "Merge",
		test:  func() { proto.Merge(nil, nil) },
		panic: true,
	}, {
		label: "Merge",
		test:  func() { proto.Merge(nil, nilMsg) },
		panic: true,
	}, {
		label: "Merge",
		test:  func() { proto.Merge(nilMsg, nil) },
		panic: true,
	}, {
		label: "Merge",
		test:  func() { proto.Merge(nilMsg, nilMsg) },
		panic: true,
	}, {
		label: "Clone",
		test:  func() { proto.Clone(nil) },
	}, {
		label: "Clone",
		test:  func() { proto.Clone(nilMsg) },
	}, {
		label: "Equal",
		test:  func() { proto.Equal(nil, nil) },
	}, {
		label: "Equal",
		test:  func() { proto.Equal(nil, nilMsg) },
	}, {
		label: "Equal",
		test:  func() { proto.Equal(nilMsg, nil) },
	}, {
		label: "Equal",
		test:  func() { proto.Equal(nilMsg, nilMsg) },
	}, {
		label: "Reset",
		test:  func() { proto.Reset(nil) },
		panic: true,
	}, {
		label: "Reset",
		test:  func() { proto.Reset(nilMsg) },
		panic: true,
	}, {
		label: "HasExtension",
		test:  func() { proto.HasExtension(nil, nil) },
		panic: true,
	}, {
		label: "HasExtension",
		test:  func() { proto.HasExtension(nil, extType) },
		panic: true,
	}, {
		label: "HasExtension",
		test:  func() { proto.HasExtension(nilMsg, nil) },
		panic: true,
	}, {
		label: "HasExtension",
		test:  func() { proto.HasExtension(nilMsg, extType) },
	}, {
		label: "GetExtension",
		test:  func() { proto.GetExtension(nil, nil) },
		panic: true,
	}, {
		label: "GetExtension",
		test:  func() { proto.GetExtension(nil, extType) },
		panic: true,
	}, {
		label: "GetExtension",
		test:  func() { proto.GetExtension(nilMsg, nil) },
		panic: true,
	}, {
		label: "GetExtension",
		test:  func() { proto.GetExtension(nilMsg, extType) },
	}, {
		label: "SetExtension",
		test:  func() { proto.SetExtension(nil, nil, true) },
		panic: true,
	}, {
		label: "SetExtension",
		test:  func() { proto.SetExtension(nil, extType, true) },
		panic: true,
	}, {
		label: "SetExtension",
		test:  func() { proto.SetExtension(nilMsg, nil, true) },
		panic: true,
	}, {
		label: "SetExtension",
		test:  func() { proto.SetExtension(nilMsg, extType, true) },
		panic: true,
	}, {
		label: "ClearExtension",
		test:  func() { proto.ClearExtension(nil, nil) },
		panic: true,
	}, {
		label: "ClearExtension",
		test:  func() { proto.ClearExtension(nil, extType) },
		panic: true,
	}, {
		label: "ClearExtension",
		test:  func() { proto.ClearExtension(nilMsg, nil) },
		panic: true,
	}, {
		label: "ClearExtension",
		test:  func() { proto.ClearExtension(nilMsg, extType) },
		panic: true,
	}}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			defer func() {
				switch gotPanic := recover() != nil; {
				case gotPanic && !tt.panic:
					t.Errorf("unexpected panic")
				case !gotPanic && tt.panic:
					t.Errorf("expected panic")
				}
			}()
			tt.test()
		})
	}
}
