// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"google.golang.org/protobuf/internal/encoding/wire"
	"google.golang.org/protobuf/internal/errors"
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
	flags    unmarshalOptionFlags
	resolver preg.ExtensionTypeResolver
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
		AllowPartial:   true,
		DiscardUnknown: o.DiscardUnknown(),
		Resolver:       o.Resolver(),
	}
}

func (o unmarshalOptions) DiscardUnknown() bool                 { return o.flags&unmarshalDiscardUnknown != 0 }
func (o unmarshalOptions) Resolver() preg.ExtensionTypeResolver { return o.resolver }

// unmarshal is protoreflect.Methods.Unmarshal.
func (mi *MessageInfo) unmarshal(b []byte, m pref.Message, opts piface.UnmarshalOptions) error {
	var p pointer
	if ms, ok := m.(*messageState); ok {
		p = ms.pointer()
	} else {
		p = m.(*messageReflectWrapper).pointer()
	}
	_, err := mi.unmarshalPointer(b, p, 0, newUnmarshalOptions(opts))
	return err
}

// errUnknown is returned during unmarshaling to indicate a parse error that
// should result in a field being placed in the unknown fields section (for example,
// when the wire type doesn't match) as opposed to the entire unmarshal operation
// failing (for example, when a field extends past the available input).
//
// This is a sentinel error which should never be visible to the user.
var errUnknown = errors.New("unknown")

func (mi *MessageInfo) unmarshalPointer(b []byte, p pointer, groupTag wire.Number, opts unmarshalOptions) (int, error) {
	mi.init()
	var exts *map[int32]ExtensionField
	start := len(b)
	for len(b) > 0 {
		// Parse the tag (field number and wire type).
		// TODO: inline 1 and 2 byte variants?
		num, wtyp, n := wire.ConsumeTag(b)
		if n < 0 {
			return 0, wire.ParseError(n)
		}
		b = b[n:]

		var f *coderFieldInfo
		if int(num) < len(mi.denseCoderFields) {
			f = mi.denseCoderFields[num]
		} else {
			f = mi.coderFields[num]
		}
		err := errUnknown
		switch {
		case f != nil:
			if f.funcs.unmarshal == nil {
				break
			}
			n, err = f.funcs.unmarshal(b, p.Apply(f.offset), wtyp, opts)
		case num == groupTag && wtyp == wire.EndGroupType:
			// End of group.
			return start - len(b), nil
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
			n, err = mi.unmarshalExtension(b, num, wtyp, *exts, opts)
		}
		if err != nil {
			if err != errUnknown {
				return 0, err
			}
			n = wire.ConsumeFieldValue(num, wtyp, b)
			if n < 0 {
				return 0, wire.ParseError(n)
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
		return 0, errors.New("missing end group marker")
	}
	return start, nil
}

func (mi *MessageInfo) unmarshalExtension(b []byte, num wire.Number, wtyp wire.Type, exts map[int32]ExtensionField, opts unmarshalOptions) (n int, err error) {
	x := exts[int32(num)]
	xt := x.GetType()
	if xt == nil {
		var err error
		xt, err = opts.Resolver().FindExtensionByNumber(mi.Desc.FullName(), num)
		if err != nil {
			if err == preg.NotFound {
				return 0, errUnknown
			}
			return 0, err
		}
	}
	xi := mi.extensionFieldInfo(xt)
	if xi.funcs.unmarshal == nil {
		return 0, errUnknown
	}
	ival := x.Value()
	if !ival.IsValid() && xi.unmarshalNeedsValue {
		// Create a new message, list, or map value to fill in.
		// For enums, create a prototype value to let the unmarshal func know the
		// concrete type.
		ival = xt.New()
	}
	v, n, err := xi.funcs.unmarshal(b, ival, num, wtyp, opts)
	if err != nil {
		return 0, err
	}
	x.Set(xt, v)
	exts[int32(num)] = x
	return n, nil
}
