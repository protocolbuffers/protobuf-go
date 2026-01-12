// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package featureresolution_test

import (
	_ "embed"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	basicpb "google.golang.org/protobuf/cmd/protoc-gen-go/testdata/featureresolution"
	testfeaturespb "google.golang.org/protobuf/cmd/protoc-gen-go/testdata/features"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/runtime/protoimpl"
	descpb "google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/gofeaturespb"
	"google.golang.org/protobuf/types/pluginpb"
)

var (
	//go:embed test_features_defaults.binpb
	featureSetDefaultsRaw []byte
	featureSetDefaults    *descpb.FeatureSetDefaults
)

func init() {
	featureSetDefaults = &descpb.FeatureSetDefaults{}
	if err := proto.Unmarshal(featureSetDefaultsRaw, featureSetDefaults); err != nil {
		panic(err)
	}
}

func createTestFile() *descpb.FileDescriptorProto {
	return protodesc.ToFileDescriptorProto(basicpb.File_cmd_protoc_gen_go_testdata_featureresolution_basic_proto)
}

func protogenFor(t *testing.T, f *descpb.FileDescriptorProto) *protogen.File {
	t.Helper()

	// Construct a Protobuf plugin code generation request based on the
	// transitive closure of dependencies of message m.
	req := &pluginpb.CodeGeneratorRequest{
		ProtoFile: []*descpb.FileDescriptorProto{
			protodesc.ToFileDescriptorProto(descpb.File_google_protobuf_descriptor_proto),
			protodesc.ToFileDescriptorProto(gofeaturespb.File_google_protobuf_go_features_proto),
			protodesc.ToFileDescriptorProto(testfeaturespb.File_cmd_protoc_gen_go_testdata_features_test_features_proto),
			f,
		},
	}
	plugin, err := protogen.Options{
		FeatureSetDefaults: featureSetDefaults,
	}.New(req)
	if err != nil {
		t.Fatalf("protogen.Options.New: %v", err)
	}
	if got, want := len(plugin.Files), len(req.ProtoFile); got != want {
		t.Fatalf("protogen returned %d plugin.Files entries, expected %d", got, want)
	}
	// The last file topologically is the one that we care about.
	return plugin.Files[len(plugin.Files)-1]
}

func checkFeature(t *testing.T, features *descpb.FeatureSet, ext *protoimpl.ExtensionInfo, name string, value any) {
	t.Helper()
	reflect := features.ProtoReflect()
	if ext != nil {
		reflect = proto.GetExtension(features, ext).(protoreflect.ProtoMessage).ProtoReflect()
	}
	field := reflect.Descriptor().Fields().ByName(protoreflect.Name(name))
	if field == nil {
		t.Fatalf("feature %q not found", name)
	}
	got := reflect.Get(field)
	want := protoreflect.ValueOf(value)
	if eq := cmp.Equal(got, want); !eq {
		t.Errorf("feature %q = %v, want %v", name, got, want)
	}
}

func checkGlobalFeature(t *testing.T, features *descpb.FeatureSet, name string, value any) {
	t.Helper()
	checkFeature(t, features, nil, name, value)
}

func checkTestFeature(t *testing.T, features *descpb.FeatureSet, name string, value any) {
	t.Helper()
	checkFeature(t, features, testfeaturespb.E_TestFeatures, name, value)
}

type testCase struct {
	name     string
	features *descpb.FeatureSet
}

func createAllTestCases(file *protogen.File) []testCase {
	testCases := []testCase{
		{
			name:     "file",
			features: file.ResolvedFeatures,
		},
		{
			name:     "top_message",
			features: file.Messages[0].ResolvedFeatures,
		},
		{
			name:     "top_enum",
			features: file.Enums[0].ResolvedFeatures,
		},
		{
			name:     "top_enum_value",
			features: file.Enums[0].Values[0].ResolvedFeatures,
		},
		{
			name:     "field",
			features: file.Messages[0].Fields[0].ResolvedFeatures,
		},
		{
			name:     "oneof",
			features: file.Messages[0].Oneofs[0].ResolvedFeatures,
		},
		{
			name:     "oneof_field",
			features: file.Messages[0].Fields[1].ResolvedFeatures,
		},
		{
			name:     "nested_message",
			features: file.Messages[0].Messages[0].ResolvedFeatures,
		},
		{
			name:     "nested_field",
			features: file.Messages[0].Messages[0].Fields[0].ResolvedFeatures,
		},
		{
			name:     "nested_enum",
			features: file.Messages[0].Enums[0].ResolvedFeatures,
		},
		{
			name:     "nested_enum_value",
			features: file.Messages[0].Enums[0].Values[0].ResolvedFeatures,
		},
		{
			name:     "service",
			features: file.Services[0].ResolvedFeatures,
		},
		{
			name:     "method",
			features: file.Services[0].Methods[0].ResolvedFeatures,
		},
	}
	if file.Proto.GetSyntax() != "proto3" {
		// Extensions aren't allowed in proto3.
		testCases = append(testCases,
			testCase{
				name:     "extension",
				features: file.Extensions[0].ResolvedFeatures,
			},
			testCase{
				name:     "nested_extension",
				features: file.Messages[0].Extensions[0].ResolvedFeatures,
			},
		)
	}
	return testCases
}

// splitTestCases splits the provided test cases into two sets: the first set
// contains all test cases that are covered by the provided names and the
// second set contains all other test cases.
func splitTestCases(testCases []testCase, split []string) ([]testCase, []testCase) {
	var covered []testCase
	var uncovered []testCase
	reached := make(map[string]bool)
	for _, tc := range testCases {
		if slices.Contains(split, tc.name) {
			covered = append(covered, tc)
			reached[tc.name] = true
		} else {
			uncovered = append(uncovered, tc)
		}
	}
	if len(reached) != len(split) {
		panic(fmt.Sprintf("%d test cases not found", len(split)-len(reached)))
	}
	return covered, uncovered
}

func TestProto2Defaults(t *testing.T) {
	fd := createTestFile()
	fd.Syntax = proto.String("proto2")
	fd.Edition = nil
	file := protogenFor(t, fd)

	for _, tc := range createAllTestCases(file) {
		t.Run(tc.name, func(t *testing.T) {
			checkGlobalFeature(t, tc.features, "field_presence", descpb.FeatureSet_EXPLICIT.Number())
			checkTestFeature(t, tc.features, "enum_feature", testfeaturespb.EnumFeature_VALUE1.Number())
			checkTestFeature(t, tc.features, "bool_feature", false)
		})
	}
}

func TestProto3Defaults(t *testing.T) {
	fd := createTestFile()
	fd.Syntax = proto.String("proto3")
	fd.Edition = nil
	fd.MessageType[0].ExtensionRange = nil
	fd.MessageType[0].Extension = nil
	fd.Extension = nil
	file := protogenFor(t, fd)

	for _, tc := range createAllTestCases(file) {
		t.Run(tc.name, func(t *testing.T) {
			checkGlobalFeature(t, tc.features, "field_presence", descpb.FeatureSet_IMPLICIT.Number())
			checkTestFeature(t, tc.features, "enum_feature", testfeaturespb.EnumFeature_VALUE1.Number())
			checkTestFeature(t, tc.features, "bool_feature", false)
		})
	}
}

func TestEdition2023Defaults(t *testing.T) {
	fd := createTestFile()
	file := protogenFor(t, fd)

	for _, tc := range createAllTestCases(file) {
		t.Run(tc.name, func(t *testing.T) {
			checkGlobalFeature(t, tc.features, "field_presence", descpb.FeatureSet_EXPLICIT.Number())
			checkTestFeature(t, tc.features, "enum_feature", testfeaturespb.EnumFeature_VALUE2.Number())
			checkTestFeature(t, tc.features, "bool_feature", true)
		})
	}
}

func TestEditionUnstable(t *testing.T) {
	fd := createTestFile()
	fd.Edition = descpb.Edition_EDITION_UNSTABLE.Enum()
	file := protogenFor(t, fd)

	checkTestFeature(t, file.ResolvedFeatures, "unstable_feature", testfeaturespb.EnumFeature_VALUE2.Number())
}

func TestInheritance(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*descpb.FileDescriptorProto, *descpb.FeatureSet)
		inherit []string
	}{
		{
			name: "file",
			setup: func(fd *descpb.FileDescriptorProto, f *descpb.FeatureSet) {
				fd.Options.Features = f
			},
			inherit: []string{
				"file",
				"top_message",
				"top_enum",
				"top_enum_value",
				"field",
				"oneof",
				"oneof_field",
				"nested_message",
				"nested_enum",
				"nested_enum_value",
				"nested_extension",
				"nested_field",
				"extension",
				"service",
				"method",
			},
		},
		{
			name: "message",
			setup: func(fd *descpb.FileDescriptorProto, f *descpb.FeatureSet) {
				fd.MessageType[0].Options = &descpb.MessageOptions{Features: f}
			},
			inherit: []string{
				"top_message",
				"field",
				"oneof",
				"oneof_field",
				"nested_message",
				"nested_enum",
				"nested_enum_value",
				"nested_extension",
				"nested_field",
			},
		},
		{
			name: "field",
			setup: func(fd *descpb.FileDescriptorProto, f *descpb.FeatureSet) {
				fd.MessageType[0].Field[0].Options = &descpb.FieldOptions{Features: f}
			},
			inherit: []string{
				"field",
			},
		},
		{
			name: "oneof",
			setup: func(fd *descpb.FileDescriptorProto, f *descpb.FeatureSet) {
				fd.MessageType[0].OneofDecl[0].Options = &descpb.OneofOptions{Features: f}
			},
			inherit: []string{
				"oneof",
				"oneof_field",
			},
		},
		{
			name: "oneof_field",
			setup: func(fd *descpb.FileDescriptorProto, f *descpb.FeatureSet) {
				fd.MessageType[0].Field[1].Options = &descpb.FieldOptions{Features: f}
			},
			inherit: []string{
				"oneof_field",
			},
		},
		{
			name: "nested_message",
			setup: func(fd *descpb.FileDescriptorProto, f *descpb.FeatureSet) {
				fd.MessageType[0].NestedType[0].Options = &descpb.MessageOptions{Features: f}
			},
			inherit: []string{
				"nested_message",
				"nested_field",
			},
		},
		{
			name: "nested_field",
			setup: func(fd *descpb.FileDescriptorProto, f *descpb.FeatureSet) {
				fd.MessageType[0].NestedType[0].Field[0].Options = &descpb.FieldOptions{Features: f}
			},
			inherit: []string{
				"nested_field",
			},
		},
		{
			name: "nested_extension",
			setup: func(fd *descpb.FileDescriptorProto, f *descpb.FeatureSet) {
				fd.MessageType[0].Extension[0].Options = &descpb.FieldOptions{Features: f}
			},
			inherit: []string{
				"nested_extension",
			},
		},
		{
			name: "nested_enum",
			setup: func(fd *descpb.FileDescriptorProto, f *descpb.FeatureSet) {
				fd.MessageType[0].EnumType[0].Options = &descpb.EnumOptions{Features: f}
			},
			inherit: []string{
				"nested_enum",
				"nested_enum_value",
			},
		},
		{
			name: "nested_enum_value",
			setup: func(fd *descpb.FileDescriptorProto, f *descpb.FeatureSet) {
				fd.MessageType[0].EnumType[0].Value[0].Options = &descpb.EnumValueOptions{Features: f}
			},
			inherit: []string{
				"nested_enum_value",
			},
		},
		{
			name: "enum",
			setup: func(fd *descpb.FileDescriptorProto, f *descpb.FeatureSet) {
				fd.EnumType[0].Options = &descpb.EnumOptions{Features: f}
			},
			inherit: []string{
				"top_enum",
				"top_enum_value",
			},
		},
		{
			name: "enum_value",
			setup: func(fd *descpb.FileDescriptorProto, f *descpb.FeatureSet) {
				fd.EnumType[0].Value[0].Options = &descpb.EnumValueOptions{Features: f}
			},
			inherit: []string{
				"top_enum_value",
			},
		},
		{
			name: "extension",
			setup: func(fd *descpb.FileDescriptorProto, f *descpb.FeatureSet) {
				fd.Extension[0].Options = &descpb.FieldOptions{Features: f}
			},
			inherit: []string{
				"extension",
			},
		},
		{
			name: "service",
			setup: func(fd *descpb.FileDescriptorProto, f *descpb.FeatureSet) {
				fd.Service[0].Options = &descpb.ServiceOptions{Features: f}
			},
			inherit: []string{
				"service",
				"method",
			},
		},
		{
			name: "method",
			setup: func(fd *descpb.FileDescriptorProto, f *descpb.FeatureSet) {
				fd.Service[0].Method[0].Options = &descpb.MethodOptions{Features: f}
			},
			inherit: []string{
				"method",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fd := createTestFile()

			features := &descpb.FeatureSet{
				FieldPresence: descpb.FeatureSet_IMPLICIT.Enum(),
			}
			proto.SetExtension(features, testfeaturespb.E_TestFeatures, testfeaturespb.TestFeatures_builder{
				EnumFeature: testfeaturespb.EnumFeature_VALUE4.Enum(),
			}.Build())

			tc.setup(fd, features)
			file := protogenFor(t, fd)

			inherit, def := splitTestCases(createAllTestCases(file), tc.inherit)

			for _, tc2 := range inherit {
				t.Run(tc2.name, func(t *testing.T) {
					checkGlobalFeature(t, tc2.features, "field_presence", descpb.FeatureSet_IMPLICIT.Number())
					checkTestFeature(t, tc2.features, "enum_feature", testfeaturespb.EnumFeature_VALUE4.Number())
				})
			}

			for _, tc2 := range def {
				t.Run(tc2.name, func(t *testing.T) {
					checkGlobalFeature(t, tc2.features, "field_presence", descpb.FeatureSet_EXPLICIT.Number())
					checkTestFeature(t, tc2.features, "enum_feature", testfeaturespb.EnumFeature_VALUE2.Number())
				})
			}
		})
	}
}

func TestOverride(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*descpb.FileDescriptorProto, *descpb.FeatureSet, *descpb.FeatureSet)
		inherit  []string
		override []string
	}{
		{
			name: "file_message",
			setup: func(fd *descpb.FileDescriptorProto, f1 *descpb.FeatureSet, f2 *descpb.FeatureSet) {
				fd.Options.Features = f1
				fd.MessageType[0].Options = &descpb.MessageOptions{Features: f2}
			},
			inherit: []string{
				"file",
				"top_enum",
				"top_enum_value",
				"extension",
				"service",
				"method",
			},
			override: []string{
				"top_message",
				"field",
				"oneof",
				"oneof_field",
				"nested_message",
				"nested_enum",
				"nested_enum_value",
				"nested_extension",
				"nested_field",
			},
		},
		{
			name: "file_enum",
			setup: func(fd *descpb.FileDescriptorProto, f1 *descpb.FeatureSet, f2 *descpb.FeatureSet) {
				fd.Options.Features = f1
				fd.EnumType[0].Options = &descpb.EnumOptions{Features: f2}
			},
			inherit: []string{
				"file",
				"top_message",
				"field",
				"oneof",
				"oneof_field",
				"nested_message",
				"nested_enum",
				"nested_enum_value",
				"nested_extension",
				"nested_field",
				"extension",
				"service",
				"method",
			},
			override: []string{
				"top_enum",
				"top_enum_value",
			},
		},
		{
			name: "message_enum",
			setup: func(fd *descpb.FileDescriptorProto, f1 *descpb.FeatureSet, f2 *descpb.FeatureSet) {
				fd.MessageType[0].Options = &descpb.MessageOptions{Features: f1}
				fd.MessageType[0].EnumType[0].Options = &descpb.EnumOptions{Features: f2}
			},
			inherit: []string{
				"top_message",
				"field",
				"oneof",
				"oneof_field",
				"nested_message",
				"nested_extension",
				"nested_field",
			},
			override: []string{
				"nested_enum",
				"nested_enum_value",
			},
		},
		{
			name: "message_message",
			setup: func(fd *descpb.FileDescriptorProto, f1 *descpb.FeatureSet, f2 *descpb.FeatureSet) {
				fd.MessageType[0].Options = &descpb.MessageOptions{Features: f1}
				fd.MessageType[0].NestedType[0].Options = &descpb.MessageOptions{Features: f2}
			},
			inherit: []string{
				"top_message",
				"field",
				"oneof",
				"oneof_field",
				"nested_enum",
				"nested_enum_value",
				"nested_extension",
			},
			override: []string{
				"nested_message",
				"nested_field",
			},
		},
		{
			name: "message_oneof",
			setup: func(fd *descpb.FileDescriptorProto, f1 *descpb.FeatureSet, f2 *descpb.FeatureSet) {
				fd.MessageType[0].Options = &descpb.MessageOptions{Features: f1}
				fd.MessageType[0].OneofDecl[0].Options = &descpb.OneofOptions{Features: f2}
			},
			inherit: []string{
				"top_message",
				"field",
				"nested_message",
				"nested_enum",
				"nested_enum_value",
				"nested_extension",
				"nested_field",
			},
			override: []string{
				"oneof",
				"oneof_field",
			},
		},
		{
			name: "message_extension",
			setup: func(fd *descpb.FileDescriptorProto, f1 *descpb.FeatureSet, f2 *descpb.FeatureSet) {
				fd.MessageType[0].Options = &descpb.MessageOptions{Features: f1}
				fd.MessageType[0].Extension[0].Options = &descpb.FieldOptions{Features: f2}
			},
			inherit: []string{
				"top_message",
				"field",
				"nested_message",
				"nested_field",
				"nested_enum",
				"nested_enum_value",
				"oneof",
				"oneof_field",
			},
			override: []string{
				"nested_extension",
			},
		},
		{
			name: "message_field",
			setup: func(fd *descpb.FileDescriptorProto, f1 *descpb.FeatureSet, f2 *descpb.FeatureSet) {
				fd.MessageType[0].Options = &descpb.MessageOptions{Features: f1}
				fd.MessageType[0].Field[0].Options = &descpb.FieldOptions{Features: f2}
			},
			inherit: []string{
				"top_message",
				"nested_message",
				"nested_field",
				"nested_extension",
				"nested_enum",
				"nested_enum_value",
				"oneof",
				"oneof_field",
			},
			override: []string{
				"field",
			},
		},
		{
			name: "enum_value",
			setup: func(fd *descpb.FileDescriptorProto, f1 *descpb.FeatureSet, f2 *descpb.FeatureSet) {
				fd.EnumType[0].Options = &descpb.EnumOptions{Features: f1}
				fd.EnumType[0].Value[0].Options = &descpb.EnumValueOptions{Features: f2}
			},
			inherit: []string{
				"top_enum",
			},
			override: []string{
				"top_enum_value",
			},
		},
		{
			name: "file_extension",
			setup: func(fd *descpb.FileDescriptorProto, f1 *descpb.FeatureSet, f2 *descpb.FeatureSet) {
				fd.Options.Features = f1
				fd.Extension[0].Options = &descpb.FieldOptions{Features: f2}
			},
			inherit: []string{
				"file",
				"top_message",
				"top_enum",
				"top_enum_value",
				"field",
				"oneof",
				"oneof_field",
				"nested_message",
				"nested_enum",
				"nested_enum_value",
				"nested_extension",
				"nested_field",
				"service",
				"method",
			},
			override: []string{
				"extension",
			},
		},
		{
			name: "file_service",
			setup: func(fd *descpb.FileDescriptorProto, f1 *descpb.FeatureSet, f2 *descpb.FeatureSet) {
				fd.Options.Features = f1
				fd.Service[0].Options = &descpb.ServiceOptions{Features: f2}
			},
			inherit: []string{
				"file",
				"top_message",
				"top_enum",
				"top_enum_value",
				"field",
				"oneof",
				"oneof_field",
				"nested_message",
				"nested_enum",
				"nested_enum_value",
				"nested_extension",
				"nested_field",
				"extension",
			},
			override: []string{
				"service",
				"method",
			},
		},
		{
			name: "service_method",
			setup: func(fd *descpb.FileDescriptorProto, f1 *descpb.FeatureSet, f2 *descpb.FeatureSet) {
				fd.Service[0].Options = &descpb.ServiceOptions{Features: f1}
				fd.Service[0].Method[0].Options = &descpb.MethodOptions{Features: f2}
			},
			inherit: []string{
				"service",
			},
			override: []string{
				"method",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fd := createTestFile()

			features1 := &descpb.FeatureSet{
				FieldPresence: descpb.FeatureSet_IMPLICIT.Enum(),
			}
			proto.SetExtension(features1, testfeaturespb.E_TestFeatures, testfeaturespb.TestFeatures_builder{
				EnumFeature: testfeaturespb.EnumFeature_VALUE4.Enum(),
			}.Build())

			features2 := &descpb.FeatureSet{
				FieldPresence: descpb.FeatureSet_EXPLICIT.Enum(),
			}
			proto.SetExtension(features2, testfeaturespb.E_TestFeatures, testfeaturespb.TestFeatures_builder{
				EnumFeature: testfeaturespb.EnumFeature_VALUE5.Enum(),
			}.Build())

			tc.setup(fd, features1, features2)
			file := protogenFor(t, fd)

			changed, def := splitTestCases(createAllTestCases(file), append(tc.inherit, tc.override...))
			override, inherit := splitTestCases(changed, tc.override)

			for _, tc2 := range inherit {
				t.Run(tc2.name, func(t *testing.T) {
					checkGlobalFeature(t, tc2.features, "field_presence", descpb.FeatureSet_IMPLICIT.Number())
					checkTestFeature(t, tc2.features, "enum_feature", testfeaturespb.EnumFeature_VALUE4.Number())
				})
			}

			for _, tc2 := range override {
				t.Run(tc2.name, func(t *testing.T) {
					checkGlobalFeature(t, tc2.features, "field_presence", descpb.FeatureSet_EXPLICIT.Number())
					checkTestFeature(t, tc2.features, "enum_feature", testfeaturespb.EnumFeature_VALUE5.Number())
				})
			}

			for _, tc2 := range def {
				t.Run(tc2.name, func(t *testing.T) {
					checkGlobalFeature(t, tc2.features, "field_presence", descpb.FeatureSet_EXPLICIT.Number())
					checkTestFeature(t, tc2.features, "enum_feature", testfeaturespb.EnumFeature_VALUE2.Number())
				})
			}
		})
	}
}

func TestErrorEditionTooEarly(t *testing.T) {
	fd := createTestFile()
	req := &pluginpb.CodeGeneratorRequest{
		ProtoFile: []*descpb.FileDescriptorProto{
			fd,
		},
	}

	defaults := proto.Clone(featureSetDefaults).(*descpb.FeatureSetDefaults)
	defaults.MinimumEdition = descpb.Edition_EDITION_2024.Enum()
	_, err := protogen.Options{
		FeatureSetDefaults: defaults,
	}.New(req)

	if err == nil {
		t.Error("protogen.Options.New: got nil, want error")
	}

	if want := "lower than the minimum supported edition"; !strings.Contains(err.Error(), want) {
		t.Errorf("protogen.Options.New: got error %v, want error containing %q", err, want)
	}
}

func TestErrorEditionTooLate(t *testing.T) {
	fd := createTestFile()
	fd.Edition = descpb.Edition_EDITION_2024.Enum()
	req := &pluginpb.CodeGeneratorRequest{
		ProtoFile: []*descpb.FileDescriptorProto{
			fd,
		},
	}

	defaults := proto.Clone(featureSetDefaults).(*descpb.FeatureSetDefaults)
	defaults.MaximumEdition = descpb.Edition_EDITION_2023.Enum()
	_, err := protogen.Options{
		FeatureSetDefaults: defaults,
	}.New(req)

	if err == nil {
		t.Error("protogen.Options.New: got nil, want error")
	}

	if !strings.Contains(err.Error(), "greater than the maximum supported edition") {
		t.Errorf("protogen.Options.New: got error %v, want error containing %q", err, "greater than the maximum supported edition")
	}
}

func TestErrorInvalidDefaults(t *testing.T) {
	fd := createTestFile()
	fd.Syntax = proto.String("proto2")
	fd.Edition = nil
	req := &pluginpb.CodeGeneratorRequest{
		ProtoFile: []*descpb.FileDescriptorProto{
			fd,
		},
	}

	defaults := proto.Clone(featureSetDefaults).(*descpb.FeatureSetDefaults)
	defaults.Defaults = defaults.GetDefaults()[2:]
	_, err := protogen.Options{
		FeatureSetDefaults: defaults,
	}.New(req)

	if err == nil {
		t.Error("protogen.Options.New: got nil, want error")
	}

	if !strings.Contains(err.Error(), "does not have a default") {
		t.Errorf("protogen.Options.New: got error %v, want error containing %q", err, "does not have a default")
	}
}
