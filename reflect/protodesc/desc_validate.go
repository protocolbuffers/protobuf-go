// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protodesc

import (
	"google.golang.org/protobuf/internal/errors"
	"google.golang.org/protobuf/internal/filedesc"

	"google.golang.org/protobuf/types/descriptorpb"
)

// TODO: Should we be responsible for validating other parts of the descriptor
// that we don't directly use?
//
// For example:
//	* That "json_name" is not set for an extension field. Maybe, maybe not.
//	* That "weak" is not set for an extension field (double check this).

// TODO: Store the input file descriptor to implement:
//	* protoreflect.Descriptor.DescriptorProto
//	* protoreflect.Descriptor.DescriptorOptions

// TODO: Should we return a File instead of protoreflect.FileDescriptor?
// This would allow users to mutate the File before converting it.
// However, this will complicate future work for validation since File may now
// diverge from the stored descriptor proto (see above TODO).

// TODO: This is important to prevent users from creating invalid types,
// but is not functionality needed now.
//
// Things to verify:
//	* Weak fields are only used if flags.Proto1Legacy is set
//	* Weak fields can only reference singular messages
//	(check if this the case for oneof fields)
//	* FieldDescriptor.MessageType cannot reference a remote type when the
//	remote name is a type within the local file.
//	* Default enum identifiers resolve to a declared number.
//	* Default values are only allowed in proto2.
//	* Default strings are valid UTF-8? Note that protoc does not check this.
//	* Field extensions are only valid in proto2, except when extending the
//	descriptor options.
//	* Remote enum and message types are actually found in imported files.
//	* Placeholder messages and types may only be for weak fields.
//	* Placeholder full names must be valid.
//	* The name of each descriptor must be valid.
//	* Options are of the correct Go type (e.g. *descriptorpb.MessageOptions).
// 	* len(ExtensionRangeOptions) <= len(ExtensionRanges)

func validateEnumDeclarations(es []filedesc.Enum, eds []*descriptorpb.EnumDescriptorProto) error {
	for i, ed := range eds {
		e := &es[i]
		for j, _ := range ed.GetValue() {
			v := &e.L2.Values.List[j]
			if e.L2.ReservedNames.Has(v.Name()) {
				return errors.New("enum %v contains value with reserved name %q", e.Name(), v.Name())
			}
			if e.L2.ReservedRanges.Has(v.Number()) {
				return errors.New("enum %v contains value with reserved number %d", e.Name(), v.Number())
			}
		}
	}
	return nil
}

func validateMessageDeclarations(ms []filedesc.Message, mds []*descriptorpb.DescriptorProto) error {
	for i, md := range mds {
		m := &ms[i]
		for j, fd := range md.GetField() {
			f := &m.L2.Fields.List[j]
			if m.L2.ReservedNames.Has(f.Name()) {
				return errors.New("%v contains field with reserved name %q", m.Name(), f.Name())
			}
			if m.L2.ReservedRanges.Has(f.Number()) {
				return errors.New("%v contains field with reserved number %d", m.Name(), f.Number())
			}
			if m.L2.ExtensionRanges.Has(f.Number()) {
				return errors.New("%v contains field with number %d in extension range", m.Name(), f.Number())
			}
			if fd.GetExtendee() != "" {
				return errors.New("message field may not have extendee")
			}
		}

		if err := validateEnumDeclarations(m.L1.Enums.List, md.GetEnumType()); err != nil {
			return err
		}
		if err := validateMessageDeclarations(m.L1.Messages.List, md.GetNestedType()); err != nil {
			return err
		}
		if err := validateExtensionDeclarations(m.L1.Extensions.List, md.GetExtension()); err != nil {
			return err
		}
	}
	return nil
}

func validateExtensionDeclarations(xs []filedesc.Extension, xds []*descriptorpb.FieldDescriptorProto) error {
	for i, xd := range xds {
		x := &xs[i]
		if xd.OneofIndex != nil {
			return errors.New("extension may not have oneof_index")
		}
		_ = x
	}
	return nil
}
