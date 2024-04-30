// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protopath

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// PathValues returns the values along a path in message m.
func PathValues(p Path, m proto.Message) (Values, error) {
	v := Values{}
	// Follow the message descriptor through the path to avoid bad dereferences
	desc := protoreflect.Descriptor(m.ProtoReflect().Descriptor())
	cursor := protoreflect.ValueOf(m.ProtoReflect())
	for i, step := range p {
		v.Path = append(v.Path, step)
		switch step.Kind() {
		case RootStep:
			if i != 0 {
				return Values{}, fmt.Errorf("root step at index %d. Must be at index 0", i)
			}
			got := step.MessageDescriptor().FullName()
			want := desc.FullName()
			if got != want {
				return Values{}, fmt.Errorf("got root type %s, want %s", got, want)
			}
			v.Values = append(v.Values, cursor)
		case FieldAccessStep:
			if f, ok := desc.(protoreflect.FieldDescriptor); ok {
				desc = f.Message()
			}
			md, ok := desc.(protoreflect.MessageDescriptor)
			if !ok {
				return Values{}, fmt.Errorf(
					"%d: cursor has descriptor %T, want MessageDescriptor or FieldDescriptor that holds a "+
						"MessageDescriptor", i, desc)
			}
			fd := step.FieldDescriptor()
			desc = md.Fields().ByNumber(fd.Number())
			if desc == nil {
				return Values{}, fmt.Errorf("%d: cursor message missing field %v", i, fd.Number())
			}
			cursor = cursor.Message().Get(fd)
			v.Values = append(v.Values, cursor)
		case ListIndexStep:
			fd, ok := desc.(protoreflect.FieldDescriptor)
			if !ok {
				return Values{}, fmt.Errorf("%d: cursor has descriptor %T, want FieldDescriptor", i, desc)
			}
			if !fd.IsList() {
				return Values{}, fmt.Errorf("%d: cursor descriptor %T is not a list", i, fd)
			}
			desc = fd.Message()
			index := step.ListIndex()
			if index < 0 || index >= cursor.List().Len() {
				return Values{}, fmt.Errorf("%d: cursor list index %v out of range", i, index)
			}
			cursor = cursor.List().Get(index)
			v.Values = append(v.Values, cursor)
		case MapIndexStep:
			fd, ok := desc.(protoreflect.FieldDescriptor)
			if !ok {
				return Values{}, fmt.Errorf("%d: cursor has descriptor %T, want FieldDescriptor", i, desc)
			}
			if !fd.IsMap() {
				return Values{}, fmt.Errorf("%d: cursor descriptor %T is not a map", i, fd)
			}
			// If MapIndex is the wrong type for Map, we can't detect that and this will panic.
			cursor = cursor.Map().Get(step.MapIndex())
			if !cursor.IsValid() {
				return Values{}, fmt.Errorf("%d: cursor map missing key %v", i, step.MapIndex())
			}
			v.Values = append(v.Values, cursor)
		case AnyExpandStep:
			if desc != step.MessageDescriptor() {
				return Values{}, fmt.Errorf("%d: cursor any expansion at %T is not %T", i, desc, step.MessageDescriptor())
			}
			desc = step.MessageDescriptor()
			v.Values = append(v.Values, cursor)
		case UnknownAccessStep:
			return Values{}, fmt.Errorf("%d: unknown access step", i)
		}
	}
	return v, nil
}
