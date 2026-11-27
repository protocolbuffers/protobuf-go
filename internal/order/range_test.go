// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package order_test

import (
	"testing"

	"google.golang.org/protobuf/internal/order"
	"google.golang.org/protobuf/internal/test/race"
	"google.golang.org/protobuf/reflect/protoreflect"

	testpb "google.golang.org/protobuf/internal/testprotos/test"
)

// fieldRanger visits fds in reverse, so that RangeFields has to sort them.
type fieldRanger struct {
	fds []protoreflect.FieldDescriptor
}

func (r *fieldRanger) Range(f order.VisitField) {
	for i := len(r.fds) - 1; i >= 0; i-- {
		if !f(r.fds[i], protoreflect.ValueOfInt32(int32(i))) {
			return
		}
	}
}

// entryRanger visits n int64 keys in reverse, so that RangeEntries has to
// sort them.
type entryRanger struct {
	n int64
}

func (r *entryRanger) Range(f order.VisitEntry) {
	for k := r.n - 1; k >= 0; k-- {
		if !f(protoreflect.ValueOfInt64(k).MapKey(), protoreflect.ValueOfInt64(k)) {
			return
		}
	}
}

func TestRangeAllocations(t *testing.T) {
	if race.Enabled {
		t.Skip("the race detector changes allocation counts")
	}

	fields := (&testpb.TestAllTypes{}).ProtoReflect().Descriptor().Fields()
	fr := &fieldRanger{}
	for i := range fields.Len() {
		fr.fds = append(fr.fds, fields.Get(i))
	}
	var visited int
	visitField := func(protoreflect.FieldDescriptor, protoreflect.Value) bool {
		visited++
		return true
	}
	if allocs := testing.AllocsPerRun(100, func() {
		order.RangeFields(fr, order.IndexNameFieldOrder, visitField)
	}); allocs > 1 {
		t.Errorf("RangeFields: got %v allocs/op, want at most 1", allocs)
	}
	if visited == 0 {
		t.Error("RangeFields should visit the fields")
	}

	er := &entryRanger{n: 100}
	visited = 0
	visitEntry := func(protoreflect.MapKey, protoreflect.Value) bool {
		visited++
		return true
	}
	if allocs := testing.AllocsPerRun(100, func() {
		order.RangeEntries(er, order.GenericKeyOrder, visitEntry)
	}); allocs > 1 {
		t.Errorf("RangeEntries: got %v allocs/op, want at most 1", allocs)
	}
	if visited == 0 {
		t.Error("RangeEntries should visit the entries")
	}
}
