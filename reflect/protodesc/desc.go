// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package protodesc provides for converting descriptorpb.FileDescriptorProto
// to/from the reflective protoreflect.FileDescriptor.
package protodesc

import (
	"google.golang.org/protobuf/internal/errors"
	"google.golang.org/protobuf/internal/filedesc"
	"google.golang.org/protobuf/internal/pragma"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"google.golang.org/protobuf/types/descriptorpb"
)

// Resolver is the resolver used by NewFile to resolve dependencies.
// The enums and messages provided must belong to some parent file,
// which is also registered.
//
// It is implemented by protoregistry.Files.
type Resolver interface {
	FindFileByPath(string) (protoreflect.FileDescriptor, error)
	FindDescriptorByName(protoreflect.FullName) (protoreflect.Descriptor, error)
}

// option is an optional argument that may be provided to NewFile.
type option struct {
	pragma.DoNotCompare
	allowUnresolvable bool
}

// allowUnresolvable configures NewFile to permissively allow unresolvable
// file, enum, or message dependencies. Unresolved dependencies are replaced by
// placeholder equivalents.
//
// The following dependencies may be left unresolved:
//	• Resolving an imported file.
//	• Resolving the type for a message field or extension field.
//	If the kind of the field is unknown, then a placeholder is used for both
//	protoreflect.FieldDescriptor.Enum and protoreflect.FieldDescriptor.Message.
//	• Resolving the enum value set as the default for an optional enum field.
//	If unresolvable, the protoreflect.FieldDescriptor.Default is set to the
//	first enum value in the associated enum (or zero if the also enum dependency
//	is also unresolvable). The protoreflect.FieldDescriptor.DefaultEnumValue
//	is set as a placeholder.
//	• Resolving the extended message type for an extension field.
//	• Resolving the input or output message type for a service method.
//
// If the unresolved dependency uses a relative name, then the placeholder will
// contain an invalid FullName with a "*." prefix, indicating that the starting
// prefix of the full name is unknown.
func allowUnresolvable() option {
	return option{allowUnresolvable: true}
}

// NewFile creates a new protoreflect.FileDescriptor from the provided
// file descriptor message. The file must represent a valid proto file according
// to protobuf semantics. The returned descriptor is a deep copy of the input.
//
// Any imported files, enum types, or message types referenced in the file are
// resolved using the provided registry. When looking up an import file path,
// the path must be unique. The newly created file descriptor is not registered
// back into the provided file registry.
func NewFile(fd *descriptorpb.FileDescriptorProto, r Resolver) (protoreflect.FileDescriptor, error) {
	// TODO: remove setting allowUnresolvable once naughty users are migrated.
	return newFile(fd, r, allowUnresolvable())
}
func newFile(fd *descriptorpb.FileDescriptorProto, r Resolver, opts ...option) (protoreflect.FileDescriptor, error) {
	// Process the options.
	var allowUnresolvable bool
	for _, o := range opts {
		allowUnresolvable = allowUnresolvable || o.allowUnresolvable
	}
	if r == nil {
		r = (*protoregistry.Files)(nil) // empty resolver
	}

	// Handle the file descriptor content.
	f := &filedesc.File{L2: &filedesc.FileL2{}}
	switch fd.GetSyntax() {
	case "proto2", "":
		f.L1.Syntax = protoreflect.Proto2
	case "proto3":
		f.L1.Syntax = protoreflect.Proto3
	default:
		return nil, errors.New("invalid syntax: %q", fd.GetSyntax())
	}
	f.L1.Path = fd.GetName()
	if f.L1.Path == "" {
		return nil, errors.New("file path must be populated")
	}
	f.L1.Package = protoreflect.FullName(fd.GetPackage())
	if !f.L1.Package.IsValid() && f.L1.Package != "" {
		return nil, errors.New("invalid package: %q", f.L1.Package)
	}
	if opts := fd.GetOptions(); opts != nil {
		opts = clone(opts).(*descriptorpb.FileOptions)
		f.L2.Options = func() protoreflect.ProtoMessage { return opts }
	}

	f.L2.Imports = make(filedesc.FileImports, len(fd.GetDependency()))
	for _, i := range fd.GetPublicDependency() {
		if !(0 <= i && int(i) < len(f.L2.Imports)) || f.L2.Imports[i].IsPublic {
			return nil, errors.New("invalid or duplicate public import index: %d", i)
		}
		f.L2.Imports[i].IsPublic = true
	}
	for _, i := range fd.GetWeakDependency() {
		if !(0 <= i && int(i) < len(f.L2.Imports)) || f.L2.Imports[i].IsWeak {
			return nil, errors.New("invalid or duplicate weak import index: %d", i)
		}
		f.L2.Imports[i].IsWeak = true
	}
	imps := importSet{f.Path(): true}
	for i, path := range fd.GetDependency() {
		imp := &f.L2.Imports[i]
		f, err := r.FindFileByPath(path)
		if err == protoregistry.NotFound && (allowUnresolvable || imp.IsWeak) {
			f = filedesc.PlaceholderFile(path)
		} else if err != nil {
			return nil, errors.New("could not resolve import %q: %v", path, err)
		}
		imp.FileDescriptor = f

		if imps[imp.Path()] {
			return nil, errors.New("already imported %q", path)
		}
		imps[imp.Path()] = true
	}
	for i, _ := range fd.GetDependency() {
		imp := &f.L2.Imports[i]
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
	r2 := &resolver{local: r1, remote: r, imports: imps, allowUnresolvable: allowUnresolvable}
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
