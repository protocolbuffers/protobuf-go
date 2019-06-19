// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style.
// license that can be found in the LICENSE file.

// Package messageset encodes and decodes the obsolete MessageSet wire format.
package messageset

import (
	"google.golang.org/protobuf/internal/encoding/wire"
	"google.golang.org/protobuf/internal/errors"
	pref "google.golang.org/protobuf/reflect/protoreflect"
)

// The MessageSet wire format is equivalent to a message defiend as follows,
// where each Item defines an extension field with a field number of 'type_id'
// and content of 'message'. MessageSet extensions must be non-repeated message
// fields.
//
//	message MessageSet {
//		repeated group Item = 1 {
//			required int32 type_id = 2;
//			required string message = 3;
//		}
//	}
const (
	FieldItem    = wire.Number(1)
	FieldTypeID  = wire.Number(2)
	FieldMessage = wire.Number(3)
)

// IsMessageSet returns whether the message uses the MessageSet wire format.
func IsMessageSet(md pref.MessageDescriptor) bool {
	xmd, ok := md.(interface{ IsMessageSet() bool })
	return ok && xmd.IsMessageSet()
}

// SizeField returns the size of a MessageSet item field containing an extension
// with the given field number, not counting the contents of the message subfield.
func SizeField(num wire.Number) int {
	return 2*wire.SizeTag(FieldItem) + wire.SizeTag(FieldTypeID) + wire.SizeVarint(uint64(num))
}

// ConsumeField parses a MessageSet item field and returns the contents of the
// type_id and message subfields and the total item length.
func ConsumeField(b []byte) (typeid wire.Number, message []byte, n int, err error) {
	num, wtyp, n := wire.ConsumeTag(b)
	if n < 0 {
		return 0, nil, 0, wire.ParseError(n)
	}
	if num != FieldItem || wtyp != wire.StartGroupType {
		return 0, nil, 0, errors.New("invalid MessageSet field number")
	}
	typeid, message, fieldLen, err := ConsumeFieldValue(b[n:], false)
	if err != nil {
		return 0, nil, 0, err
	}
	return typeid, message, n + fieldLen, nil
}

// ConsumeFieldValue parses b as a MessageSet item field value until and including
// the trailing end group marker. It assumes the start group tag has already been parsed.
// It returns the contents of the type_id and message subfields and the total
// item length.
//
// If wantLen is true, the returned message value includes the length prefix.
// This is ugly, but simplifies the fast-path decoder in internal/impl.
func ConsumeFieldValue(b []byte, wantLen bool) (typeid wire.Number, message []byte, n int, err error) {
	ilen := len(b)
	for {
		num, wtyp, n := wire.ConsumeTag(b)
		if n < 0 {
			return 0, nil, 0, wire.ParseError(n)
		}
		b = b[n:]
		switch {
		case num == FieldItem && wtyp == wire.EndGroupType:
			if wantLen && len(message) == 0 {
				// The message field was missing, which should never happen.
				// Be prepared for this case anyway.
				message = wire.AppendVarint(message, 0)
			}
			return typeid, message, ilen - len(b), nil
		case num == FieldTypeID && wtyp == wire.VarintType:
			v, n := wire.ConsumeVarint(b)
			if n < 0 {
				return 0, nil, 0, wire.ParseError(n)
			}
			b = b[n:]
			typeid = wire.Number(v)
		case num == FieldMessage && wtyp == wire.BytesType:
			m, n := wire.ConsumeBytes(b)
			if n < 0 {
				return 0, nil, 0, wire.ParseError(n)
			}
			if message == nil {
				if wantLen {
					message = b[:n]
				} else {
					message = m
				}
			} else {
				// This case should never happen in practice, but handle it for
				// correctness: The MessageSet item contains multiple message
				// fields, which need to be merged.
				//
				// In the case where we're returning the length, this becomes
				// quite inefficient since we need to strip the length off
				// the existing data and reconstruct it with the combined length.
				if wantLen {
					_, nn := wire.ConsumeVarint(message)
					m0 := message[nn:]
					message = message[:0]
					message = wire.AppendVarint(message, uint64(len(m0)+len(m)))
					message = append(message, m0...)
					message = append(message, m...)
				} else {
					message = append(message, m...)
				}
			}
			b = b[n:]
		}
	}
}

// AppendFieldStart appends the start of a MessageSet item field containing
// an extension with the given number. The caller must add the message
// subfield (including the tag).
func AppendFieldStart(b []byte, num wire.Number) []byte {
	b = wire.AppendTag(b, FieldItem, wire.StartGroupType)
	b = wire.AppendTag(b, FieldTypeID, wire.VarintType)
	b = wire.AppendVarint(b, uint64(num))
	return b
}

// AppendFieldEnd appends the trailing end group marker for a MessageSet item field.
func AppendFieldEnd(b []byte) []byte {
	return wire.AppendTag(b, FieldItem, wire.EndGroupType)
}
