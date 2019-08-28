// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style.
// license that can be found in the LICENSE file.

package proto

import (
	"google.golang.org/protobuf/internal/encoding/messageset"
	"google.golang.org/protobuf/internal/encoding/wire"
	"google.golang.org/protobuf/internal/errors"
	"google.golang.org/protobuf/internal/flags"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func sizeMessageSet(m protoreflect.Message) (size int) {
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		size += messageset.SizeField(fd.Number())
		size += wire.SizeTag(messageset.FieldMessage)
		size += wire.SizeBytes(sizeMessage(v.Message()))
		return true
	})
	size += len(m.GetUnknown())
	return size
}

func marshalMessageSet(b []byte, m protoreflect.Message, o MarshalOptions) ([]byte, error) {
	if !flags.ProtoLegacy {
		return b, errors.New("no support for message_set_wire_format")
	}
	var err error
	o.rangeFields(m, func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		b, err = marshalMessageSetField(b, fd, v, o)
		return err == nil
	})
	if err != nil {
		return b, err
	}
	b = append(b, m.GetUnknown()...)
	return b, nil
}

func marshalMessageSetField(b []byte, fd protoreflect.FieldDescriptor, value protoreflect.Value, o MarshalOptions) ([]byte, error) {
	b = messageset.AppendFieldStart(b, fd.Number())
	b = wire.AppendTag(b, messageset.FieldMessage, wire.BytesType)
	b = wire.AppendVarint(b, uint64(o.Size(value.Message().Interface())))
	b, err := o.marshalMessage(b, value.Message())
	if err != nil {
		return b, err
	}
	b = messageset.AppendFieldEnd(b)
	return b, nil
}

func unmarshalMessageSet(b []byte, m protoreflect.Message, o UnmarshalOptions) error {
	if !flags.ProtoLegacy {
		return errors.New("no support for message_set_wire_format")
	}
	md := m.Descriptor()
	for len(b) > 0 {
		err := func() error {
			num, v, n, err := messageset.ConsumeField(b)
			if err != nil {
				// Not a message set field.
				//
				// Return errUnknown to try to add this to the unknown fields.
				// If the field is completely unparsable, we'll catch it
				// when trying to skip the field.
				return errUnknown
			}
			if !md.ExtensionRanges().Has(num) {
				return errUnknown
			}
			xt, err := o.Resolver.FindExtensionByNumber(md.FullName(), num)
			if err == protoregistry.NotFound {
				return errUnknown
			}
			if err != nil {
				return err
			}
			xd := xt.TypeDescriptor()
			if err := o.unmarshalMessage(v, m.Mutable(xd).Message()); err != nil {
				// Contents cannot be unmarshaled.
				return err
			}
			b = b[n:]
			return nil
		}()
		if err == errUnknown {
			_, _, n := wire.ConsumeField(b)
			if n < 0 {
				return wire.ParseError(n)
			}
			m.SetUnknown(append(m.GetUnknown(), b[:n]...))
			b = b[n:]
			continue
		}
		if err != nil {
			return err
		}
	}
	return nil
}
