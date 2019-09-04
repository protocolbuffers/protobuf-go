// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style.
// license that can be found in the LICENSE file.

package proto

import (
	"google.golang.org/protobuf/internal/encoding/messageset"
	"google.golang.org/protobuf/internal/encoding/wire"
	"google.golang.org/protobuf/internal/errors"
	"google.golang.org/protobuf/internal/pragma"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/runtime/protoiface"
)

// UnmarshalOptions configures the unmarshaler.
//
// Example usage:
//   err := UnmarshalOptions{DiscardUnknown: true}.Unmarshal(b, m)
type UnmarshalOptions struct {
	pragma.NoUnkeyedLiterals

	// AllowPartial accepts input for messages that will result in missing
	// required fields. If AllowPartial is false (the default), Unmarshal will
	// return an error if there are any missing required fields.
	AllowPartial bool

	// If DiscardUnknown is set, unknown fields are ignored.
	DiscardUnknown bool

	// Resolver is used for looking up types when unmarshaling extension fields.
	// If nil, this defaults to using protoregistry.GlobalTypes.
	Resolver interface {
		protoregistry.ExtensionTypeResolver
	}
}

var _ = protoiface.UnmarshalOptions(UnmarshalOptions{})

// Unmarshal parses the wire-format message in b and places the result in m.
func Unmarshal(b []byte, m Message) error {
	return UnmarshalOptions{}.Unmarshal(b, m)
}

// Unmarshal parses the wire-format message in b and places the result in m.
func (o UnmarshalOptions) Unmarshal(b []byte, m Message) error {
	if o.Resolver == nil {
		o.Resolver = protoregistry.GlobalTypes
	}

	// TODO: Reset m?
	err := o.unmarshalMessage(b, m.ProtoReflect())
	if err != nil {
		return err
	}
	if o.AllowPartial {
		return nil
	}
	return IsInitialized(m)
}

func (o UnmarshalOptions) unmarshalMessage(b []byte, m protoreflect.Message) error {
	if methods := protoMethods(m); methods != nil && methods.Unmarshal != nil &&
		!(o.DiscardUnknown && methods.Flags&protoiface.SupportUnmarshalDiscardUnknown == 0) {
		return methods.Unmarshal(b, m, protoiface.UnmarshalOptions(o))
	}
	return o.unmarshalMessageSlow(b, m)
}

func (o UnmarshalOptions) unmarshalMessageSlow(b []byte, m protoreflect.Message) error {
	md := m.Descriptor()
	if messageset.IsMessageSet(md) {
		return unmarshalMessageSet(b, m, o)
	}
	fields := md.Fields()
	for len(b) > 0 {
		// Parse the tag (field number and wire type).
		num, wtyp, tagLen := wire.ConsumeTag(b)
		if tagLen < 0 {
			return wire.ParseError(tagLen)
		}

		// Parse the field value.
		fd := fields.ByNumber(num)
		if fd == nil && md.ExtensionRanges().Has(num) {
			extType, err := o.Resolver.FindExtensionByNumber(md.FullName(), num)
			if err != nil && err != protoregistry.NotFound {
				return err
			}
			if extType != nil {
				fd = extType.TypeDescriptor()
			}
		}
		var err error
		var valLen int
		switch {
		case fd == nil:
			err = errUnknown
		case fd.IsList():
			valLen, err = o.unmarshalList(b[tagLen:], wtyp, m.Mutable(fd).List(), fd)
		case fd.IsMap():
			valLen, err = o.unmarshalMap(b[tagLen:], wtyp, m.Mutable(fd).Map(), fd)
		default:
			valLen, err = o.unmarshalSingular(b[tagLen:], wtyp, m, fd)
		}
		if err == errUnknown {
			valLen = wire.ConsumeFieldValue(num, wtyp, b[tagLen:])
			if valLen < 0 {
				return wire.ParseError(valLen)
			}
			m.SetUnknown(append(m.GetUnknown(), b[:tagLen+valLen]...))
		} else if err != nil {
			return err
		}
		b = b[tagLen+valLen:]
	}
	return nil
}

func (o UnmarshalOptions) unmarshalSingular(b []byte, wtyp wire.Type, m protoreflect.Message, fd protoreflect.FieldDescriptor) (n int, err error) {
	v, n, err := o.unmarshalScalar(b, wtyp, fd)
	if err != nil {
		return 0, err
	}
	switch fd.Kind() {
	case protoreflect.GroupKind, protoreflect.MessageKind:
		m2 := m.Mutable(fd).Message()
		if err := o.unmarshalMessage(v.Bytes(), m2); err != nil {
			return n, err
		}
	default:
		// Non-message scalars replace the previous value.
		m.Set(fd, v)
	}
	return n, nil
}

func (o UnmarshalOptions) unmarshalMap(b []byte, wtyp wire.Type, mapv protoreflect.Map, fd protoreflect.FieldDescriptor) (n int, err error) {
	if wtyp != wire.BytesType {
		return 0, errUnknown
	}
	b, n = wire.ConsumeBytes(b)
	if n < 0 {
		return 0, wire.ParseError(n)
	}
	var (
		keyField = fd.MapKey()
		valField = fd.MapValue()
		key      protoreflect.Value
		val      protoreflect.Value
		haveKey  bool
		haveVal  bool
	)
	switch valField.Kind() {
	case protoreflect.GroupKind, protoreflect.MessageKind:
		val = mapv.NewValue()
	}
	// Map entries are represented as a two-element message with fields
	// containing the key and value.
	for len(b) > 0 {
		num, wtyp, n := wire.ConsumeTag(b)
		if n < 0 {
			return 0, wire.ParseError(n)
		}
		b = b[n:]
		err = errUnknown
		switch num {
		case 1:
			key, n, err = o.unmarshalScalar(b, wtyp, keyField)
			if err != nil {
				break
			}
			haveKey = true
		case 2:
			var v protoreflect.Value
			v, n, err = o.unmarshalScalar(b, wtyp, valField)
			if err != nil {
				break
			}
			switch valField.Kind() {
			case protoreflect.GroupKind, protoreflect.MessageKind:
				if err := o.unmarshalMessage(v.Bytes(), val.Message()); err != nil {
					return 0, err
				}
			default:
				val = v
			}
			haveVal = true
		}
		if err == errUnknown {
			n = wire.ConsumeFieldValue(num, wtyp, b)
			if n < 0 {
				return 0, wire.ParseError(n)
			}
		} else if err != nil {
			return 0, err
		}
		b = b[n:]
	}
	// Every map entry should have entries for key and value, but this is not strictly required.
	if !haveKey {
		key = keyField.Default()
	}
	if !haveVal {
		switch valField.Kind() {
		case protoreflect.GroupKind, protoreflect.MessageKind:
		default:
			val = valField.Default()
		}
	}
	mapv.Set(key.MapKey(), val)
	return n, nil
}

// errUnknown is used internally to indicate fields which should be added
// to the unknown field set of a message. It is never returned from an exported
// function.
var errUnknown = errors.New("BUG: internal error (unknown)")
