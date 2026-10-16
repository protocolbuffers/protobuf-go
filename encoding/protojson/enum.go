// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protojson

import (
	"google.golang.org/protobuf/internal/genid"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/jsonenumvalueoptionspb"
)

// enumValueJSONNameOption reads the (pb.enumvalue.json).string option on desc,
// returning the custom JSON name and true if set.
func enumValueJSONNameOption(desc protoreflect.EnumValueDescriptor) (string, bool) {
	opts := desc.Options()
	if opts == nil || !opts.ProtoReflect().IsValid() {
		return "", false
	}
	// We must not use proto.GetExtension(opts, jsonenumvalueoptionspb.E_Json)
	// because that only works for messages we generated, but not for
	// dynamicpb messages. See golang/protobuf#1669.
	jsonOpts := opts.ProtoReflect().Get(jsonenumvalueoptionspb.E_Json.TypeDescriptor())
	if !jsonOpts.IsValid() {
		return "", false
	}
	jm, ok := jsonOpts.Interface().(protoreflect.Message)
	if !ok {
		return "", false
	}
	fields := jm.Descriptor().Fields()
	if fd := fields.ByNumber(genid.JsonEnumValueOptions_String__field_number); fd != nil &&
		!fd.IsList() &&
		fd.Kind() == protoreflect.StringKind &&
		jm.Has(fd) {
		return jm.Get(fd).String(), true
	}
	return "", false
}
