// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import (
	"google.golang.org/protobuf/internal/pragma"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// MergeOptions configures the merger.
//
// Example usage:
//   MergeOptions{Shallow: true}.Merge(dst, src)
type MergeOptions struct {
	pragma.NoUnkeyedLiterals

	// Shallow configures Merge to shallow copy messages, lists, and maps
	// instead of allocating new ones in the destination if it does not already
	// have one populated. Scalar bytes are copied by reference.
	// If true, Merge must be given messages of the same concrete type.
	//
	// If false, Merge is guaranteed to produce deep copies of all mutable
	// objects from the source into the destination. Since scalar bytes are
	// mutable they are deep copied as a result.
	//
	// Invariant:
	//	var dst1, dst2 Message = ...
	//	Equal(dst1, dst2) // assume equal initially
	//	MergeOptions{Shallow: true}.Merge(dst1, src)
	//	MergeOptions{Shallow: false}.Merge(dst2, src)
	//	Equal(dst1, dst2) // equal regardless of whether Shallow is specified
	Shallow bool
}

// Merge merges src into dst, which must be messages with the same descriptor.
// See MergeOptions.Merge for details.
func Merge(dst, src Message) {
	MergeOptions{}.Merge(dst, src)
}

// Merge merges src into dst, which must be messages with the same descriptor.
//
// Populated scalar fields in src are copied to dst, while populated
// singular messages in src are merged into dst by recursively calling Merge.
// The elements of every list field in src is appended to the corresponded
// list fields in dst. The entries of every map field in src is copied into
// the corresponding map field in dst, possibly replacing existing entries.
// The unknown fields of src are appended to the unknown fields of dst.
//
// It is semantically equivalent to unmarshaling the encoded form of src
// into dst with the UnmarshalOptions.Merge option specified.
func (o MergeOptions) Merge(dst, src Message) {
	dstMsg, srcMsg := dst.ProtoReflect(), src.ProtoReflect()
	if o.Shallow {
		if dstMsg.Type() != srcMsg.Type() {
			panic("type mismatch")
		}
	} else {
		if dstMsg.Descriptor() != srcMsg.Descriptor() {
			panic("descriptor mismatch")
		}
	}
	o.mergeMessage(dstMsg, srcMsg)
}

func (o MergeOptions) mergeMessage(dst, src protoreflect.Message) {
	src.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch {
		case fd.IsList():
			if o.Shallow && !dst.Has(fd) {
				dst.Set(fd, v)
			} else {
				o.mergeList(dst.Mutable(fd).List(), v.List(), fd)
			}
		case fd.IsMap():
			if o.Shallow && !dst.Has(fd) {
				dst.Set(fd, v)
			} else {
				o.mergeMap(dst.Mutable(fd).Map(), v.Map(), fd.MapValue())
			}
		case fd.Message() != nil:
			if o.Shallow && !dst.Has(fd) {
				dst.Set(fd, v)
			} else {
				o.mergeMessage(dst.Mutable(fd).Message(), v.Message())
			}
		case fd.Kind() == protoreflect.BytesKind:
			dst.Set(fd, o.cloneBytes(v))
		default:
			dst.Set(fd, v)
		}
		return true
	})

	if len(src.GetUnknown()) > 0 {
		if o.Shallow && dst.GetUnknown() == nil {
			dst.SetUnknown(src.GetUnknown())
		} else {
			dst.SetUnknown(append(dst.GetUnknown(), src.GetUnknown()...))
		}
	}
}

func (o MergeOptions) mergeList(dst, src protoreflect.List, fd protoreflect.FieldDescriptor) {
	// Merge semantics appends to the end of the existing list.
	for i, n := 0, src.Len(); i < n; i++ {
		switch v := src.Get(i); {
		case fd.Message() != nil:
			if o.Shallow {
				dst.Append(v)
			} else {
				dstv := dst.NewElement()
				o.mergeMessage(dstv.Message(), v.Message())
				dst.Append(dstv)
			}
		case fd.Kind() == protoreflect.BytesKind:
			dst.Append(o.cloneBytes(v))
		default:
			dst.Append(v)
		}
	}
}

func (o MergeOptions) mergeMap(dst, src protoreflect.Map, fd protoreflect.FieldDescriptor) {
	// Merge semantics replaces, rather than merges into existing entries.
	src.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
		switch {
		case fd.Message() != nil:
			if o.Shallow {
				dst.Set(k, v)
			} else {
				dstv := dst.NewValue()
				o.mergeMessage(dstv.Message(), v.Message())
				dst.Set(k, dstv)
			}
		case fd.Kind() == protoreflect.BytesKind:
			dst.Set(k, o.cloneBytes(v))
		default:
			dst.Set(k, v)
		}
		return true
	})
}

func (o MergeOptions) cloneBytes(v protoreflect.Value) protoreflect.Value {
	if o.Shallow {
		return v
	}
	return protoreflect.ValueOfBytes(append([]byte{}, v.Bytes()...))
}
