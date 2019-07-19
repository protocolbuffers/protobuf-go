// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package internal_gengo is internal to the protobuf module.
package internal_gengo

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/internal/encoding/tag"
	"google.golang.org/protobuf/internal/fieldnum"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/runtime/protoimpl"

	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	// generateEnumMapVars specifies whether to generate enum maps,
	// which provide a bi-directional mapping between enum numbers and names.
	generateEnumMapVars = true

	// generateEnumJSONMethods specifies whether to generate the UnmarshalJSON
	// method for proto2 enums.
	generateEnumJSONMethods = true

	// generateRawDescMethods specifies whether to generate EnumDescriptor and
	// Descriptor methods for enums and messages. These methods return the
	// GZIP'd contents of the raw file descriptor and the path from the root
	// to the given enum or message descriptor.
	generateRawDescMethods = true

	// generateOneofWrapperMethods specifies whether to generate
	// XXX_OneofWrappers methods on messages with oneofs.
	generateOneofWrapperMethods = false

	// generateExtensionRangeMethods specifies whether to generate the
	// ExtensionRangeArray method for messages that support extensions.
	generateExtensionRangeMethods = true

	// generateMessageStateFields specifies whether to generate an unexported
	// protoimpl.MessageState as the first field.
	generateMessageStateFields = true

	// generateNoUnkeyedLiteralFields specifies whether to generate
	// the XXX_NoUnkeyedLiteral field.
	generateNoUnkeyedLiteralFields = false

	// generateExportedSizeCacheFields specifies whether to generate an exported
	// XXX_sizecache field instead of an unexported sizeCache field.
	generateExportedSizeCacheFields = false

	// generateExportedUnknownFields specifies whether to generate an exported
	// XXX_unrecognized field instead of an unexported unknownFields field.
	generateExportedUnknownFields = false

	// generateExportedExtensionFields specifies whether to generate an exported
	// XXX_InternalExtensions field instead of an unexported extensionFields field.
	generateExportedExtensionFields = false
)

// Standard library dependencies.
const (
	mathPackage    = protogen.GoImportPath("math")
	reflectPackage = protogen.GoImportPath("reflect")
	syncPackage    = protogen.GoImportPath("sync")
)

// Protobuf library dependencies.
//
// These are declared as an interface type so that they can be more easily
// patched to support unique build environments that impose restrictions
// on the dependencies of generated source code.
var (
	protoifacePackage    goImportPath = protogen.GoImportPath("google.golang.org/protobuf/runtime/protoiface")
	protoimplPackage     goImportPath = protogen.GoImportPath("google.golang.org/protobuf/runtime/protoimpl")
	protoreflectPackage  goImportPath = protogen.GoImportPath("google.golang.org/protobuf/reflect/protoreflect")
	protoregistryPackage goImportPath = protogen.GoImportPath("google.golang.org/protobuf/reflect/protoregistry")
	prototypePackage     goImportPath = protogen.GoImportPath("google.golang.org/protobuf/reflect/prototype")
)

type goImportPath interface {
	String() string
	Ident(string) protogen.GoIdent
}

type fileInfo struct {
	*protogen.File

	allEnums      []*protogen.Enum
	allMessages   []*protogen.Message
	allExtensions []*protogen.Extension

	allEnumsByPtr         map[*protogen.Enum]int    // value is index into allEnums
	allMessagesByPtr      map[*protogen.Message]int // value is index into allMessages
	allMessageFieldsByPtr map[*protogen.Message]*structFields
}

type structFields struct {
	count      int
	unexported map[int]string
}

func (sf *structFields) append(name string) {
	if r, _ := utf8.DecodeRuneInString(name); !unicode.IsUpper(r) {
		if sf.unexported == nil {
			sf.unexported = make(map[int]string)
		}
		sf.unexported[sf.count] = name
	}
	sf.count++
}

// GenerateFile generates the contents of a .pb.go file.
func GenerateFile(gen *protogen.Plugin, file *protogen.File) *protogen.GeneratedFile {
	filename := file.GeneratedFilenamePrefix + ".pb.go"
	g := gen.NewGeneratedFile(filename, file.GoImportPath)
	f := &fileInfo{
		File: file,
	}

	// Collect all enums, messages, and extensions in "flattened ordering".
	// See filetype.TypeBuilder.
	f.allEnums = append(f.allEnums, f.Enums...)
	f.allMessages = append(f.allMessages, f.Messages...)
	f.allExtensions = append(f.allExtensions, f.Extensions...)
	walkMessages(f.Messages, func(m *protogen.Message) {
		f.allEnums = append(f.allEnums, m.Enums...)
		f.allMessages = append(f.allMessages, m.Messages...)
		f.allExtensions = append(f.allExtensions, m.Extensions...)
	})

	// Derive a reverse mapping of enum and message pointers to their index
	// in allEnums and allMessages.
	if len(f.allEnums) > 0 {
		f.allEnumsByPtr = make(map[*protogen.Enum]int)
		for i, e := range f.allEnums {
			f.allEnumsByPtr[e] = i
		}
	}
	if len(f.allMessages) > 0 {
		f.allMessagesByPtr = make(map[*protogen.Message]int)
		f.allMessageFieldsByPtr = make(map[*protogen.Message]*structFields)
		for i, m := range f.allMessages {
			f.allMessagesByPtr[m] = i
			f.allMessageFieldsByPtr[m] = new(structFields)
		}
	}

	g.P("// Code generated by protoc-gen-go. DO NOT EDIT.")
	if f.Proto.GetOptions().GetDeprecated() {
		g.P("// ", f.Desc.Path(), " is a deprecated file.")
	} else {
		g.P("// source: ", f.Desc.Path())
	}
	g.P()
	g.PrintLeadingComments(protogen.Location{
		SourceFile: f.Proto.GetName(),
		Path:       []int32{fieldnum.FileDescriptorProto_Package},
	})
	g.P()
	g.P("package ", f.GoPackageName)
	g.P()

	// Emit a static check that enforces a minimum version of the proto package.
	g.P("const (")
	g.P("// Verify that runtime/protoimpl is sufficiently up-to-date.")
	g.P("_ = ", protoimplPackage.Ident("EnforceVersion"), "(", protoimplPackage.Ident("MaxVersion"), " - ", protoimpl.Version, ")")
	g.P("// Verify that this generated code is sufficiently up-to-date.")
	g.P("_ = ", protoimplPackage.Ident("EnforceVersion"), "(", protoimpl.Version, " - ", protoimplPackage.Ident("MinVersion"), ")")
	g.P(")")
	g.P()

	for i, imps := 0, f.Desc.Imports(); i < imps.Len(); i++ {
		genImport(gen, g, f, imps.Get(i))
	}
	for _, enum := range f.allEnums {
		genEnum(gen, g, f, enum)
	}
	for _, message := range f.allMessages {
		genMessage(gen, g, f, message)
	}
	genExtensions(gen, g, f)

	genReflectFileDescriptor(gen, g, f)

	return g
}

// walkMessages calls f on each message and all of its descendants.
func walkMessages(messages []*protogen.Message, f func(*protogen.Message)) {
	for _, m := range messages {
		f(m)
		walkMessages(m.Messages, f)
	}
}

func genImport(gen *protogen.Plugin, g *protogen.GeneratedFile, f *fileInfo, imp protoreflect.FileImport) {
	impFile, ok := gen.FileByName(imp.Path())
	if !ok {
		return
	}
	if impFile.GoImportPath == f.GoImportPath {
		// Don't generate imports or aliases for types in the same Go package.
		return
	}
	// Generate imports for all non-weak dependencies, even if they are not
	// referenced, because other code and tools depend on having the
	// full transitive closure of protocol buffer types in the binary.
	if !imp.IsWeak {
		g.Import(impFile.GoImportPath)
	}
	if !imp.IsPublic {
		return
	}

	// Generate public imports by generating the imported file, parsing it,
	// and extracting every symbol that should receive a forwarding declaration.
	impGen := GenerateFile(gen, impFile)
	impGen.Skip()
	b, err := impGen.Content()
	if err != nil {
		gen.Error(err)
		return
	}
	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, "", b, parser.ParseComments)
	if err != nil {
		gen.Error(err)
		return
	}
	genForward := func(tok token.Token, name string, expr ast.Expr) {
		// Don't import unexported symbols.
		r, _ := utf8.DecodeRuneInString(name)
		if !unicode.IsUpper(r) {
			return
		}
		// Don't import the FileDescriptor.
		if name == impFile.GoDescriptorIdent.GoName {
			return
		}
		// Don't import decls referencing a symbol defined in another package.
		// i.e., don't import decls which are themselves public imports:
		//
		//	type T = somepackage.T
		if _, ok := expr.(*ast.SelectorExpr); ok {
			return
		}
		g.P(tok, " ", name, " = ", impFile.GoImportPath.Ident(name))
	}
	g.P("// Symbols defined in public import of ", imp.Path())
	g.P()
	for _, decl := range astFile.Decls {
		switch decl := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					genForward(decl.Tok, spec.Name.Name, spec.Type)
				case *ast.ValueSpec:
					for i, name := range spec.Names {
						var expr ast.Expr
						if i < len(spec.Values) {
							expr = spec.Values[i]
						}
						genForward(decl.Tok, name.Name, expr)
					}
				case *ast.ImportSpec:
				default:
					panic(fmt.Sprintf("can't generate forward for spec type %T", spec))
				}
			}
		}
	}
	g.P()
}

func genEnum(gen *protogen.Plugin, g *protogen.GeneratedFile, f *fileInfo, enum *protogen.Enum) {
	// Enum type declaration.
	g.PrintLeadingComments(enum.Location)
	g.Annotate(enum.GoIdent.GoName, enum.Location)
	g.P("type ", enum.GoIdent, " int32",
		deprecationComment(enum.Desc.Options().(*descriptorpb.EnumOptions).GetDeprecated()))

	// Enum value constants.
	g.P("const (")
	for _, value := range enum.Values {
		g.PrintLeadingComments(value.Location)
		g.Annotate(value.GoIdent.GoName, value.Location)
		g.P(value.GoIdent, " ", enum.GoIdent, " = ", value.Desc.Number(),
			deprecationComment(value.Desc.Options().(*descriptorpb.EnumValueOptions).GetDeprecated()))
	}
	g.P(")")
	g.P()

	// Enum value mapping (number -> name).
	if generateEnumMapVars {
		nameMap := enum.GoIdent.GoName + "_name"
		g.P("var ", nameMap, " = map[int32]string{")
		generated := make(map[protoreflect.EnumNumber]bool)
		for _, value := range enum.Values {
			duplicate := ""
			if _, present := generated[value.Desc.Number()]; present {
				duplicate = "// Duplicate value: "
			}
			g.P(duplicate, value.Desc.Number(), ": ", strconv.Quote(string(value.Desc.Name())), ",")
			generated[value.Desc.Number()] = true
		}
		g.P("}")
		g.P()
	}

	// Enum value mapping (name -> number).
	if generateEnumMapVars {
		valueMap := enum.GoIdent.GoName + "_value"
		g.P("var ", valueMap, " = map[string]int32{")
		for _, value := range enum.Values {
			g.P(strconv.Quote(string(value.Desc.Name())), ": ", value.Desc.Number(), ",")
		}
		g.P("}")
		g.P()
	}

	// Enum method.
	//
	// NOTE: A pointer value is needed to represent presence in proto2.
	// Since a proto2 message can reference a proto3 enum, it is useful to
	// always generate this method (even on proto3 enums) to support that case.
	g.P("func (x ", enum.GoIdent, ") Enum() *", enum.GoIdent, " {")
	g.P("p := new(", enum.GoIdent, ")")
	g.P("*p = x")
	g.P("return p")
	g.P("}")
	g.P()

	// String method.
	g.P("func (x ", enum.GoIdent, ") String() string {")
	g.P("return ", protoimplPackage.Ident("X"), ".EnumStringOf(x.Descriptor(), ", protoreflectPackage.Ident("EnumNumber"), "(x))")
	g.P("}")
	g.P()

	genEnumReflectMethods(gen, g, f, enum)

	// UnmarshalJSON method.
	if generateEnumJSONMethods && enum.Desc.Syntax() == protoreflect.Proto2 {
		g.P("// Deprecated: Do not use.")
		g.P("func (x *", enum.GoIdent, ") UnmarshalJSON(b []byte) error {")
		g.P("num, err := ", protoimplPackage.Ident("X"), ".UnmarshalJSONEnum(x.Descriptor(), b)")
		g.P("if err != nil {")
		g.P("return err")
		g.P("}")
		g.P("*x = ", enum.GoIdent, "(num)")
		g.P("return nil")
		g.P("}")
		g.P()
	}

	// EnumDescriptor method.
	if generateRawDescMethods {
		var indexes []string
		for i := 1; i < len(enum.Location.Path); i += 2 {
			indexes = append(indexes, strconv.Itoa(int(enum.Location.Path[i])))
		}
		g.P("// Deprecated: Use ", enum.GoIdent, ".Descriptor instead.")
		g.P("func (", enum.GoIdent, ") EnumDescriptor() ([]byte, []int) {")
		g.P("return ", rawDescVarName(f), "GZIP(), []int{", strings.Join(indexes, ","), "}")
		g.P("}")
		g.P()
	}
}

// enumLegacyName returns the name used by the v1 proto package.
//
// Confusingly, this is <proto_package>.<go_ident>. This probably should have
// been the full name of the proto enum type instead, but changing it at this
// point would require thought.
func enumLegacyName(enum *protogen.Enum) string {
	fdesc := enum.Desc.ParentFile()
	if fdesc.Package() == "" {
		return enum.GoIdent.GoName
	}
	return string(fdesc.Package()) + "." + enum.GoIdent.GoName
}

func genMessage(gen *protogen.Plugin, g *protogen.GeneratedFile, f *fileInfo, message *protogen.Message) {
	if message.Desc.IsMapEntry() {
		return
	}

	// Message type declaration.
	hasComment := g.PrintLeadingComments(message.Location)
	if message.Desc.Options().(*descriptorpb.MessageOptions).GetDeprecated() {
		if hasComment {
			g.P("//")
		}
		g.P(deprecationComment(true))
	}
	g.Annotate(message.GoIdent.GoName, message.Location)
	g.P("type ", message.GoIdent, " struct {")
	genMessageFields(g, f, message)
	g.P("}")
	g.P()

	genDefaultConsts(g, message)
	genMessageMethods(gen, g, f, message)
	genOneofWrapperTypes(gen, g, f, message)
}

func genMessageFields(g *protogen.GeneratedFile, f *fileInfo, message *protogen.Message) {
	sf := f.allMessageFieldsByPtr[message]
	if generateMessageStateFields {
		g.P("state ", protoimplPackage.Ident("MessageState"))
		sf.append("state")
	}
	for _, field := range message.Fields {
		genMessageField(g, message, field, sf)
	}
	genMessageInternalFields(g, message, sf)
}

func genMessageField(g *protogen.GeneratedFile, message *protogen.Message, field *protogen.Field, sf *structFields) {
	if oneof := field.Oneof; oneof != nil {
		// It would be a bit simpler to iterate over the oneofs below,
		// but generating the field here keeps the contents of the Go
		// struct in the same order as the contents of the source
		// .proto file.
		if oneof.Fields[0] != field {
			return // only generate for first appearance
		}
		if g.PrintLeadingComments(oneof.Location) {
			g.P("//")
		}
		g.P("// Types that are valid to be assigned to ", oneof.GoName, ":")
		for _, field := range oneof.Fields {
			g.PrintLeadingComments(field.Location)
			g.P("//\t*", fieldOneofType(field))
		}
		g.Annotate(message.GoIdent.GoName+"."+oneof.GoName, oneof.Location)
		g.P(oneof.GoName, " ", oneofInterfaceName(oneof), " `protobuf_oneof:\"", oneof.Desc.Name(), "\"`")
		sf.append(oneof.GoName)
		return
	}
	g.PrintLeadingComments(field.Location)
	goType, pointer := fieldGoType(g, field)
	if pointer {
		goType = "*" + goType
	}
	tags := []string{
		fmt.Sprintf("protobuf:%q", fieldProtobufTag(field)),
		fmt.Sprintf("json:%q", fieldJSONTag(field)),
	}
	if field.Desc.IsMap() {
		key := field.Message.Fields[0]
		val := field.Message.Fields[1]
		tags = append(tags,
			fmt.Sprintf("protobuf_key:%q", fieldProtobufTag(key)),
			fmt.Sprintf("protobuf_val:%q", fieldProtobufTag(val)),
		)
	}

	name := field.GoName
	if field.Desc.IsWeak() {
		name = "XXX_weak_" + name
	}
	g.Annotate(message.GoIdent.GoName+"."+name, field.Location)
	g.P(name, " ", goType, " `", strings.Join(tags, " "), "`",
		deprecationComment(field.Desc.Options().(*descriptorpb.FieldOptions).GetDeprecated()))
	sf.append(field.GoName)
}

func genMessageInternalFields(g *protogen.GeneratedFile, message *protogen.Message, sf *structFields) {
	if generateNoUnkeyedLiteralFields {
		g.P("XXX_NoUnkeyedLiteral", " struct{} `json:\"-\"`")
		sf.append("XXX_NoUnkeyedLiteral")
	}
	if generateExportedSizeCacheFields {
		g.P("XXX_sizecache", " ", protoimplPackage.Ident("SizeCache"), " `json:\"-\"`")
		sf.append("XXX_sizecache")
	} else {
		g.P("sizeCache", " ", protoimplPackage.Ident("SizeCache"))
		sf.append("sizeCache")
	}
	if loadMessageAPIFlags(message).WeakMapField {
		g.P("XXX_weak", " ", protoimplPackage.Ident("WeakFields"), " `json:\"-\"`")
		sf.append("XXX_weak")
	}
	if generateExportedUnknownFields {
		g.P("XXX_unrecognized", " ", protoimplPackage.Ident("UnknownFields"), " `json:\"-\"`")
		sf.append("XXX_unrecognized")
	} else {
		g.P("unknownFields", " ", protoimplPackage.Ident("UnknownFields"))
		sf.append("unknownFields")
	}
	if message.Desc.ExtensionRanges().Len() > 0 {
		if generateExportedExtensionFields {
			g.P("XXX_InternalExtensions", " ", protoimplPackage.Ident("ExtensionFields"), " `json:\"-\"`")
			sf.append("XXX_InternalExtensions")
		} else {
			g.P("extensionFields", " ", protoimplPackage.Ident("ExtensionFields"))
			sf.append("extensionFields")
		}
	}
}

// genDefaultConsts generates consts and vars holding the default
// values of fields.
func genDefaultConsts(g *protogen.GeneratedFile, message *protogen.Message) {
	for _, field := range message.Fields {
		if !field.Desc.HasDefault() {
			continue
		}
		defVarName := "Default_" + message.GoIdent.GoName + "_" + field.GoName
		def := field.Desc.Default()
		switch field.Desc.Kind() {
		case protoreflect.StringKind:
			g.P("const ", defVarName, " string = ", strconv.Quote(def.String()))
		case protoreflect.BytesKind:
			g.P("var ", defVarName, " []byte = []byte(", strconv.Quote(string(def.Bytes())), ")")
		case protoreflect.EnumKind:
			evalueDesc := field.Desc.DefaultEnumValue()
			enum := field.Enum
			evalue := enum.Values[evalueDesc.Index()]
			g.P("const ", defVarName, " ", field.Enum.GoIdent, " = ", evalue.GoIdent)
		case protoreflect.FloatKind, protoreflect.DoubleKind:
			// Floating point numbers need extra handling for -Inf/Inf/NaN.
			f := field.Desc.Default().Float()
			goType := "float64"
			if field.Desc.Kind() == protoreflect.FloatKind {
				goType = "float32"
			}
			// funcCall returns a call to a function in the math package,
			// possibly converting the result to float32.
			funcCall := func(fn, param string) string {
				s := g.QualifiedGoIdent(mathPackage.Ident(fn)) + param
				if goType != "float64" {
					s = goType + "(" + s + ")"
				}
				return s
			}
			switch {
			case math.IsInf(f, -1):
				g.P("var ", defVarName, " ", goType, " = ", funcCall("Inf", "(-1)"))
			case math.IsInf(f, 1):
				g.P("var ", defVarName, " ", goType, " = ", funcCall("Inf", "(1)"))
			case math.IsNaN(f):
				g.P("var ", defVarName, " ", goType, " = ", funcCall("NaN", "()"))
			default:
				g.P("const ", defVarName, " ", goType, " = ", field.Desc.Default().Interface())
			}
		default:
			goType, _ := fieldGoType(g, field)
			g.P("const ", defVarName, " ", goType, " = ", def.Interface())
		}
	}
	g.P()
}

func genMessageMethods(gen *protogen.Plugin, g *protogen.GeneratedFile, f *fileInfo, message *protogen.Message) {
	genMessageBaseMethods(gen, g, f, message)
	genMessageGetterMethods(gen, g, f, message)
	genMessageSetterMethods(gen, g, f, message)
}

func genMessageBaseMethods(gen *protogen.Plugin, g *protogen.GeneratedFile, f *fileInfo, message *protogen.Message) {
	// Reset method.
	g.P("func (x *", message.GoIdent, ") Reset() {")
	g.P("*x = ", message.GoIdent, "{}")
	g.P("}")
	g.P()

	// String method.
	g.P("func (x *", message.GoIdent, ") String() string {")
	g.P("return ", protoimplPackage.Ident("X"), ".MessageStringOf(x)")
	g.P("}")
	g.P()

	// ProtoMessage method.
	g.P("func (*", message.GoIdent, ") ProtoMessage() {}")
	g.P()

	// ProtoReflect method.
	genMessageReflectMethods(gen, g, f, message)

	// Descriptor method.
	if generateRawDescMethods {
		var indexes []string
		for i := 1; i < len(message.Location.Path); i += 2 {
			indexes = append(indexes, strconv.Itoa(int(message.Location.Path[i])))
		}
		g.P("// Deprecated: Use ", message.GoIdent, ".ProtoReflect.Descriptor instead.")
		g.P("func (*", message.GoIdent, ") Descriptor() ([]byte, []int) {")
		g.P("return ", rawDescVarName(f), "GZIP(), []int{", strings.Join(indexes, ","), "}")
		g.P("}")
		g.P()
	}

	// ExtensionRangeArray method.
	if generateExtensionRangeMethods {
		if extranges := message.Desc.ExtensionRanges(); extranges.Len() > 0 {
			protoExtRange := protoifacePackage.Ident("ExtensionRangeV1")
			extRangeVar := "extRange_" + message.GoIdent.GoName
			g.P("var ", extRangeVar, " = []", protoExtRange, " {")
			for i := 0; i < extranges.Len(); i++ {
				r := extranges.Get(i)
				g.P("{Start:", r[0], ", End:", r[1]-1 /* inclusive */, "},")
			}
			g.P("}")
			g.P()
			g.P("// Deprecated: Use ", message.GoIdent, ".ProtoReflect.Descriptor.ExtensionRanges instead.")
			g.P("func (*", message.GoIdent, ") ExtensionRangeArray() []", protoExtRange, " {")
			g.P("return ", extRangeVar)
			g.P("}")
			g.P()
		}
	}

	// XXX_OneofWrappers method.
	if generateOneofWrapperMethods && len(message.Oneofs) > 0 {
		idx := f.allMessagesByPtr[message]
		typesVar := messageTypesVarName(f)
		g.P("// XXX_OneofWrappers is for the internal use of the proto package.")
		g.P("func (*", message.GoIdent.GoName, ") XXX_OneofWrappers() []interface{} {")
		g.P("return ", typesVar, "[", idx, "].OneofWrappers")
		g.P("}")
		g.P()
	}
}

func genMessageGetterMethods(gen *protogen.Plugin, g *protogen.GeneratedFile, f *fileInfo, message *protogen.Message) {
	for _, field := range message.Fields {
		// Getter for parent oneof.
		if oneof := field.Oneof; oneof != nil && oneof.Fields[0] == field {
			g.Annotate(message.GoIdent.GoName+".Get"+oneof.GoName, oneof.Location)
			g.P("func (m *", message.GoIdent.GoName, ") Get", oneof.GoName, "() ", oneofInterfaceName(oneof), " {")
			g.P("if m != nil {")
			g.P("return m.", oneof.GoName)
			g.P("}")
			g.P("return nil")
			g.P("}")
			g.P()
		}

		// Getter for message field.
		goType, pointer := fieldGoType(g, field)
		defaultValue := fieldDefaultValue(g, message, field)
		if field.Desc.Options().(*descriptorpb.FieldOptions).GetDeprecated() {
			g.P(deprecationComment(true))
		}
		g.Annotate(message.GoIdent.GoName+".Get"+field.GoName, field.Location)
		switch {
		case field.Desc.IsWeak():
			g.P("func (x *", message.GoIdent, ") Get", field.GoName, "() ", protoifacePackage.Ident("MessageV1"), "{")
			g.P("if x != nil {")
			g.P("v := x.XXX_weak[", field.Desc.Number(), "]")
			g.P("_ = x.XXX_weak_" + field.GoName) // for field tracking
			g.P("if v != nil {")
			g.P("return v")
			g.P("}")
			g.P("}")
			g.P("return ", protoimplPackage.Ident("X"), ".WeakNil(", strconv.Quote(string(field.Message.Desc.FullName())), ")")
			g.P("}")
		case field.Oneof != nil:
			g.P("func (x *", message.GoIdent, ") Get", field.GoName, "() ", goType, " {")
			g.P("if x, ok := x.Get", field.Oneof.GoName, "().(*", fieldOneofType(field), "); ok {")
			g.P("return x.", field.GoName)
			g.P("}")
			g.P("return ", defaultValue)
			g.P("}")
		default:
			g.P("func (x *", message.GoIdent, ") Get", field.GoName, "() ", goType, " {")
			if field.Desc.Syntax() == protoreflect.Proto3 || defaultValue == "nil" {
				g.P("if x != nil {")
			} else {
				g.P("if x != nil && x.", field.GoName, " != nil {")
			}
			star := ""
			if pointer {
				star = "*"
			}
			g.P("return ", star, " x.", field.GoName)
			g.P("}")
			g.P("return ", defaultValue)
			g.P("}")
		}
		g.P()
	}
}

func genMessageSetterMethods(gen *protogen.Plugin, g *protogen.GeneratedFile, f *fileInfo, message *protogen.Message) {
	for _, field := range message.Fields {
		if field.Desc.IsWeak() {
			g.Annotate(message.GoIdent.GoName+".Set"+field.GoName, field.Location)
			g.P("func (x *", message.GoIdent, ") Set", field.GoName, "(v ", protoifacePackage.Ident("MessageV1"), ") {")
			g.P("if x.XXX_weak == nil {")
			g.P("x.XXX_weak = make(", protoimplPackage.Ident("WeakFields"), ")")
			g.P("}")
			g.P("if v == nil {")
			g.P("delete(x.XXX_weak, ", field.Desc.Number(), ")")
			g.P("} else {")
			g.P("x.XXX_weak[", field.Desc.Number(), "] = v")
			g.P("x.XXX_weak_"+field.GoName, " = struct{}{}") // for field tracking
			g.P("}")
			g.P("}")
			g.P()
		}
	}
}

// fieldGoType returns the Go type used for a field.
//
// If it returns pointer=true, the struct field is a pointer to the type.
func fieldGoType(g *protogen.GeneratedFile, field *protogen.Field) (goType string, pointer bool) {
	if field.Desc.IsWeak() {
		return "struct{}", false
	}

	pointer = true
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		goType = "bool"
	case protoreflect.EnumKind:
		goType = g.QualifiedGoIdent(field.Enum.GoIdent)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		goType = "int32"
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		goType = "uint32"
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		goType = "int64"
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		goType = "uint64"
	case protoreflect.FloatKind:
		goType = "float32"
	case protoreflect.DoubleKind:
		goType = "float64"
	case protoreflect.StringKind:
		goType = "string"
	case protoreflect.BytesKind:
		goType = "[]byte"
		pointer = false
	case protoreflect.MessageKind, protoreflect.GroupKind:
		goType = "*" + g.QualifiedGoIdent(field.Message.GoIdent)
		pointer = false
	}
	switch {
	case field.Desc.IsList():
		goType = "[]" + goType
		pointer = false
	case field.Desc.IsMap():
		keyType, _ := fieldGoType(g, field.Message.Fields[0])
		valType, _ := fieldGoType(g, field.Message.Fields[1])
		return fmt.Sprintf("map[%v]%v", keyType, valType), false
	}

	// Extension fields always have pointer type, even when defined in a proto3 file.
	if field.Desc.Syntax() == protoreflect.Proto3 && !field.Desc.IsExtension() {
		pointer = false
	}
	return goType, pointer
}

func fieldProtobufTag(field *protogen.Field) string {
	var enumName string
	if field.Desc.Kind() == protoreflect.EnumKind {
		enumName = enumLegacyName(field.Enum)
	}
	return tag.Marshal(field.Desc, enumName)
}

func fieldDefaultValue(g *protogen.GeneratedFile, message *protogen.Message, field *protogen.Field) string {
	if field.Desc.IsList() {
		return "nil"
	}
	if field.Desc.HasDefault() {
		defVarName := "Default_" + message.GoIdent.GoName + "_" + field.GoName
		if field.Desc.Kind() == protoreflect.BytesKind {
			return "append([]byte(nil), " + defVarName + "...)"
		}
		return defVarName
	}
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		return "false"
	case protoreflect.StringKind:
		return `""`
	case protoreflect.MessageKind, protoreflect.GroupKind, protoreflect.BytesKind:
		return "nil"
	case protoreflect.EnumKind:
		return g.QualifiedGoIdent(field.Enum.Values[0].GoIdent)
	default:
		return "0"
	}
}

func fieldJSONTag(field *protogen.Field) string {
	return string(field.Desc.Name()) + ",omitempty"
}

func genExtensions(gen *protogen.Plugin, g *protogen.GeneratedFile, f *fileInfo) {
	if len(f.allExtensions) == 0 {
		return
	}

	g.P("var ", extDescsVarName(f), " = []", protoifacePackage.Ident("ExtensionDescV1"), "{")
	for _, extension := range f.allExtensions {
		g.P("{")
		g.P("ExtendedType: (*", extension.Extendee.GoIdent, ")(nil),")
		goType, pointer := fieldGoType(g, extension)
		if pointer {
			goType = "*" + goType
		}
		g.P("ExtensionType: (", goType, ")(nil),")
		g.P("Field: ", extension.Desc.Number(), ",")
		g.P("Name: ", strconv.Quote(string(extension.Desc.FullName())), ",")
		g.P("Tag: ", strconv.Quote(fieldProtobufTag(extension)), ",")
		g.P("Filename: ", strconv.Quote(f.Desc.Path()), ",")
		g.P("},")
	}
	g.P("}")

	g.P("var (")
	for i, extension := range f.allExtensions {
		ed := extension.Desc
		targetName := string(ed.ContainingMessage().FullName())
		typeName := ed.Kind().String()
		switch ed.Kind() {
		case protoreflect.EnumKind:
			typeName = string(ed.Enum().FullName())
		case protoreflect.MessageKind, protoreflect.GroupKind:
			typeName = string(ed.Message().FullName())
		}
		fieldName := string(ed.Name())
		g.P("// extend ", targetName, " { ", ed.Cardinality().String(), " ", typeName, " ", fieldName, " = ", ed.Number(), "; }")
		g.P(extensionVar(f.File, extension), " = &", extDescsVarName(f), "[", i, "]")
		g.P()
	}
	g.P(")")
}

// extensionVar returns the var holding the ExtensionDesc for an extension.
func extensionVar(f *protogen.File, extension *protogen.Extension) protogen.GoIdent {
	name := "E_"
	if extension.Parent != nil {
		name += extension.Parent.GoIdent.GoName + "_"
	}
	name += extension.GoName
	return f.GoImportPath.Ident(name)
}

// deprecationComment returns a standard deprecation comment if deprecated is true.
func deprecationComment(deprecated bool) string {
	if !deprecated {
		return ""
	}
	return "// Deprecated: Do not use."
}

// genOneofWrapperTypes generates the oneof wrapper types and
// associates the types with the parent message type.
func genOneofWrapperTypes(gen *protogen.Plugin, g *protogen.GeneratedFile, f *fileInfo, message *protogen.Message) {
	for _, oneof := range message.Oneofs {
		ifName := oneofInterfaceName(oneof)
		g.P("type ", ifName, " interface {")
		g.P(ifName, "()")
		g.P("}")
		g.P()
		for _, field := range oneof.Fields {
			name := fieldOneofType(field)
			g.Annotate(name.GoName, field.Location)
			g.Annotate(name.GoName+"."+field.GoName, field.Location)
			g.P("type ", name, " struct {")
			goType, _ := fieldGoType(g, field)
			tags := []string{
				fmt.Sprintf("protobuf:%q", fieldProtobufTag(field)),
			}
			g.P(field.GoName, " ", goType, " `", strings.Join(tags, " "), "`")
			g.P("}")
			g.P()
		}
		for _, field := range oneof.Fields {
			g.P("func (*", fieldOneofType(field), ") ", ifName, "() {}")
			g.P()
		}
	}
}

// oneofInterfaceName returns the name of the interface type implemented by
// the oneof field value types.
func oneofInterfaceName(oneof *protogen.Oneof) string {
	return fmt.Sprintf("is%s_%s", oneof.Parent.GoIdent.GoName, oneof.GoName)
}

// fieldOneofType returns the wrapper type used to represent a field in a oneof.
func fieldOneofType(field *protogen.Field) protogen.GoIdent {
	ident := protogen.GoIdent{
		GoImportPath: field.Parent.GoIdent.GoImportPath,
		GoName:       field.Parent.GoIdent.GoName + "_" + field.GoName,
	}
	// Check for collisions with nested messages or enums.
	//
	// This conflict resolution is incomplete: Among other things, it
	// does not consider collisions with other oneof field types.
	//
	// TODO: Consider dropping this entirely. Detecting conflicts and
	// producing an error is almost certainly better than permuting
	// field and type names in mostly unpredictable ways.
Loop:
	for {
		for _, message := range field.Parent.Messages {
			if message.GoIdent == ident {
				ident.GoName += "_"
				continue Loop
			}
		}
		for _, enum := range field.Parent.Enums {
			if enum.GoIdent == ident {
				ident.GoName += "_"
				continue Loop
			}
		}
		return ident
	}
}
