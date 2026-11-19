// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package messageset_test

import (
	"bytes"
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/internal/encoding/messageset"
)

func TestConsumeFieldValueRepeatedMessagesAllocations(t *testing.T) {
	const (
		typeID    = protowire.Number(123)
		numChunks = 20000
	)
	var body []byte
	body = protowire.AppendTag(body, messageset.FieldTypeID, protowire.VarintType)
	body = protowire.AppendVarint(body, uint64(typeID))
	wantPayload := make([]byte, numChunks)
	for i := range numChunks {
		b := byte(i)
		wantPayload[i] = b
		body = protowire.AppendTag(body, messageset.FieldMessage, protowire.BytesType)
		body = protowire.AppendBytes(body, []byte{b})
	}
	body = messageset.AppendFieldEnd(body)

	var gotTypeID protowire.Number
	var gotMsg []byte
	var gotN int
	allocs := testing.AllocsPerRun(5, func() {
		var err error
		gotTypeID, gotMsg, gotN, err = messageset.ConsumeFieldValue(body, true)
		if err != nil {
			t.Fatal(err)
		}
	})

	if gotTypeID != typeID {
		t.Errorf("ConsumeFieldValue(wantLen=true) typeid = %v, want %v", gotTypeID, typeID)
	}
	if gotN != len(body) {
		t.Errorf("ConsumeFieldValue(wantLen=true) n = %d, want %d", gotN, len(body))
	}
	wantMsg := protowire.AppendBytes(nil, wantPayload)
	if !bytes.Equal(gotMsg, wantMsg) {
		t.Errorf("ConsumeFieldValue(wantLen=true) returned unexpected merged message (got len %d, want len %d)", len(gotMsg), len(wantMsg))
	}
	if allocs > 64 {
		t.Errorf("ConsumeFieldValue(wantLen=true) allocs for %d chunks = %v, want <= 64", numChunks, allocs)
	}
}
