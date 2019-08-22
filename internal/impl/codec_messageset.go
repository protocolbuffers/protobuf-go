// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style.
// license that can be found in the LICENSE file.

package impl

import (
	"sort"

	"google.golang.org/protobuf/internal/encoding/messageset"
	"google.golang.org/protobuf/internal/encoding/wire"
	"google.golang.org/protobuf/internal/errors"
	"google.golang.org/protobuf/internal/flags"
)

func makeMessageSetFieldCoder(mi *MessageInfo) pointerCoderFuncs {
	return pointerCoderFuncs{
		size: func(p pointer, tagsize int, opts marshalOptions) int {
			return sizeMessageSet(mi, p, tagsize, opts)
		},
		marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
			return marshalMessageSet(mi, b, p, wiretag, opts)
		},
		unmarshal: func(b []byte, p pointer, wtyp wire.Type, opts unmarshalOptions) (int, error) {
			return unmarshalMessageSet(mi, b, p, wtyp, opts)
		},
	}
}

func sizeMessageSet(mi *MessageInfo, p pointer, tagsize int, opts marshalOptions) (n int) {
	ext := *p.Extensions()
	if ext == nil {
		return 0
	}
	for _, x := range ext {
		xi := mi.extensionFieldInfo(x.GetType())
		if xi.funcs.size == nil {
			continue
		}
		num, _ := wire.DecodeTag(xi.wiretag)
		n += messageset.SizeField(num)
		n += xi.funcs.size(x.Value(), wire.SizeTag(messageset.FieldMessage), opts)
	}
	return n
}

func marshalMessageSet(mi *MessageInfo, b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
	if !flags.ProtoLegacy {
		return b, errors.New("no support for message_set_wire_format")
	}
	ext := *p.Extensions()
	if ext == nil {
		return b, nil
	}
	switch len(ext) {
	case 0:
		return b, nil
	case 1:
		// Fast-path for one extension: Don't bother sorting the keys.
		for _, x := range ext {
			var err error
			b, err = marshalMessageSetField(mi, b, x, opts)
			if err != nil {
				return b, err
			}
		}
		return b, nil
	default:
		// Sort the keys to provide a deterministic encoding.
		// Not sure this is required, but the old code does it.
		keys := make([]int, 0, len(ext))
		for k := range ext {
			keys = append(keys, int(k))
		}
		sort.Ints(keys)
		for _, k := range keys {
			var err error
			b, err = marshalMessageSetField(mi, b, ext[int32(k)], opts)
			if err != nil {
				return b, err
			}
		}
		return b, nil
	}
}

func marshalMessageSetField(mi *MessageInfo, b []byte, x ExtensionField, opts marshalOptions) ([]byte, error) {
	xi := mi.extensionFieldInfo(x.GetType())
	num, _ := wire.DecodeTag(xi.wiretag)
	b = messageset.AppendFieldStart(b, num)
	b, err := xi.funcs.marshal(b, x.Value(), wire.EncodeTag(messageset.FieldMessage, wire.BytesType), opts)
	if err != nil {
		return b, err
	}
	b = messageset.AppendFieldEnd(b)
	return b, nil
}

func unmarshalMessageSet(mi *MessageInfo, b []byte, p pointer, wtyp wire.Type, opts unmarshalOptions) (int, error) {
	if !flags.ProtoLegacy {
		return 0, errors.New("no support for message_set_wire_format")
	}
	if wtyp != wire.StartGroupType {
		return 0, errUnknown
	}
	ep := p.Extensions()
	if *ep == nil {
		*ep = make(map[int32]ExtensionField)
	}
	ext := *ep
	num, v, n, err := messageset.ConsumeFieldValue(b, true)
	if err != nil {
		return 0, err
	}
	if _, err := mi.unmarshalExtension(v, num, wire.BytesType, ext, opts); err != nil {
		return 0, err
	}
	return n, nil
}
