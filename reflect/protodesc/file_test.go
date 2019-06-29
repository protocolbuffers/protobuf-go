// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protodesc

import (
	"fmt"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"

	"google.golang.org/protobuf/types/descriptorpb"
)

func mustParseFile(s string) *descriptorpb.FileDescriptorProto {
	pb := new(descriptorpb.FileDescriptorProto)
	if err := prototext.Unmarshal([]byte(s), pb); err != nil {
		panic(err)
	}
	return pb
}

func cloneFile(in *descriptorpb.FileDescriptorProto) *descriptorpb.FileDescriptorProto {
	out := new(descriptorpb.FileDescriptorProto)
	proto.Merge(out, in)
	return out
}

func TestNewFile(t *testing.T) {
	tests := []struct {
		label    string
		inDeps   []*descriptorpb.FileDescriptorProto
		inDesc   *descriptorpb.FileDescriptorProto
		inOpts   []option
		wantDesc *descriptorpb.FileDescriptorProto
		wantErr  string
	}{{
		label: "resolve relative reference",
		inDesc: mustParseFile(`
			name: "test.proto"
			package: "fizz.buzz"
			message_type: [{
				name: "A"
				field: [{name:"F" number:1 label:LABEL_OPTIONAL type:TYPE_MESSAGE type_name:"B.C"}]
				nested_type: [{name: "B"}]
			}, {
				name: "B"
				nested_type: [{name: "C"}]
			}]
		`),
		wantDesc: mustParseFile(`
			name: "test.proto"
			package: "fizz.buzz"
			message_type: [{
				name: "A"
				field: [{name:"F" number:1 label:LABEL_OPTIONAL type:TYPE_MESSAGE type_name:".fizz.buzz.B.C"}]
				nested_type: [{name: "B"}]
			}, {
				name: "B"
				nested_type: [{name: "C"}]
			}]
		`),
	}, {
		label: "resolve the wrong type",
		inDesc: mustParseFile(`
			name: "test.proto"
			package: ""
			message_type: [{
				name: "M"
				field: [{name:"F" number:1 label:LABEL_OPTIONAL type:TYPE_MESSAGE type_name:"E"}]
				enum_type: [{name: "E" value: [{name:"V0" number:0}, {name:"V1" number:1}]}]
			}]
		`),
		wantErr: `message field "M.F" cannot resolve type: resolved "M.E", but it is not an message`,
	}, {
		label: "auto-resolve unknown kind",
		inDesc: mustParseFile(`
			name: "test.proto"
			package: ""
			message_type: [{
				name: "M"
				field: [{name:"F" number:1 label:LABEL_OPTIONAL type_name:"E"}]
				enum_type: [{name: "E" value: [{name:"V0" number:0}, {name:"V1" number:1}]}]
			}]
		`),
		wantDesc: mustParseFile(`
			name: "test.proto"
			package: ""
			message_type: [{
				name: "M"
				field: [{name:"F" number:1 label:LABEL_OPTIONAL type:TYPE_ENUM type_name:".M.E"}]
				enum_type: [{name: "E" value: [{name:"V0" number:0}, {name:"V1" number:1}]}]
			}]
		`),
	}, {
		label: "unresolved import",
		inDesc: mustParseFile(`
			name: "test.proto"
			package: "fizz.buzz"
			dependency: "remote.proto"
		`),
		wantErr: `could not resolve import "remote.proto": not found`,
	}, {
		label: "unresolved message field",
		inDesc: mustParseFile(`
			name: "test.proto"
			package: "fizz.buzz"
			message_type: [{
				name: "M"
				field: [{name:"F1" number:1 label:LABEL_OPTIONAL type:TYPE_ENUM type_name:"some.other.enum" default_value:"UNKNOWN"}]
			}]
		`),
		wantErr: `message field "fizz.buzz.M.F1" cannot resolve type: "*.some.other.enum" not found`,
	}, {
		label: "unresolved default enum value",
		inDesc: mustParseFile(`
			name: "test.proto"
			package: "fizz.buzz"
			message_type: [{
				name: "M"
				field: [{name:"F1" number:1 label:LABEL_OPTIONAL type:TYPE_ENUM type_name:"E" default_value:"UNKNOWN"}]
				enum_type: [{name:"E" value:[{name:"V0" number:0}]}]
			}]
		`),
		wantErr: `message field "fizz.buzz.M.F1" has invalid default: could not parse value for enum: "UNKNOWN"`,
	}, {
		label: "allowed unresolved default enum value",
		inDesc: mustParseFile(`
			name: "test.proto"
			package: "fizz.buzz"
			message_type: [{
				name: "M"
				field: [{name:"F1" number:1 label:LABEL_OPTIONAL type:TYPE_ENUM type_name:".fizz.buzz.M.E" default_value:"UNKNOWN"}]
				enum_type: [{name:"E" value:[{name:"V0" number:0}]}]
			}]
		`),
		inOpts: []option{allowUnresolvable()},
	}, {
		label: "unresolved extendee",
		inDesc: mustParseFile(`
			name: "test.proto"
			package: "fizz.buzz"
			extension: [{name:"X" number:1 label:LABEL_OPTIONAL extendee:"some.extended.message" type:TYPE_MESSAGE type_name:"some.other.message"}]
		`),
		wantErr: `extension field "fizz.buzz.X" cannot resolve extendee: "*.some.extended.message" not found`,
	}, {
		label: "unresolved method input",
		inDesc: mustParseFile(`
			name: "test.proto"
			package: "fizz.buzz"
			service: [{
				name: "S"
				method: [{name:"M" input_type:"foo.bar.input" output_type:".absolute.foo.bar.output"}]
			}]
		`),
		wantErr: `service method "fizz.buzz.S.M" cannot resolve input: "*.foo.bar.input" not found`,
	}, {
		label: "allowed unresolved references",
		inDesc: mustParseFile(`
			name: "test.proto"
			package: "fizz.buzz"
			dependency: "remote.proto"
			message_type: [{
				name: "M"
				field: [{name:"F1" number:1 label:LABEL_OPTIONAL type_name:"some.other.enum" default_value:"UNKNOWN"}]
			}]
			extension: [{name:"X" number:1 label:LABEL_OPTIONAL extendee:"some.extended.message" type:TYPE_MESSAGE type_name:"some.other.message"}]
			service: [{
				name: "S"
				method: [{name:"M" input_type:"foo.bar.input" output_type:".absolute.foo.bar.output"}]
			}]
		`),
		inOpts: []option{allowUnresolvable()},
	}, {
		label: "resolved but not imported",
		inDeps: []*descriptorpb.FileDescriptorProto{mustParseFile(`
			name: "dep.proto"
			package: "fizz"
			message_type: [{name:"M" nested_type:[{name:"M"}]}]
		`)},
		inDesc: mustParseFile(`
			name: "test.proto"
			package: "fizz.buzz"
			message_type: [{
				name: "M"
				field: [{name:"F" number:1 label:LABEL_OPTIONAL type:TYPE_MESSAGE type_name:"M.M"}]
			}]
		`),
		wantErr: `message field "fizz.buzz.M.F" cannot resolve type: resolved "fizz.M.M", but "dep.proto" is not imported`,
	}, {
		label: "resolved from remote import",
		inDeps: []*descriptorpb.FileDescriptorProto{mustParseFile(`
			name: "dep.proto"
			package: "fizz"
			message_type: [{name:"M" nested_type:[{name:"M"}]}]
		`)},
		inDesc: mustParseFile(`
			name: "test.proto"
			package: "fizz.buzz"
			dependency: "dep.proto"
			message_type: [{
				name: "M"
				field: [{name:"F" number:1 label:LABEL_OPTIONAL type:TYPE_MESSAGE type_name:"M.M"}]
			}]
		`),
		wantDesc: mustParseFile(`
			name: "test.proto"
			package: "fizz.buzz"
			dependency: "dep.proto"
			message_type: [{
				name: "M"
				field: [{name:"F" number:1 label:LABEL_OPTIONAL type:TYPE_MESSAGE type_name:".fizz.M.M"}]
			}]
		`),
	}}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			r := new(protoregistry.Files)
			for i, dep := range tt.inDeps {
				f, err := newFile(dep, r)
				if err != nil {
					t.Fatalf("dependency %d: unexpected NewFile() error: %v", i, err)
				}
				if err := r.Register(f); err != nil {
					t.Fatalf("dependency %d: unexpected Register() error: %v", i, err)
				}
			}
			var gotDesc *descriptorpb.FileDescriptorProto
			if tt.wantErr == "" && tt.wantDesc == nil {
				tt.wantDesc = cloneFile(tt.inDesc)
			}
			gotFile, err := newFile(tt.inDesc, r, tt.inOpts...)
			if gotFile != nil {
				gotDesc = ToFileDescriptorProto(gotFile)
			}
			if !proto.Equal(gotDesc, tt.wantDesc) {
				t.Errorf("NewFile() mismatch:\ngot  %v\nwant %v", gotDesc, tt.wantDesc)
			}
			if ((err == nil) != (tt.wantErr == "")) || !strings.Contains(fmt.Sprint(err), tt.wantErr) {
				t.Errorf("NewFile() error:\ngot:  %v\nwant: %v", err, tt.wantErr)
			}
		})
	}
}
