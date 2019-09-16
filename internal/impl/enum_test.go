// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl_test

import (
	"reflect"
	"testing"

	pref "google.golang.org/protobuf/reflect/protoreflect"

	testpb "google.golang.org/protobuf/internal/testprotos/test"
)

func TestEnum(t *testing.T) {
	et := testpb.ForeignEnum_FOREIGN_FOO.Type()
	if got, want := et.GoType(), reflect.TypeOf(testpb.ForeignEnum_FOREIGN_FOO); got != want {
		t.Errorf("testpb.ForeignEnum_FOREIGN_FOO.Type().GoType() = %v, want %v", got, want)
	}
	if got, want := et.New(pref.EnumNumber(testpb.ForeignEnum_FOREIGN_FOO)), pref.Enum(testpb.ForeignEnum_FOREIGN_FOO); got != want {
		t.Errorf("testpb.ForeignEnum_FOREIGN_FOO.Type().New() = %[1]T(%[1]v), want %[2]T(%[2]v)", got, want)
	}
}
