// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style.
// license that can be found in the LICENSE file.

package proto_test

import (
	"fmt"
	"testing"

	"google.golang.org/protobuf/internal/impl"
	"google.golang.org/protobuf/reflect/protoregistry"
	piface "google.golang.org/protobuf/runtime/protoiface"
)

// TestValidate tests the internal message validator.
//
// Despite being more properly associated with the internal/impl package,
// it is located here to take advantage of the test wire encoder/decoder inputs.

func TestValidateValid(t *testing.T) {
	for _, test := range testValidMessages {
		for _, m := range test.decodeTo {
			t.Run(fmt.Sprintf("%s (%T)", test.desc, m), func(t *testing.T) {
				mt := m.ProtoReflect().Type()
				want := impl.ValidationValidInitialized
				if test.validationStatus != 0 {
					want = test.validationStatus
				} else if test.partial {
					want = impl.ValidationValidMaybeUninitalized
				}
				var opts piface.UnmarshalOptions
				opts.Resolver = protoregistry.GlobalTypes
				if got, want := impl.Validate(test.wire, mt, opts), want; got != want {
					t.Errorf("Validate(%x) = %v, want %v", test.wire, got, want)
				}
			})
		}
	}
}

func TestValidateInvalid(t *testing.T) {
	for _, test := range testInvalidMessages {
		for _, m := range test.decodeTo {
			t.Run(fmt.Sprintf("%s (%T)", test.desc, m), func(t *testing.T) {
				mt := m.ProtoReflect().Type()
				var opts piface.UnmarshalOptions
				opts.Resolver = protoregistry.GlobalTypes
				if got, want := impl.Validate(test.wire, mt, opts), impl.ValidationInvalid; got != want {
					t.Errorf("Validate(%x) = %v, want %v", test.wire, got, want)
				}
			})
		}
	}
}
