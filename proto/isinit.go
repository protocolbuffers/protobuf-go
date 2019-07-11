// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style.
// license that can be found in the LICENSE file.

package proto

import (
	"google.golang.org/protobuf/internal/errors"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// IsInitialized returns an error if any required fields in m are not set.
func IsInitialized(m Message) error {
	return isInitialized(m.ProtoReflect())
}

// IsInitialized returns an error if any required fields in m are not set.
func isInitialized(m protoreflect.Message) error {
	if methods := protoMethods(m); methods != nil && methods.IsInitialized != nil {
		return methods.IsInitialized(m)
	}
	return isInitializedSlow(m)
}

func isInitializedSlow(m protoreflect.Message) error {
	md := m.Descriptor()
	fds := md.Fields()
	for i, nums := 0, md.RequiredNumbers(); i < nums.Len(); i++ {
		fd := fds.ByNumber(nums.Get(i))
		if !m.Has(fd) {
			return errors.RequiredNotSet(string(fd.FullName()))
		}
	}
	var err error
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch {
		case fd.IsList():
			if fd.Message() == nil {
				return true
			}
			for i, list := 0, v.List(); i < list.Len() && err == nil; i++ {
				err = isInitialized(list.Get(i).Message())
			}
		case fd.IsMap():
			if fd.MapValue().Message() == nil {
				return true
			}
			v.Map().Range(func(key protoreflect.MapKey, v protoreflect.Value) bool {
				err = isInitialized(v.Message())
				return err == nil
			})
		default:
			if fd.Message() == nil {
				return true
			}
			err = isInitialized(v.Message())
		}
		return err == nil
	})
	return err
}
