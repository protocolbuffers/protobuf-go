// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protodesc

import (
	"strings"

	"google.golang.org/protobuf/internal/encoding/defval"
	"google.golang.org/protobuf/internal/errors"
	"google.golang.org/protobuf/internal/filedesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"google.golang.org/protobuf/types/descriptorpb"
)

type resolver struct {
	local   descsByName
	remote  Resolver
	imports importSet
}

func (r *resolver) resolveMessageDependencies(ms []filedesc.Message, mds []*descriptorpb.DescriptorProto) (err error) {
	for i, md := range mds {
		m := &ms[i]
		for j, fd := range md.GetField() {
			f := &m.L2.Fields.List[j]
			if f.L1.Cardinality == protoreflect.Required {
				m.L2.RequiredNumbers.List = append(m.L2.RequiredNumbers.List, f.L1.Number)
			}
			if fd.OneofIndex != nil {
				k := int(fd.GetOneofIndex())
				if k >= len(md.GetOneofDecl()) {
					return errors.New("invalid oneof index: %d", k)
				}
				o := &m.L2.Oneofs.List[k]
				f.L1.ContainingOneof = o
				o.L1.Fields.List = append(o.L1.Fields.List, f)
			}
			switch f.L1.Kind {
			case protoreflect.EnumKind:
				ed, err := findEnumDescriptor(fd.GetTypeName(), f.L1.IsWeak, r)
				if err != nil {
					return err
				}
				f.L1.Enum = ed
			case protoreflect.MessageKind, protoreflect.GroupKind:
				md, err := findMessageDescriptor(fd.GetTypeName(), f.L1.IsWeak, r)
				if err != nil {
					return err
				}
				f.L1.Message = md
			default:
				if fd.GetTypeName() != "" {
					return errors.New("field of kind %v has type_name set", f.L1.Kind)
				}
			}
			if fd.DefaultValue != nil {
				// Handle default value after resolving the enum since the
				// list of enum values is needed to resolve enums by name.
				var evs protoreflect.EnumValueDescriptors
				if f.L1.Kind == protoreflect.EnumKind {
					evs = f.L1.Enum.Values()
				}
				v, ev, err := defval.Unmarshal(fd.GetDefaultValue(), f.L1.Kind, evs, defval.Descriptor)
				if err != nil {
					return err
				}
				f.L1.Default = filedesc.DefaultValue(v, ev)
			}
		}

		if err := r.resolveMessageDependencies(m.L1.Messages.List, md.GetNestedType()); err != nil {
			return err
		}
		if err := r.resolveExtensionDependencies(m.L1.Extensions.List, md.GetExtension()); err != nil {
			return err
		}
	}
	return nil
}

func (r *resolver) resolveExtensionDependencies(xs []filedesc.Extension, xds []*descriptorpb.FieldDescriptorProto) error {
	for i, xd := range xds {
		x := &xs[i]
		md, err := findMessageDescriptor(xd.GetExtendee(), false, r)
		if err != nil {
			return err
		}
		x.L1.Extendee = md
		switch x.L1.Kind {
		case protoreflect.EnumKind:
			ed, err := findEnumDescriptor(xd.GetTypeName(), false, r)
			if err != nil {
				return err
			}
			x.L2.Enum = ed
		case protoreflect.MessageKind, protoreflect.GroupKind:
			md, err := findMessageDescriptor(xd.GetTypeName(), false, r)
			if err != nil {
				return err
			}
			x.L2.Message = md
		default:
			if xd.GetTypeName() != "" {
				return errors.New("field of kind %v has type_name set", x.L1.Kind)
			}
		}
		if xd.DefaultValue != nil {
			// Handle default value after resolving the enum since the
			// list of enum values is needed to resolve enums by name.
			var evs protoreflect.EnumValueDescriptors
			if x.L1.Kind == protoreflect.EnumKind {
				evs = x.L2.Enum.Values()
			}
			v, ev, err := defval.Unmarshal(xd.GetDefaultValue(), x.L1.Kind, evs, defval.Descriptor)
			if err != nil {
				return err
			}
			x.L2.Default = filedesc.DefaultValue(v, ev)
		}
	}
	return nil
}

func (r *resolver) resolveServiceDependencies(ss []filedesc.Service, sds []*descriptorpb.ServiceDescriptorProto) (err error) {
	for i, sd := range sds {
		s := &ss[i]
		for j, md := range sd.GetMethod() {
			m := &s.L2.Methods.List[j]
			m.L1.Input, err = findMessageDescriptor(md.GetInputType(), false, r)
			if err != nil {
				return err
			}
			m.L1.Output, err = findMessageDescriptor(md.GetOutputType(), false, r)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// TODO: Should we allow relative names? The protoc compiler has emitted
// absolute names for some time now. Requiring absolute names as an input
// simplifies our implementation as we won't need to implement C++'s namespace
// scoping rules.

func (r resolver) FindFileByPath(s string) (protoreflect.FileDescriptor, error) {
	return r.remote.FindFileByPath(s)
}

func (r resolver) FindDescriptorByName(s protoreflect.FullName) (protoreflect.Descriptor, error) {
	if d, ok := r.local[s]; ok {
		return d, nil
	}
	return r.remote.FindDescriptorByName(s)
}

func findEnumDescriptor(s string, isWeak bool, r *resolver) (protoreflect.EnumDescriptor, error) {
	d, err := findDescriptor(s, isWeak, r)
	if err != nil {
		return nil, err
	}
	if ed, ok := d.(protoreflect.EnumDescriptor); ok {
		if err == protoregistry.NotFound {
			if isWeak {
				return filedesc.PlaceholderEnum(protoreflect.FullName(s[1:])), nil
			}
			// TODO: This should be an error.
			return filedesc.PlaceholderEnum(protoreflect.FullName(s[1:])), nil
			// return nil, errors.New("could not resolve enum: %v", name)
		}
		return ed, nil
	}
	return nil, errors.New("invalid descriptor type")
}

func findMessageDescriptor(s string, isWeak bool, r *resolver) (protoreflect.MessageDescriptor, error) {
	d, err := findDescriptor(s, isWeak, r)
	if err != nil {
		if err == protoregistry.NotFound {
			if isWeak {
				return filedesc.PlaceholderMessage(protoreflect.FullName(s[1:])), nil
			}
			// TODO: This should be an error.
			return filedesc.PlaceholderMessage(protoreflect.FullName(s[1:])), nil
			// return nil, errors.New("could not resolve enum: %v", name)
		}
		return nil, err
	}
	if md, ok := d.(protoreflect.MessageDescriptor); ok {
		return md, nil
	}
	return nil, errors.New("invalid descriptor type")
}

func findDescriptor(s string, isWeak bool, r *resolver) (protoreflect.Descriptor, error) {
	if !strings.HasPrefix(s, ".") {
		return nil, errors.New("identifier name must be fully qualified with a leading dot: %v", s)
	}
	name := protoreflect.FullName(strings.TrimPrefix(s, "."))
	d, err := r.FindDescriptorByName(name)
	if err != nil {
		return nil, err
	}
	if err := r.imports.check(d); err != nil {
		return nil, err
	}
	return d, nil
}
