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

// size{{.Name}} returns the size of wire encoding a {{.GoType}} pointer as a {{.Name}}.
// The zero value is not encoded.
func size{{.Name}}NoZero(p pointer, tagsize int, _ marshalOptions) (size int) {
	v := *p.{{.GoType.PointerMethod}}()
	if {{template "IsZero" .}} {
		return 0
	}
	return tagsize + {{template "Size" .}}
}

// append{{.Name}} wire encodes a {{.GoType}} pointer as a {{.Name}}.
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

var coder{{.Name}}NoZero = pointerCoderFuncs{
	size:      size{{.Name}}NoZero,
	marshal:   append{{.Name}}NoZero,
	unmarshal: consume{{.Name}},
}

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

// size{{.Name}}Iface returns the size of wire encoding a {{.GoType}} value as a {{.Name}}.
func size{{.Name}}Iface(ival interface{}, tagsize int, _ marshalOptions) int {
	{{- if not .WireType.ConstSize}}
	v := ival.({{.GoType}})
	{{end -}}
	return tagsize + {{template "Size" .}}
}

// append{{.Name}}Iface encodes a {{.GoType}} value as a {{.Name}}.
func append{{.Name}}Iface(b []byte, ival interface{}, wiretag uint64, _ marshalOptions) ([]byte, error) {
	v := ival.({{.GoType}})
	b = wire.AppendVarint(b, wiretag)
	{{template "Append" .}}
	return b, nil
}

// consume{{.Name}}Iface decodes a {{.GoType}} value as a {{.Name}}.
func consume{{.Name}}Iface(b []byte, _ interface{}, _ wire.Number, wtyp wire.Type, _ unmarshalOptions) (interface{}, int, error) {
	if wtyp != {{.WireType.Expr}} {
		return nil, 0, errUnknown
	}
	v, n := {{template "Consume" .}}
	if n < 0 {
		return nil, 0, wire.ParseError(n)
	}
	return {{.ToGoType}}, n, nil
}

var coder{{.Name}}Iface = ifaceCoderFuncs{
	size:    size{{.Name}}Iface,
	marshal: append{{.Name}}Iface,
	unmarshal: consume{{.Name}}Iface,
}

// size{{.Name}}SliceIface returns the size of wire encoding a []{{.GoType}} value as a repeated {{.Name}}.
func size{{.Name}}SliceIface(ival interface{}, tagsize int, _ marshalOptions) (size int) {
	s := *ival.(*[]{{.GoType}})
	{{if .WireType.ConstSize -}}
	size = len(s) * (tagsize + {{template "Size" .}})
	{{- else -}}
	for _, v := range s {
		size += tagsize + {{template "Size" .}}
	}
	{{- end}}
	return size
}

// append{{.Name}}SliceIface encodes a []{{.GoType}} value as a repeated {{.Name}}.
func append{{.Name}}SliceIface(b []byte, ival interface{}, wiretag uint64, _ marshalOptions) ([]byte, error) {
	s := *ival.(*[]{{.GoType}})
	for _, v := range s {
		b = wire.AppendVarint(b, wiretag)
		{{template "Append" .}}
	}
	return b, nil
}

// consume{{.Name}}SliceIface wire decodes a []{{.GoType}} value as a repeated {{.Name}}.
func consume{{.Name}}SliceIface(b []byte, ival interface{}, _ wire.Number, wtyp wire.Type, _ unmarshalOptions) (_ interface{}, n int, err error) {
	sp := ival.(*[]{{.GoType}})
	{{- if .WireType.Packable}}
	if wtyp == wire.BytesType {
		s := *sp
		b, n = wire.ConsumeBytes(b)
		if n < 0 {
			return nil, 0, wire.ParseError(n)
		}
		for len(b) > 0 {
			v, n := {{template "Consume" .}}
			if n < 0 {
				return nil, 0, wire.ParseError(n)
			}
			s = append(s, {{.ToGoType}})
			b = b[n:]
		}
		*sp = s
		return ival, n, nil
	}
	{{- end}}
	if wtyp != {{.WireType.Expr}} {
		return nil, 0, errUnknown
	}
	v, n := {{template "Consume" .}}
	if n < 0 {
		return nil, 0, wire.ParseError(n)
	}
	*sp = append(*sp, {{.ToGoType}})
	return ival, n, nil
}

var coder{{.Name}}SliceIface = ifaceCoderFuncs{
	size:      size{{.Name}}SliceIface,
	marshal:   append{{.Name}}SliceIface,
	unmarshal: consume{{.Name}}SliceIface,
}

{{if or (eq .WireType "Varint") (eq .WireType "Fixed32") (eq .WireType "Fixed64")}}
// size{{.Name}}PackedSliceIface returns the size of wire encoding a []{{.GoType}} value as a packed repeated {{.Name}}.
func size{{.Name}}PackedSliceIface(ival interface{}, tagsize int, _ marshalOptions) (size int) {
	s := *ival.(*[]{{.GoType}})
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

// append{{.Name}}PackedSliceIface encodes a []{{.GoType}} value as a packed repeated {{.Name}}.
func append{{.Name}}PackedSliceIface(b []byte, ival interface{}, wiretag uint64, _ marshalOptions) ([]byte, error) {
	s := *ival.(*[]{{.GoType}})
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

var coder{{.Name}}PackedSliceIface = ifaceCoderFuncs{
	size:      size{{.Name}}PackedSliceIface,
	marshal:   append{{.Name}}PackedSliceIface,
	unmarshal: consume{{.Name}}SliceIface,
}
{{end}}

{{end -}}
{{end -}}

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
	return m.mi.PBType.Descriptor()
}
func (m *{{.}}) New() protoreflect.Message {
	return m.mi.PBType.New()
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
	return m.pointer().AsIfaceOf(m.mi.GoType.Elem())
}

func (m *{{.}}) Range(f func(protoreflect.FieldDescriptor, protoreflect.Value) bool) {
	m.mi.init()
	for _, fi := range m.mi.fields {
		if fi.has(m.pointer()) {
			if !f(fi.fieldDesc, fi.get(m.pointer())) {
				return
			}
		}
	}
	m.mi.extensionMap(m.pointer()).Range(f)
}
func (m *{{.}}) Has(fd protoreflect.FieldDescriptor) bool {
	m.mi.init()
	if fi, xt := m.mi.checkField(fd); fi != nil {
		return fi.has(m.pointer())
	} else {
		return m.mi.extensionMap(m.pointer()).Has(xt)
	}
}
func (m *{{.}}) Clear(fd protoreflect.FieldDescriptor) {
	m.mi.init()
	if fi, xt := m.mi.checkField(fd); fi != nil {
		fi.clear(m.pointer())
	} else {
		m.mi.extensionMap(m.pointer()).Clear(xt)
	}
}
func (m *{{.}}) Get(fd protoreflect.FieldDescriptor) protoreflect.Value {
	m.mi.init()
	if fi, xt := m.mi.checkField(fd); fi != nil {
		return fi.get(m.pointer())
	} else {
		return m.mi.extensionMap(m.pointer()).Get(xt)
	}
}
func (m *{{.}}) Set(fd protoreflect.FieldDescriptor, v protoreflect.Value) {
	m.mi.init()
	if fi, xt := m.mi.checkField(fd); fi != nil {
		fi.set(m.pointer(), v)
	} else {
		m.mi.extensionMap(m.pointer()).Set(xt, v)
	}
}
func (m *{{.}}) Mutable(fd protoreflect.FieldDescriptor) protoreflect.Value {
	m.mi.init()
	if fi, xt := m.mi.checkField(fd); fi != nil {
		return fi.mutable(m.pointer())
	} else {
		return m.mi.extensionMap(m.pointer()).Mutable(xt)
	}
}
func (m *{{.}}) NewMessage(fd protoreflect.FieldDescriptor) protoreflect.Message {
	m.mi.init()
	if fi, xt := m.mi.checkField(fd); fi != nil {
		return fi.newMessage()
	} else {
		return xt.New().Message()
	}
}
func (m *{{.}}) WhichOneof(od protoreflect.OneofDescriptor) protoreflect.FieldDescriptor {
	m.mi.init()
	if oi := m.mi.oneofs[od.Name()]; oi != nil && oi.oneofDesc == od {
		return od.Fields().ByNumber(oi.which(m.pointer()))
	}
	panic("invalid oneof descriptor")
}
func (m *{{.}}) GetUnknown() protoreflect.RawFields {
	m.mi.init()
	return m.mi.getUnknown(m.pointer())
}
func (m *{{.}}) SetUnknown(b protoreflect.RawFields) {
	m.mi.init()
	m.mi.setUnknown(m.pointer(), b)
}

{{end}}
`))
