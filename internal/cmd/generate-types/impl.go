// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"text/template"
)

func generateImplCodec() string {
	return mustExecute(implCodecTemplate, ProtoKinds)
}

var implCodecTemplate = template.Must(template.New("").Parse(`
{{- /*
  IsZero is an expression testing if 'v' is the zero value.
*/ -}}
{{- define "IsZero" -}}
{{if eq .WireType "Bytes" -}}
len(v) == 0
{{- else if or (eq .Name "Double") (eq .Name "Float") -}}
v == 0 && !math.Signbit(float64(v))
{{- else -}}
v == {{.GoType.Zero}}
{{- end -}}
{{- end -}}

{{- /*
  Size is an expression computing the size of 'v'.
*/ -}}
{{- define "Size" -}}
{{- if .WireType.ConstSize -}}
wire.Size{{.WireType}}()
{{- else if eq .WireType "Bytes" -}}
wire.SizeBytes(len({{.FromGoType}}))
{{- else -}}
wire.Size{{.WireType}}({{.FromGoType}})
{{- end -}}
{{- end -}}

{{- define "SizeValue" -}}
{{- if .WireType.ConstSize -}}
wire.Size{{.WireType}}()
{{- else if eq .WireType "Bytes" -}}
wire.SizeBytes(len({{.FromValue}}))
{{- else -}}
wire.Size{{.WireType}}({{.FromValue}})
{{- end -}}
{{- end -}}

{{- /*
  Append is a set of statements appending 'v' to 'b'.
*/ -}}
{{- define "Append" -}}
{{- if eq .Name "String" -}}
b = wire.AppendString(b, {{.FromGoType}})
{{- else -}}
b = wire.Append{{.WireType}}(b, {{.FromGoType}})
{{- end -}}
{{- end -}}

{{- define "AppendValue" -}}
{{- if eq .Name "String" -}}
b = wire.AppendString(b, {{.FromValue}})
{{- else -}}
b = wire.Append{{.WireType}}(b, {{.FromValue}})
{{- end -}}
{{- end -}}

{{- define "Consume" -}}
{{- if eq .Name "String" -}}
wire.ConsumeString(b)
{{- else -}}
wire.Consume{{.WireType}}(b)
{{- end -}}
{{- end -}}

{{- range .}}

{{- if .FromGoType }}
// size{{.Name}} returns the size of wire encoding a {{.GoType}} pointer as a {{.Name}}.
func size{{.Name}}(p pointer, tagsize int, _ marshalOptions) (size int) {
	{{if not .WireType.ConstSize -}}
	v := *p.{{.GoType.PointerMethod}}()
	{{- end}}
	return tagsize + {{template "Size" .}}
}

// append{{.Name}} wire encodes a {{.GoType}} pointer as a {{.Name}}.
func append{{.Name}}(b []byte, p pointer, wiretag uint64, _ marshalOptions) ([]byte, error) {
	v := *p.{{.GoType.PointerMethod}}()
	b = wire.AppendVarint(b, wiretag)
	{{template "Append" .}}
	return b, nil
}

// consume{{.Name}} wire decodes a {{.GoType}} pointer as a {{.Name}}.
func consume{{.Name}}(b []byte, p pointer, wtyp wire.Type, _ unmarshalOptions) (n int, err error) {
	if wtyp != {{.WireType.Expr}} {
		return 0, errUnknown
	}
	v, n := {{template "Consume" .}}
	if n < 0 {
		return 0, wire.ParseError(n)
	}
	*p.{{.GoType.PointerMethod}}() = {{.ToGoType}}
	return n, nil
}

var coder{{.Name}} = pointerCoderFuncs{
	size:      size{{.Name}},
	marshal:   append{{.Name}},
	unmarshal: consume{{.Name}},
}

{{if or (eq .Name "Bytes") (eq .Name "String")}}
// append{{.Name}}ValidateUTF8 wire encodes a {{.GoType}} pointer as a {{.Name}}.
func append{{.Name}}ValidateUTF8(b []byte, p pointer, wiretag uint64, _ marshalOptions) ([]byte, error) {
	v := *p.{{.GoType.PointerMethod}}()
	b = wire.AppendVarint(b, wiretag)
	{{template "Append" .}}
	if !utf8.Valid{{if eq .Name "String"}}String{{end}}(v) {
		return b, errInvalidUTF8{}
	}
	return b, nil
}

// consume{{.Name}}ValidateUTF8 wire decodes a {{.GoType}} pointer as a {{.Name}}.
func consume{{.Name}}ValidateUTF8(b []byte, p pointer, wtyp wire.Type, _ unmarshalOptions) (n int, err error) {
	if wtyp != {{.WireType.Expr}} {
		return 0, errUnknown
	}
	v, n := {{template "Consume" .}}
	if n < 0 {
		return 0, wire.ParseError(n)
	}
	if !utf8.Valid{{if eq .Name "String"}}String{{end}}(v) {
		return 0, errInvalidUTF8{}
	}
	*p.{{.GoType.PointerMethod}}() = {{.ToGoType}}
	return n, nil
}

var coder{{.Name}}ValidateUTF8 = pointerCoderFuncs{
	size:      size{{.Name}},
	marshal:   append{{.Name}}ValidateUTF8,
	unmarshal: consume{{.Name}}ValidateUTF8,
}
{{end}}

// size{{.Name}}NoZero returns the size of wire encoding a {{.GoType}} pointer as a {{.Name}}.
// The zero value is not encoded.
func size{{.Name}}NoZero(p pointer, tagsize int, _ marshalOptions) (size int) {
	v := *p.{{.GoType.PointerMethod}}()
	if {{template "IsZero" .}} {
		return 0
	}
	return tagsize + {{template "Size" .}}
}

// append{{.Name}}NoZero wire encodes a {{.GoType}} pointer as a {{.Name}}.
// The zero value is not encoded.
func append{{.Name}}NoZero(b []byte, p pointer, wiretag uint64, _ marshalOptions) ([]byte, error) {
	v := *p.{{.GoType.PointerMethod}}()
	if {{template "IsZero" .}} {
		return b, nil
	}
	b = wire.AppendVarint(b, wiretag)
	{{template "Append" .}}
	return b, nil
}

{{if .ToGoTypeNoZero}}
// consume{{.Name}}NoZero wire decodes a {{.GoType}} pointer as a {{.Name}}.
// The zero value is not decoded.
func consume{{.Name}}NoZero(b []byte, p pointer, wtyp wire.Type, _ unmarshalOptions) (n int, err error) {
	if wtyp != {{.WireType.Expr}} {
		return 0, errUnknown
	}
	v, n := {{template "Consume" .}}
	if n < 0 {
		return 0, wire.ParseError(n)
	}
	*p.{{.GoType.PointerMethod}}() = {{.ToGoTypeNoZero}}
	return n, nil
}
{{end}}

var coder{{.Name}}NoZero = pointerCoderFuncs{
	size:      size{{.Name}}NoZero,
	marshal:   append{{.Name}}NoZero,
	unmarshal: consume{{.Name}}{{if .ToGoTypeNoZero}}NoZero{{end}},
}

{{if or (eq .Name "Bytes") (eq .Name "String")}}
// append{{.Name}}NoZeroValidateUTF8 wire encodes a {{.GoType}} pointer as a {{.Name}}.
// The zero value is not encoded.
func append{{.Name}}NoZeroValidateUTF8(b []byte, p pointer, wiretag uint64, _ marshalOptions) ([]byte, error) {
	v := *p.{{.GoType.PointerMethod}}()
	if {{template "IsZero" .}} {
		return b, nil
	}
	b = wire.AppendVarint(b, wiretag)
	{{template "Append" .}}
	if !utf8.Valid{{if eq .Name "String"}}String{{end}}(v) {
		return b, errInvalidUTF8{}
	}
	return b, nil
}

{{if .ToGoTypeNoZero}}
// consume{{.Name}}NoZeroValidateUTF8 wire decodes a {{.GoType}} pointer as a {{.Name}}.
func consume{{.Name}}NoZeroValidateUTF8(b []byte, p pointer, wtyp wire.Type, _ unmarshalOptions) (n int, err error) {
	if wtyp != {{.WireType.Expr}} {
		return 0, errUnknown
	}
	v, n := {{template "Consume" .}}
	if n < 0 {
		return 0, wire.ParseError(n)
	}
	if !utf8.Valid{{if eq .Name "String"}}String{{end}}(v) {
		return 0, errInvalidUTF8{}
	}
	*p.{{.GoType.PointerMethod}}() = {{.ToGoTypeNoZero}}
	return n, nil
}
{{end}}

var coder{{.Name}}NoZeroValidateUTF8 = pointerCoderFuncs{
	size:      size{{.Name}}NoZero,
	marshal:   append{{.Name}}NoZeroValidateUTF8,
	unmarshal: consume{{.Name}}{{if .ToGoTypeNoZero}}NoZero{{end}}ValidateUTF8,
}
{{end}}

{{- if not .NoPointer}}
// size{{.Name}}Ptr returns the size of wire encoding a *{{.GoType}} pointer as a {{.Name}}.
// It panics if the pointer is nil.
func size{{.Name}}Ptr(p pointer, tagsize int, _ marshalOptions) (size int) {
	{{if not .WireType.ConstSize -}}
	v := **p.{{.GoType.PointerMethod}}Ptr()
	{{end -}}
	return tagsize + {{template "Size" .}}
}

// append{{.Name}}Ptr wire encodes a *{{.GoType}} pointer as a {{.Name}}.
// It panics if the pointer is nil.
func append{{.Name}}Ptr(b []byte, p pointer, wiretag uint64, _ marshalOptions) ([]byte, error) {
	v := **p.{{.GoType.PointerMethod}}Ptr()
	b = wire.AppendVarint(b, wiretag)
	{{template "Append" .}}
	return b, nil
}

// consume{{.Name}}Ptr wire decodes a *{{.GoType}} pointer as a {{.Name}}.
func consume{{.Name}}Ptr(b []byte, p pointer, wtyp wire.Type, _ unmarshalOptions) (n int, err error) {
	if wtyp != {{.WireType.Expr}} {
		return 0, errUnknown
	}
	v, n := {{template "Consume" .}}
	if n < 0 {
		return 0, wire.ParseError(n)
	}
	vp := p.{{.GoType.PointerMethod}}Ptr()
	if *vp == nil {
		*vp = new({{.GoType}})
	}
	**vp = {{.ToGoType}}
	return n, nil
}

var coder{{.Name}}Ptr = pointerCoderFuncs{
	size:      size{{.Name}}Ptr,
	marshal:   append{{.Name}}Ptr,
	unmarshal: consume{{.Name}}Ptr,
}
{{end}}

// size{{.Name}}Slice returns the size of wire encoding a []{{.GoType}} pointer as a repeated {{.Name}}.
func size{{.Name}}Slice(p pointer, tagsize int, _ marshalOptions) (size int) {
	s := *p.{{.GoType.PointerMethod}}Slice()
	{{if .WireType.ConstSize -}}
	size = len(s) * (tagsize + {{template "Size" .}})
	{{- else -}}
	for _, v := range s {
		size += tagsize + {{template "Size" .}}
	}
	{{- end}}
	return size
}

// append{{.Name}}Slice encodes a []{{.GoType}} pointer as a repeated {{.Name}}.
func append{{.Name}}Slice(b []byte, p pointer, wiretag uint64, _ marshalOptions) ([]byte, error) {
	s := *p.{{.GoType.PointerMethod}}Slice()
	for _, v := range s {
		b = wire.AppendVarint(b, wiretag)
		{{template "Append" .}}
	}
	return b, nil
}

// consume{{.Name}}Slice wire decodes a []{{.GoType}} pointer as a repeated {{.Name}}.
func consume{{.Name}}Slice(b []byte, p pointer, wtyp wire.Type, _ unmarshalOptions) (n int, err error) {
	sp := p.{{.GoType.PointerMethod}}Slice()
	{{- if .WireType.Packable}}
	if wtyp == wire.BytesType {
		s := *sp
		b, n = wire.ConsumeBytes(b)
		if n < 0 {
			return 0, wire.ParseError(n)
		}
		for len(b) > 0 {
			v, n := {{template "Consume" .}}
			if n < 0 {
				return 0, wire.ParseError(n)
			}
			s = append(s, {{.ToGoType}})
			b = b[n:]
		}
		*sp = s
		return n, nil
	}
	{{- end}}
	if wtyp != {{.WireType.Expr}} {
		return 0, errUnknown
	}
	v, n := {{template "Consume" .}}
	if n < 0 {
		return 0, wire.ParseError(n)
	}
	*sp = append(*sp, {{.ToGoType}})
	return n, nil
}

var coder{{.Name}}Slice = pointerCoderFuncs{
	size:      size{{.Name}}Slice,
	marshal:   append{{.Name}}Slice,
	unmarshal: consume{{.Name}}Slice,
}

{{if or (eq .Name "Bytes") (eq .Name "String")}}
// append{{.Name}}SliceValidateUTF8 encodes a []{{.GoType}} pointer as a repeated {{.Name}}.
func append{{.Name}}SliceValidateUTF8(b []byte, p pointer, wiretag uint64, _ marshalOptions) ([]byte, error) {
	s := *p.{{.GoType.PointerMethod}}Slice()
	for _, v := range s {
		b = wire.AppendVarint(b, wiretag)
		{{template "Append" .}}
		if !utf8.Valid{{if eq .Name "String"}}String{{end}}(v) {
			return b, errInvalidUTF8{}
		}
	}
	return b, nil
}

// consume{{.Name}}SliceValidateUTF8 wire decodes a []{{.GoType}} pointer as a repeated {{.Name}}.
func consume{{.Name}}SliceValidateUTF8(b []byte, p pointer, wtyp wire.Type, _ unmarshalOptions) (n int, err error) {
	sp := p.{{.GoType.PointerMethod}}Slice()
	if wtyp != {{.WireType.Expr}} {
		return 0, errUnknown
	}
	v, n := {{template "Consume" .}}
	if n < 0 {
		return 0, wire.ParseError(n)
	}
	if !utf8.Valid{{if eq .Name "String"}}String{{end}}(v) {
		return 0, errInvalidUTF8{}
	}
	*sp = append(*sp, {{.ToGoType}})
	return n, nil
}

var coder{{.Name}}SliceValidateUTF8 = pointerCoderFuncs{
	size:      size{{.Name}}Slice,
	marshal:   append{{.Name}}SliceValidateUTF8,
	unmarshal: consume{{.Name}}SliceValidateUTF8,
}
{{end}}

{{if or (eq .WireType "Varint") (eq .WireType "Fixed32") (eq .WireType "Fixed64")}}
// size{{.Name}}PackedSlice returns the size of wire encoding a []{{.GoType}} pointer as a packed repeated {{.Name}}.
func size{{.Name}}PackedSlice(p pointer, tagsize int, _ marshalOptions) (size int) {
	s := *p.{{.GoType.PointerMethod}}Slice()
	if len(s) == 0 {
		return 0
	}
	{{if .WireType.ConstSize -}}
	n := len(s) * {{template "Size" .}}
	{{- else -}}
	n := 0
	for _, v := range s {
		n += {{template "Size" .}}
	}
	{{- end}}
	return tagsize + wire.SizeBytes(n)
}

// append{{.Name}}PackedSlice encodes a []{{.GoType}} pointer as a packed repeated {{.Name}}.
func append{{.Name}}PackedSlice(b []byte, p pointer, wiretag uint64, _ marshalOptions) ([]byte, error) {
	s := *p.{{.GoType.PointerMethod}}Slice()
	if len(s) == 0 {
		return b, nil
	}
	b = wire.AppendVarint(b, wiretag)
	{{if .WireType.ConstSize -}}
	n := len(s) * {{template "Size" .}}
	{{- else -}}
	n := 0
	for _, v := range s {
		n += {{template "Size" .}}
	}
	{{- end}}
	b = wire.AppendVarint(b, uint64(n))
	for _, v := range s {
		{{template "Append" .}}
	}
	return b, nil
}

var coder{{.Name}}PackedSlice = pointerCoderFuncs{
	size:      size{{.Name}}PackedSlice,
	marshal:   append{{.Name}}PackedSlice,
	unmarshal: consume{{.Name}}Slice,
}
{{end}}

{{end -}}

{{- if not .NoValueCodec}}
// size{{.Name}}Value returns the size of wire encoding a {{.GoType}} value as a {{.Name}}.
func size{{.Name}}Value(v protoreflect.Value, tagsize int, _ marshalOptions) int {
	return tagsize + {{template "SizeValue" .}}
}

// append{{.Name}}Value encodes a {{.GoType}} value as a {{.Name}}.
func append{{.Name}}Value(b []byte, v protoreflect.Value, wiretag uint64, _ marshalOptions) ([]byte, error) {
	b = wire.AppendVarint(b, wiretag)
	{{template "AppendValue" .}}
	return b, nil
}

// consume{{.Name}}Value decodes a {{.GoType}} value as a {{.Name}}.
func consume{{.Name}}Value(b []byte, _ protoreflect.Value, _ wire.Number, wtyp wire.Type, _ unmarshalOptions) (protoreflect.Value, int, error) {
	if wtyp != {{.WireType.Expr}} {
		return protoreflect.Value{}, 0, errUnknown
	}
	v, n := {{template "Consume" .}}
	if n < 0 {
		return protoreflect.Value{}, 0, wire.ParseError(n)
	}
	return {{.ToValue}}, n, nil
}

var coder{{.Name}}Value = valueCoderFuncs{
	size:    size{{.Name}}Value,
	marshal: append{{.Name}}Value,
	unmarshal: consume{{.Name}}Value,
}

{{if or (eq .Name "Bytes") (eq .Name "String")}}
// append{{.Name}}ValueValidateUTF8 encodes a {{.GoType}} value as a {{.Name}}.
func append{{.Name}}ValueValidateUTF8(b []byte, v protoreflect.Value, wiretag uint64, _ marshalOptions) ([]byte, error) {
	b = wire.AppendVarint(b, wiretag)
	{{template "AppendValue" .}}
	if !utf8.Valid{{if eq .Name "String"}}String{{end}}({{.FromValue}}) {
		return b, errInvalidUTF8{}
	}
	return b, nil
}

// consume{{.Name}}ValueValidateUTF8 decodes a {{.GoType}} value as a {{.Name}}.
func consume{{.Name}}ValueValidateUTF8(b []byte, _ protoreflect.Value, _ wire.Number, wtyp wire.Type, _ unmarshalOptions) (protoreflect.Value, int, error) {
	if wtyp != {{.WireType.Expr}} {
		return protoreflect.Value{}, 0, errUnknown
	}
	v, n := {{template "Consume" .}}
	if n < 0 {
		return protoreflect.Value{}, 0, wire.ParseError(n)
	}
	if !utf8.Valid{{if eq .Name "String"}}String{{end}}(v) {
		return protoreflect.Value{}, 0, errInvalidUTF8{}
	}
	return {{.ToValue}}, n, nil
}

var coder{{.Name}}ValueValidateUTF8 = valueCoderFuncs{
	size:      size{{.Name}}Value,
	marshal:   append{{.Name}}ValueValidateUTF8,
	unmarshal: consume{{.Name}}ValueValidateUTF8,
}
{{end}}

// size{{.Name}}SliceValue returns the size of wire encoding a []{{.GoType}} value as a repeated {{.Name}}.
func size{{.Name}}SliceValue(listv protoreflect.Value, tagsize int, _ marshalOptions) (size int) {
	list := listv.List()
	{{if .WireType.ConstSize -}}
	size = list.Len() * (tagsize + {{template "SizeValue" .}})
	{{- else -}}
	for i, llen := 0, list.Len(); i < llen; i++ {
		v := list.Get(i)
		size += tagsize + {{template "SizeValue" .}}
	}
	{{- end}}
	return size
}

// append{{.Name}}SliceValue encodes a []{{.GoType}} value as a repeated {{.Name}}.
func append{{.Name}}SliceValue(b []byte, listv protoreflect.Value, wiretag uint64, _ marshalOptions) ([]byte, error) {
	list := listv.List()
	for i, llen := 0, list.Len(); i < llen; i++ {
		v := list.Get(i)
		b = wire.AppendVarint(b, wiretag)
		{{template "AppendValue" .}}
	}
	return b, nil
}

// consume{{.Name}}SliceValue wire decodes a []{{.GoType}} value as a repeated {{.Name}}.
func consume{{.Name}}SliceValue(b []byte, listv protoreflect.Value, _ wire.Number, wtyp wire.Type, _ unmarshalOptions) (_ protoreflect.Value, n int, err error) {
	list := listv.List()
	{{- if .WireType.Packable}}
	if wtyp == wire.BytesType {
		b, n = wire.ConsumeBytes(b)
		if n < 0 {
			return protoreflect.Value{}, 0, wire.ParseError(n)
		}
		for len(b) > 0 {
			v, n := {{template "Consume" .}}
			if n < 0 {
				return protoreflect.Value{}, 0, wire.ParseError(n)
			}
			list.Append({{.ToValue}})
			b = b[n:]
		}
		return listv, n, nil
	}
	{{- end}}
	if wtyp != {{.WireType.Expr}} {
		return protoreflect.Value{}, 0, errUnknown
	}
	v, n := {{template "Consume" .}}
	if n < 0 {
		return protoreflect.Value{}, 0, wire.ParseError(n)
	}
	list.Append({{.ToValue}})
	return listv, n, nil
}

var coder{{.Name}}SliceValue = valueCoderFuncs{
	size:      size{{.Name}}SliceValue,
	marshal:   append{{.Name}}SliceValue,
	unmarshal: consume{{.Name}}SliceValue,
}

{{if or (eq .WireType "Varint") (eq .WireType "Fixed32") (eq .WireType "Fixed64")}}
// size{{.Name}}PackedSliceValue returns the size of wire encoding a []{{.GoType}} value as a packed repeated {{.Name}}.
func size{{.Name}}PackedSliceValue(listv protoreflect.Value, tagsize int, _ marshalOptions) (size int) {
	list := listv.List()
	{{if .WireType.ConstSize -}}
	n := list.Len() * {{template "SizeValue" .}}
	{{- else -}}
	n := 0
	for i, llen := 0, list.Len(); i < llen; i++ {
		v := list.Get(i)
		n += {{template "SizeValue" .}}
	}
	{{- end}}
	return tagsize + wire.SizeBytes(n)
}

// append{{.Name}}PackedSliceValue encodes a []{{.GoType}} value as a packed repeated {{.Name}}.
func append{{.Name}}PackedSliceValue(b []byte, listv protoreflect.Value, wiretag uint64, _ marshalOptions) ([]byte, error) {
	list := listv.List()
	llen := list.Len()
	if llen == 0 {
		return b, nil
	}
	b = wire.AppendVarint(b, wiretag)
	{{if .WireType.ConstSize -}}
	n := llen * {{template "SizeValue" .}}
	{{- else -}}
	n := 0
	for i := 0; i < llen; i++ {
		v := list.Get(i)
		n += {{template "SizeValue" .}}
	}
	{{- end}}
	b = wire.AppendVarint(b, uint64(n))
	for i := 0; i < llen; i++ {
		v := list.Get(i)
		{{template "AppendValue" .}}
	}
	return b, nil
}

var coder{{.Name}}PackedSliceValue = valueCoderFuncs{
	size:      size{{.Name}}PackedSliceValue,
	marshal:   append{{.Name}}PackedSliceValue,
	unmarshal: consume{{.Name}}SliceValue,
}
{{end}}

{{- end}}{{/* if not .NoValueCodec */}}

{{end -}}

// We append to an empty array rather than a nil []byte to get non-nil zero-length byte slices.
var emptyBuf [0]byte

var wireTypes = map[protoreflect.Kind]wire.Type{
{{range . -}}
	protoreflect.{{.Name}}Kind: {{.WireType.Expr}},
{{end}}
}
`))

func generateImplMessage() string {
	return mustExecute(implMessageTemplate, []string{"messageState", "messageReflectWrapper"})
}

var implMessageTemplate = template.Must(template.New("").Parse(`
{{range . -}}
func (m *{{.}}) Descriptor() protoreflect.MessageDescriptor {
	return m.messageInfo().Desc
}
func (m *{{.}}) Type() protoreflect.MessageType {
	return m.messageInfo()
}
func (m *{{.}}) New() protoreflect.Message {
	return m.messageInfo().New()
}
func (m *{{.}}) Interface() protoreflect.ProtoMessage {
	{{if eq . "messageState" -}}
	return m.ProtoUnwrap().(protoreflect.ProtoMessage)
	{{- else -}}
	if m, ok := m.ProtoUnwrap().(protoreflect.ProtoMessage); ok {
		return m
	}
	return (*messageIfaceWrapper)(m)
	{{- end -}}
}
func (m *{{.}}) ProtoUnwrap() interface{} {
	return m.pointer().AsIfaceOf(m.messageInfo().GoReflectType.Elem())
}
func (m *{{.}}) ProtoMethods() *protoiface.Methods {
	m.messageInfo().init()
	return &m.messageInfo().methods
}

// ProtoMessageInfo is a pseudo-internal API for allowing the v1 code
// to be able to retrieve a v2 MessageInfo struct.
//
// WARNING: This method is exempt from the compatibility promise and
// may be removed in the future without warning.
func (m *{{.}}) ProtoMessageInfo() *MessageInfo {
	return m.messageInfo()
}

func (m *{{.}}) Range(f func(protoreflect.FieldDescriptor, protoreflect.Value) bool) {
	m.messageInfo().init()
	for _, fi := range m.messageInfo().fields {
		if fi.has(m.pointer()) {
			if !f(fi.fieldDesc, fi.get(m.pointer())) {
				return
			}
		}
	}
	m.messageInfo().extensionMap(m.pointer()).Range(f)
}
func (m *{{.}}) Has(fd protoreflect.FieldDescriptor) bool {
	m.messageInfo().init()
	if fi, xt := m.messageInfo().checkField(fd); fi != nil {
		return fi.has(m.pointer())
	} else {
		return m.messageInfo().extensionMap(m.pointer()).Has(xt)
	}
}
func (m *{{.}}) Clear(fd protoreflect.FieldDescriptor) {
	m.messageInfo().init()
	if fi, xt := m.messageInfo().checkField(fd); fi != nil {
		fi.clear(m.pointer())
	} else {
		m.messageInfo().extensionMap(m.pointer()).Clear(xt)
	}
}
func (m *{{.}}) Get(fd protoreflect.FieldDescriptor) protoreflect.Value {
	m.messageInfo().init()
	if fi, xt := m.messageInfo().checkField(fd); fi != nil {
		return fi.get(m.pointer())
	} else {
		return m.messageInfo().extensionMap(m.pointer()).Get(xt)
	}
}
func (m *{{.}}) Set(fd protoreflect.FieldDescriptor, v protoreflect.Value) {
	m.messageInfo().init()
	if fi, xt := m.messageInfo().checkField(fd); fi != nil {
		fi.set(m.pointer(), v)
	} else {
		m.messageInfo().extensionMap(m.pointer()).Set(xt, v)
	}
}
func (m *{{.}}) Mutable(fd protoreflect.FieldDescriptor) protoreflect.Value {
	m.messageInfo().init()
	if fi, xt := m.messageInfo().checkField(fd); fi != nil {
		return fi.mutable(m.pointer())
	} else {
		return m.messageInfo().extensionMap(m.pointer()).Mutable(xt)
	}
}
func (m *{{.}}) NewMessage(fd protoreflect.FieldDescriptor) protoreflect.Message {
	return m.NewField(fd).Message()
}
func (m *{{.}}) NewField(fd protoreflect.FieldDescriptor) protoreflect.Value {
	m.messageInfo().init()
	if fi, xt := m.messageInfo().checkField(fd); fi != nil {
		return fi.newField()
	} else {
		return xt.New()
	}
}
func (m *{{.}}) WhichOneof(od protoreflect.OneofDescriptor) protoreflect.FieldDescriptor {
	m.messageInfo().init()
	if oi := m.messageInfo().oneofs[od.Name()]; oi != nil && oi.oneofDesc == od {
		return od.Fields().ByNumber(oi.which(m.pointer()))
	}
	panic("invalid oneof descriptor")
}
func (m *{{.}}) GetUnknown() protoreflect.RawFields {
	m.messageInfo().init()
	return m.messageInfo().getUnknown(m.pointer())
}
func (m *{{.}}) SetUnknown(b protoreflect.RawFields) {
	m.messageInfo().init()
	m.messageInfo().setUnknown(m.pointer(), b)
}

{{end}}
`))
