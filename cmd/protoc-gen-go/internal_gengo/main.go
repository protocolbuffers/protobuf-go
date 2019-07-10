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
	"runtime"
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

// GenerateVersionMarkers specifies whether to generate version markers.
var GenerateVersionMarkers = true

const (
	// generateEnumJSONMethods specifies whether to generate the UnmarshalJSON
	// method for proto2 enums.
	generateEnumJSONMethods = true

	// generateRawDescMethods specifies whether to generate EnumDescriptor and
	// Descriptor methods for enums and messages. These methods return the
	// GZIP'd contents of the raw file descriptor and the path from the root
	// to the given enum or message descriptor.
	generateRawDescMethods = true

	// generateExtensionRangeMethods specifies whether to generate the
	// ExtensionRangeArray method for messages that support extensions.
	generateExtensionRangeMethods = true

	// generateMessageStateFields specifies whether to generate an unexported
	// protoimpl.MessageState as the first field.
	generateMessageStateFields = true

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
	protoifacePackage   goImportPath = protogen.GoImportPath("google.golang.org/protobuf/runtime/protoiface")
	protoimplPackage    goImportPath = protogen.GoImportPath("google.golang.org/protobuf/runtime/protoimpl")
	protoreflectPackage goImportPath = protogen.GoImportPath("google.golang.org/protobuf/reflect/protoreflect")
	prototypePackage    goImportPath = protogen.GoImportPath("google.golang.org/protobuf/reflect/prototype")
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

	genStandaloneComments(g, f, fieldnum.FileDescriptorProto_Syntax)
	genGeneratedHeader(gen, g, f)
	genStandaloneComments(g, f, fieldnum.FileDescriptorProto_Package)
	g.P("package ", f.GoPackageName)
	g.P()

	// Emit a static check that enforces a minimum version of the proto package.
	if GenerateVersionMarkers {
		g.P("const (")
		g.P("// Verify that this generated code is sufficiently up-to-date.")
		g.P("_ = ", protoimplPackage.Ident("EnforceVersion"), "(", protoimpl.GenVersion, " - ", protoimplPackage.Ident("MinVersion"), ")")
		g.P("// Verify that runtime/protoimpl is sufficiently up-to-date.")
		g.P("_ = ", protoimplPackage.Ident("EnforceVersion"), "(", protoimplPackage.Ident("MaxVersion"), " - ", protoimpl.GenVersion, ")")
		g.P(")")
		g.P()
	}

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

// genStandaloneComments prints all leading comments for a FileDescriptorProto
// location identified by the field number n.
func genStandaloneComments(g *protogen.GeneratedFile, f *fileInfo, n int32) {
	for _, loc := range f.Proto.GetSourceCodeInfo().GetLocation() {
		if len(loc.Path) == 1 && loc.Path[0] == n {
			for _, s := range loc.GetLeadingDetachedComments() {
				g.P(protogen.Comments(s))
				g.P()
			}
			if s := loc.GetLeadingComments(); s != "" {
				g.P(protogen.Comments(s))
				g.P()
			}
		}
	}
}

func genGeneratedHeader(gen *protogen.Plugin, g *protogen.GeneratedFile, f *fileInfo) {
	g.P("// Code generated by protoc-gen-go. DO NOT EDIT.")

	if GenerateVersionMarkers {
		g.P("// versions:")
		protocGenGoVersion := protoimpl.VersionString()
		protocVersion := "(unknown)"
		if v := gen.Request.GetCompilerVersion(); v != nil {
			protocVersion = fmt.Sprintf("v%v.%v.%v", v.GetMajor(), v.GetMinor(), v.GetPatch())
		}
		goVersion := runtime.Version()
		if strings.HasPrefix(goVersion, "go") {
			goVersion = "v" + goVersion[len("go"):]
		}
		g.P("// \tprotoc-gen-go ", protocGenGoVersion)
		g.P("// \tprotoc        ", protocVersion)
		g.P("// \tgo            ", goVersion)
	}

	if f.Proto.GetOptions().GetDeprecated() {
		g.P("// ", f.Desc.Path(), " is a deprecated file.")
	} else {
		g.P("// source: ", f.Desc.Path())
	}
	g.P()
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
	g.P("// Symbols defined in public import of ", imp.Path(), ".")
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
	g.Annotate(enum.GoIdent.GoName, enum.Location)
	leadingComments := appendDeprecationSuffix(enum.Comments.Leading,
		enum.Desc.Options().(*descriptorpb.EnumOptions).GetDeprecated())
	g.P(leadingComments,
		"type ", enum.GoIdent, " int32")

	// Enum value constants.
	g.P("const (")
	for _, value := range enum.Values {
		g.Annotate(value.GoIdent.GoName, value.Location)
		leadingComments := appendDeprecationSuffix(value.Comments.Leading,
			value.Desc.Options().(*descriptorpb.EnumValueOptions).GetDeprecated())
		g.P(leadingComments,
			value.GoIdent, " ", enum.GoIdent, " = ", value.Desc.Number(),
			trailingComment(value.Comments.Trailing))
	}
	g.P(")")
	g.P()

	// Enum value maps.
	g.P("// Enum value maps for ", enum.GoIdent, ".")
	g.P("var (")
	g.P(enum.GoIdent.GoName+"_name", " = map[int32]string{")
	for _, value := range enum.Values {
		duplicate := ""
		if value.Desc != enum.Desc.Values().ByNumber(value.Desc.Number()) {
			duplicate = "// Duplicate value: "
		}
		g.P(duplicate, value.Desc.Number(), ": ", strconv.Quote(string(value.Desc.Name())), ",")
	}
	g.P("}")
	g.P(enum.GoIdent.GoName+"_value", " = map[string]int32{")
	for _, value := range enum.Values {
		g.P(strconv.Quote(string(value.Desc.Name())), ": ", value.Desc.Number(), ",")
	}
	g.P("}")
	g.P(")")
	g.P()

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
	g.Annotate(message.GoIdent.GoName, message.Location)
	leadingComments := appendDeprecationSuffix(message.Comments.Leading,
		message.Desc.Options().(*descriptorpb.MessageOptions).GetDeprecated())
	g.P(leadingComments,
		"type ", message.GoIdent, " struct {")
	genMessageFields(g, f, message)
	g.P("}")
	g.P()

	genDefaultDecls(g, f, message)
	genMessageMethods(gen, g, f, message)
	genOneofWrapperTypes(gen, g, f, message)
}

func genMessageFields(g *protogen.GeneratedFile, f *fileInfo, message *protogen.Message) {
	sf := f.allMessageFieldsByPtr[message]
	genMessageInternalFields(g, message, sf)
	for _, field := range message.Fields {
		genMessageField(g, f, message, field, sf)
	}
}

func genMessageInternalFields(g *protogen.GeneratedFile, message *protogen.Message, sf *structFields) {
	if generateMessageStateFields {
		g.P("state ", protoimplPackage.Ident("MessageState"))
		sf.append("state")
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
	if sf.count > 0 {
		g.P()
	}
}

func genMessageField(g *protogen.GeneratedFile, f *fileInfo, message *protogen.Message, field *protogen.Field, sf *structFields) {
	if oneof := field.Oneof; oneof != nil {
		// It would be a bit simpler to iterate over the oneofs below,
		// but generating the field here keeps the contents of the Go
		// struct in the same order as the contents of the source
		// .proto file.
		if oneof.Fields[0] != field {
			return // only generate for first appearance
		}

		g.Annotate(message.GoIdent.GoName+"."+oneof.GoName, oneof.Location)
		leadingComments := oneof.Comments.Leading
		if leadingComments != "" {
			leadingComments += "\n"
		}
		ss := []string{fmt.Sprintf(" Types that are assignable to %s:\n", oneof.GoName)}
		for _, field := range oneof.Fields {
			ss = append(ss, "\t*"+fieldOneofType(field).GoName+"\n")
		}
		leadingComments += protogen.Comments(strings.Join(ss, ""))
		g.P(leadingComments,
			oneof.GoName, " ", oneofInterfaceName(oneof), " `protobuf_oneof:\"", oneof.Desc.Name(), "\"`")
		sf.append(oneof.GoName)
		return
	}
	goType, pointer := fieldGoType(g, f, field)
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
	leadingComments := appendDeprecationSuffix(field.Comments.Leading,
		field.Desc.Options().(*descriptorpb.FieldOptions).GetDeprecated())
	g.P(leadingComments,
		name, " ", goType, " `", strings.Join(tags, " "), "`",
		trailingComment(field.Comments.Trailing))
	sf.append(field.GoName)
}

// genDefaultDecls generates consts and vars holding the default
// values of fields.
func genDefaultDecls(g *protogen.GeneratedFile, f *fileInfo, message *protogen.Message) {
	var consts, vars []string
	for _, field := range message.Fields {
		if !field.Desc.HasDefault() {
			continue
		}
		name := "Default_" + message.GoIdent.GoName + "_" + field.GoName
		goType, _ := fieldGoType(g, f, field)
		defVal := field.Desc.Default()
		switch field.Desc.Kind() {
		case protoreflect.StringKind:
			consts = append(consts, fmt.Sprintf("%s = %s(%q)", name, goType, defVal.String()))
		case protoreflect.BytesKind:
			vars = append(vars, fmt.Sprintf("%s = %s(%q)", name, goType, defVal.Bytes()))
		case protoreflect.EnumKind:
			idx := field.Desc.DefaultEnumValue().Index()
			val := field.Enum.Values[idx]
			consts = append(consts, fmt.Sprintf("%s = %s", name, g.QualifiedGoIdent(val.GoIdent)))
		case protoreflect.FloatKind, protoreflect.DoubleKind:
			if f := defVal.Float(); math.IsNaN(f) || math.IsInf(f, 0) {
				var fn, arg string
				switch f := defVal.Float(); {
				case math.IsInf(f, -1):
					fn, arg = g.QualifiedGoIdent(mathPackage.Ident("Inf")), "-1"
				case math.IsInf(f, +1):
					fn, arg = g.QualifiedGoIdent(mathPackage.Ident("Inf")), "+1"
				case math.IsNaN(f):
					fn, arg = g.QualifiedGoIdent(mathPackage.Ident("NaN")), ""
				}
				vars = append(vars, fmt.Sprintf("%s = %s(%s(%s))", name, goType, fn, arg))
			} else {
				consts = append(consts, fmt.Sprintf("%s = %s(%v)", name, goType, f))
			}
		default:
			consts = append(consts, fmt.Sprintf("%s = %s(%v)", name, goType, defVal.Interface()))
		}
	}
	if len(consts) > 0 {
		g.P("// Default values for ", message.GoIdent, " fields.")
		g.P("const (")
		for _, s := range consts {
			g.P(s)
		}
		g.P(")")
	}
	if len(vars) > 0 {
		g.P("// Default values for ", message.GoIdent, " fields.")
		g.P("var (")
		for _, s := range vars {
			g.P(s)
		}
		g.P(")")
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
		goType, pointer := fieldGoType(g, f, field)
		defaultValue := fieldDefaultValue(g, message, field)
		g.Annotate(message.GoIdent.GoName+".Get"+field.GoName, field.Location)
		leadingComments := appendDeprecationSuffix("",
			field.Desc.Options().(*descriptorpb.FieldOptions).GetDeprecated())
		switch {
		case field.Desc.IsWeak():
			g.P(leadingComments, "func (x *", message.GoIdent, ") Get", field.GoName, "() ", protoifacePackage.Ident("MessageV1"), "{")
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
			g.P(leadingComments, "func (x *", message.GoIdent, ") Get", field.GoName, "() ", goType, " {")
			g.P("if x, ok := x.Get", field.Oneof.GoName, "().(*", fieldOneofType(field), "); ok {")
			g.P("return x.", field.GoName)
			g.P("}")
			g.P("return ", defaultValue)
			g.P("}")
		default:
			g.P(leadingComments, "func (x *", message.GoIdent, ") Get", field.GoName, "() ", goType, " {")
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
			leadingComments := appendDeprecationSuffix("",
				field.Desc.Options().(*descriptorpb.FieldOptions).GetDeprecated())
			g.P(leadingComments, "func (x *", message.GoIdent, ") Set", field.GoName, "(v ", protoifacePackage.Ident("MessageV1"), ") {")
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
func fieldGoType(g *protogen.GeneratedFile, f *fileInfo, field *protogen.Field) (goType string, pointer bool) {
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
		keyType, _ := fieldGoType(g, f, field.Message.Fields[0])
		valType, _ := fieldGoType(g, f, field.Message.Fields[1])
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
		goType, pointer := fieldGoType(g, f, extension)
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
	g.P()

	// Group extensions by the target message.
	var orderedTargets []protogen.GoIdent
	allExtensionsByTarget := make(map[protogen.GoIdent][]*protogen.Extension)
	allExtensionsByPtr := make(map[*protogen.Extension]int)
	for i, extension := range f.allExtensions {
		target := extension.Extendee.GoIdent
		if len(allExtensionsByTarget[target]) == 0 {
			orderedTargets = append(orderedTargets, target)
		}
		allExtensionsByTarget[target] = append(allExtensionsByTarget[target], extension)
		allExtensionsByPtr[extension] = i
	}
	for _, target := range orderedTargets {
		g.P("// Extension fields to ", target, ".")
		g.P("var (")
		for _, extension := range allExtensionsByTarget[target] {
			xd := extension.Desc
			typeName := xd.Kind().String()
			switch xd.Kind() {
			case protoreflect.EnumKind:
				typeName = string(xd.Enum().FullName())
			case protoreflect.MessageKind, protoreflect.GroupKind:
				typeName = string(xd.Message().FullName())
			}
			fieldName := string(xd.Name())

			leadingComments := extension.Comments.Leading
			if leadingComments != "" {
				leadingComments += "\n"
			}
			leadingComments += protogen.Comments(fmt.Sprintf(" %v %v %v = %v;\n",
				xd.Cardinality(), typeName, fieldName, xd.Number()))
			leadingComments = appendDeprecationSuffix(leadingComments,
				extension.Desc.Options().(*descriptorpb.FieldOptions).GetDeprecated())
			g.P(leadingComments,
				extensionVar(f.File, extension), " = &", extDescsVarName(f), "[", allExtensionsByPtr[extension], "]",
				trailingComment(extension.Comments.Trailing))
		}
		g.P(")")
		g.P()
	}
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
			goType, _ := fieldGoType(g, f, field)
			tags := []string{
				fmt.Sprintf("protobuf:%q", fieldProtobufTag(field)),
			}
			leadingComments := appendDeprecationSuffix(field.Comments.Leading,
				field.Desc.Options().(*descriptorpb.FieldOptions).GetDeprecated())
			g.P(leadingComments,
				field.GoName, " ", goType, " `", strings.Join(tags, " "), "`",
				trailingComment(field.Comments.Trailing))
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

// appendDeprecationSuffix optionally appends a deprecation notice as a suffix.
func appendDeprecationSuffix(prefix protogen.Comments, deprecated bool) protogen.Comments {
	if !deprecated {
		return prefix
	}
	if prefix != "" {
		prefix += "\n"
	}
	return prefix + " Deprecated: Do not use.\n"
}

// trailingComment is like protogen.Comments, but lacks a trailing newline.
type trailingComment protogen.Comments

func (c trailingComment) String() string {
	s := strings.TrimSuffix(protogen.Comments(c).String(), "\n")
	if strings.Contains(s, "\n") {
		// We don't support multi-lined trailing comments as it is unclear
		// how to best render them in the generated code.
		return ""
	}
	return s
}
