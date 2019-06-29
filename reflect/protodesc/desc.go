// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package protodesc provides for converting descriptorpb.FileDescriptorProto
// to/from the reflective protoreflect.FileDescriptor.
package protodesc

import (
	"google.golang.org/protobuf/internal/errors"
	"google.golang.org/protobuf/internal/filedesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"google.golang.org/protobuf/types/descriptorpb"
)

// Resolver is the resolver used by NewFile to resolve dependencies.
// It is implemented by protoregistry.Files.
type Resolver interface {
	FindFileByPath(string) (protoreflect.FileDescriptor, error)
	FindDescriptorByName(protoreflect.FullName) (protoreflect.Descriptor, error)
}

// NewFile creates a new protoreflect.FileDescriptor from the provided
// file descriptor message. The file must represent a valid proto file according
// to protobuf semantics.
//
// Any import files, enum types, or message types referenced in the file are
// resolved using the provided registry. When looking up an import file path,
// the path must be unique. The newly created file descriptor is not registered
// back into the provided file registry.
//
// The caller must relinquish full ownership of the input fd and must not
// access or mutate any fields.
func NewFile(fd *descriptorpb.FileDescriptorProto, r Resolver) (protoreflect.FileDescriptor, error) {
	if r == nil {
		r = (*protoregistry.Files)(nil) // empty resolver
	}
	f := &filedesc.File{L2: &filedesc.FileL2{}}
	switch fd.GetSyntax() {
	case "proto2", "":
		f.L1.Syntax = protoreflect.Proto2
	case "proto3":
		f.L1.Syntax = protoreflect.Proto3
	default:
		return nil, errors.New("invalid syntax: %v", fd.GetSyntax())
	}
	f.L1.Path = fd.GetName()
	f.L1.Package = protoreflect.FullName(fd.GetPackage())
	if opts := fd.GetOptions(); opts != nil {
		f.L2.Options = func() protoreflect.ProtoMessage { return opts }
	}

	f.L2.Imports = make(filedesc.FileImports, len(fd.GetDependency()))
	for _, i := range fd.GetPublicDependency() {
		if int(i) >= len(f.L2.Imports) || f.L2.Imports[i].IsPublic {
			return nil, errors.New("invalid or duplicate public import index: %d", i)
		}
		f.L2.Imports[i].IsPublic = true
	}
	for _, i := range fd.GetWeakDependency() {
		if int(i) >= len(f.L2.Imports) || f.L2.Imports[i].IsWeak {
			return nil, errors.New("invalid or duplicate weak import index: %d", i)
		}
		f.L2.Imports[i].IsWeak = true
	}
	imps := importSet{f.Path(): true}
	for i, path := range fd.GetDependency() {
		imp := &f.L2.Imports[i]
		f, err := r.FindFileByPath(path)
		if err != nil {
			// TODO: This should be an error.
			f = filedesc.PlaceholderFile(path)
		}
		imp.FileDescriptor = f

		imps[imp.Path()] = true
		imps.importPublic(imp.Imports())
	}

	// Step 1: Allocate and derive the names for all declarations.
	// This copies all fields from the descriptor proto except:
	//	google.protobuf.FieldDescriptorProto.type_name
	//	google.protobuf.FieldDescriptorProto.default_value
	//	google.protobuf.FieldDescriptorProto.oneof_index
	//	google.protobuf.FieldDescriptorProto.extendee
	//	google.protobuf.MethodDescriptorProto.input
	//	google.protobuf.MethodDescriptorProto.output
	var err error
	r1 := make(descsByName)
	if f.L1.Enums.List, err = r1.initEnumDeclarations(fd.GetEnumType(), f); err != nil {
		return nil, err
	}
	if f.L1.Messages.List, err = r1.initMessagesDeclarations(fd.GetMessageType(), f); err != nil {
		return nil, err
	}
	if f.L1.Extensions.List, err = r1.initExtensionDeclarations(fd.GetExtension(), f); err != nil {
		return nil, err
	}
	if f.L1.Services.List, err = r1.initServiceDeclarations(fd.GetService(), f); err != nil {
		return nil, err
	}

	// Step 2: Resolve every dependency reference not handled by step 1.
	r2 := &resolver{local: r1, remote: r, imports: imps}
	if err := r2.resolveMessageDependencies(f.L1.Messages.List, fd.GetMessageType()); err != nil {
		return nil, err
	}
	if err := r2.resolveExtensionDependencies(f.L1.Extensions.List, fd.GetExtension()); err != nil {
		return nil, err
	}
	if err := r2.resolveServiceDependencies(f.L1.Services.List, fd.GetService()); err != nil {
		return nil, err
	}

	// Step 3: Validate every enum, message, and extension declaration.
	if err := validateEnumDeclarations(f.L1.Enums.List, fd.GetEnumType()); err != nil {
		return nil, err
	}
	if err := validateMessageDeclarations(f.L1.Messages.List, fd.GetMessageType()); err != nil {
		return nil, err
	}
	if err := validateExtensionDeclarations(f.L1.Extensions.List, fd.GetExtension()); err != nil {
		return nil, err
	}

	return f, nil
}

type importSet map[string]bool

func (is importSet) importPublic(imps protoreflect.FileImports) {
	for i := 0; i < imps.Len(); i++ {
		if imp := imps.Get(i); imp.IsPublic {
			is[imp.Path()] = true
			is.importPublic(imp.Imports())
		}
	}
}

// check returns an error if d does not belong to a currently imported file.
func (is importSet) check(d protoreflect.Descriptor) error {
	if !is[d.ParentFile().Path()] {
		return errors.New("reference to type %v without import of %v", d.FullName(), d.ParentFile().Path())
	}
	return nil
}
