// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"math/bits"

	"google.golang.org/protobuf/internal/encoding/wire"
	"google.golang.org/protobuf/internal/errors"
	"google.golang.org/protobuf/internal/flags"
	"google.golang.org/protobuf/proto"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	preg "google.golang.org/protobuf/reflect/protoregistry"
	piface "google.golang.org/protobuf/runtime/protoiface"
)

// unmarshalOptions is a more efficient representation of UnmarshalOptions.
//
// We don't preserve the AllowPartial flag, because fast-path (un)marshal
// operations always allow partial messages.
type unmarshalOptions struct {
	flags unmarshalOptionFlags

	// Keep this field's type identical to (proto.UnmarshalOptions).Resolver
	// to avoid a type conversion on assignment.
	resolver interface {
		FindExtensionByName(field pref.FullName) (pref.ExtensionType, error)
		FindExtensionByNumber(message pref.FullName, field pref.FieldNumber) (pref.ExtensionType, error)
	}
}

type unmarshalOptionFlags uint8

const (
	unmarshalDiscardUnknown unmarshalOptionFlags = 1 << iota
)

func newUnmarshalOptions(opts piface.UnmarshalOptions) unmarshalOptions {
	o := unmarshalOptions{
		resolver: opts.Resolver,
	}
	if opts.DiscardUnknown {
		o.flags |= unmarshalDiscardUnknown
	}
	return o
}

func (o unmarshalOptions) Options() proto.UnmarshalOptions {
	return proto.UnmarshalOptions{
		Merge:          true,
		AllowPartial:   true,
		DiscardUnknown: o.DiscardUnknown(),
		Resolver:       o.Resolver(),
	}
}

func (o unmarshalOptions) DiscardUnknown() bool                 { return o.flags&unmarshalDiscardUnknown != 0 }
func (o unmarshalOptions) Resolver() preg.ExtensionTypeResolver { return o.resolver }

type unmarshalOutput struct {
	n           int // number of bytes consumed
	initialized bool
}

// unmarshal is protoreflect.Methods.Unmarshal.
func (mi *MessageInfo) unmarshal(m pref.Message, in piface.UnmarshalInput, opts piface.UnmarshalOptions) (piface.UnmarshalOutput, error) {
	var p pointer
	if ms, ok := m.(*messageState); ok {
		p = ms.pointer()
	} else {
		p = m.(*messageReflectWrapper).pointer()
	}
	out, err := mi.unmarshalPointer(in.Buf, p, 0, newUnmarshalOptions(opts))
	return piface.UnmarshalOutput{
		Initialized: out.initialized,
	}, err
}

// errUnknown is returned during unmarshaling to indicate a parse error that
// should result in a field being placed in the unknown fields section (for example,
// when the wire type doesn't match) as opposed to the entire unmarshal operation
// failing (for example, when a field extends past the available input).
//
// This is a sentinel error which should never be visible to the user.
var errUnknown = errors.New("unknown")

func (mi *MessageInfo) unmarshalPointer(b []byte, p pointer, groupTag wire.Number, opts unmarshalOptions) (out unmarshalOutput, err error) {
	mi.init()
	if flags.ProtoLegacy && mi.isMessageSet {
		return unmarshalMessageSet(mi, b, p, opts)
	}
	initialized := true
	var requiredMask uint64
	var exts *map[int32]ExtensionField
	start := len(b)
	for len(b) > 0 {
		// Parse the tag (field number and wire type).
		var tag uint64
		if b[0] < 0x80 {
			tag = uint64(b[0])
			b = b[1:]
		} else if len(b) >= 2 && b[1] < 128 {
			tag = uint64(b[0]&0x7f) + uint64(b[1])<<7
			b = b[2:]
		} else {
			var n int
			tag, n = wire.ConsumeVarint(b)
			if n < 0 {
				return out, wire.ParseError(n)
			}
			b = b[n:]
		}
		num := wire.Number(tag >> 3)
		wtyp := wire.Type(tag & 7)

		if num < wire.MinValidNumber || num > wire.MaxValidNumber {
			return out, errors.New("invalid field number")
		}

		if wtyp == wire.EndGroupType {
			if num != groupTag {
				return out, errors.New("mismatching end group marker")
			}
			groupTag = 0
			break
		}

		var f *coderFieldInfo
		if int(num) < len(mi.denseCoderFields) {
			f = mi.denseCoderFields[num]
		} else {
			f = mi.coderFields[num]
		}
		var n int
		err := errUnknown
		switch {
		case f != nil:
			if f.funcs.unmarshal == nil {
				break
			}
			var o unmarshalOutput
			o, err = f.funcs.unmarshal(b, p.Apply(f.offset), wtyp, opts)
			n = o.n
			if err != nil {
				break
			}
			requiredMask |= f.validation.requiredBit
			if f.funcs.isInit != nil && !o.initialized {
				initialized = false
			}
		default:
			// Possible extension.
			if exts == nil && mi.extensionOffset.IsValid() {
				exts = p.Apply(mi.extensionOffset).Extensions()
				if *exts == nil {
					*exts = make(map[int32]ExtensionField)
				}
			}
			if exts == nil {
				break
			}
			var o unmarshalOutput
			o, err = mi.unmarshalExtension(b, num, wtyp, *exts, opts)
			n = o.n
			if !o.initialized {
				initialized = false
			}
		}
		if err != nil {
			if err != errUnknown {
				return out, err
			}
			n = wire.ConsumeFieldValue(num, wtyp, b)
			if n < 0 {
				return out, wire.ParseError(n)
			}
			if mi.unknownOffset.IsValid() {
				u := p.Apply(mi.unknownOffset).Bytes()
				*u = wire.AppendTag(*u, num, wtyp)
				*u = append(*u, b[:n]...)
			}
		}
		b = b[n:]
	}
	if groupTag != 0 {
		return out, errors.New("missing end group marker")
	}
	if mi.numRequiredFields > 0 && bits.OnesCount64(requiredMask) != int(mi.numRequiredFields) {
		initialized = false
	}
	if initialized {
		out.initialized = true
	}
	out.n = start - len(b)
	return out, nil
}

func (mi *MessageInfo) unmarshalExtension(b []byte, num wire.Number, wtyp wire.Type, exts map[int32]ExtensionField, opts unmarshalOptions) (out unmarshalOutput, err error) {
	x := exts[int32(num)]
	xt := x.Type()
	if xt == nil {
		var err error
		xt, err = opts.Resolver().FindExtensionByNumber(mi.Desc.FullName(), num)
		if err != nil {
			if err == preg.NotFound {
				return out, errUnknown
			}
			return out, err
		}
	}
	xi := getExtensionFieldInfo(xt)
	if xi.funcs.unmarshal == nil {
		return out, errUnknown
	}
	ival := x.Value()
	if !ival.IsValid() && xi.unmarshalNeedsValue {
		// Create a new message, list, or map value to fill in.
		// For enums, create a prototype value to let the unmarshal func know the
		// concrete type.
		ival = xt.New()
	}
	v, out, err := xi.funcs.unmarshal(b, ival, num, wtyp, opts)
	if err != nil {
		return out, err
	}
	x.Set(xt, v)
	exts[int32(num)] = x
	return out, nil
}
