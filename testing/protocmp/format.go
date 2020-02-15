// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protocmp

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/internal/detrand"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// This implements a custom text marshaler similar to the prototext format.
// We don't use the prototext marshaler so that we can:
//	• have finer grain control over the ordering of fields
//	• marshal maps with a more aesthetically pleasant output
//
// TODO: If the prototext format gains a map-specific syntax, consider just
// using the prototext marshaler instead.

func appendValue(b []byte, v interface{}) []byte {
	switch v := v.(type) {
	case bool, int32, int64, uint32, uint64, float32, float64:
		return append(b, fmt.Sprint(v)...)
	case string:
		return append(b, strconv.Quote(string(v))...)
	case []byte:
		return append(b, strconv.Quote(string(v))...)
	case Enum:
		return append(b, v.String()...)
	case Message:
		return appendMessage(b, v)
	case protoreflect.RawFields:
		return appendValue(b, transformRawFields(v))
	default:
		switch v := reflect.ValueOf(v); v.Kind() {
		case reflect.Slice:
			return appendList(b, v)
		case reflect.Map:
			return appendMap(b, v)
		default:
			panic(fmt.Sprintf("invalid type: %v", v.Type()))
		}
	}
}

func appendMessage(b []byte, m Message) []byte {
	var knownKeys, extensionKeys, unknownKeys []string
	for k := range m {
		switch {
		case protoreflect.Name(k).IsValid():
			knownKeys = append(knownKeys, k)
		case strings.HasPrefix(k, "[") && strings.HasSuffix(k, "]"):
			extensionKeys = append(extensionKeys, k)
		case len(strings.Trim(k, "0123456789")) == 0:
			unknownKeys = append(unknownKeys, k)
		}
	}
	sort.Slice(knownKeys, func(i, j int) bool {
		fdi := m.Descriptor().Fields().ByName(protoreflect.Name(knownKeys[i]))
		fdj := m.Descriptor().Fields().ByName(protoreflect.Name(knownKeys[j]))
		return fdi.Index() < fdj.Index()
	})
	sort.Slice(extensionKeys, func(i, j int) bool {
		return extensionKeys[i] < extensionKeys[j]
	})
	sort.Slice(unknownKeys, func(i, j int) bool {
		ni, _ := strconv.Atoi(unknownKeys[i])
		nj, _ := strconv.Atoi(unknownKeys[j])
		return ni < nj
	})
	ks := append(append(append([]string(nil), knownKeys...), extensionKeys...), unknownKeys...)

	b = append(b, '{')
	for _, k := range ks {
		b = append(b, k...)
		b = append(b, ':')
		b = appendValue(b, m[k])
		b = append(b, delim()...)
	}
	b = bytes.TrimRight(b, delim())
	b = append(b, '}')
	return b
}

func appendList(b []byte, v reflect.Value) []byte {
	b = append(b, '[')
	for i := 0; i < v.Len(); i++ {
		b = appendValue(b, v.Index(i).Interface())
		b = append(b, delim()...)
	}
	b = bytes.TrimRight(b, delim())
	b = append(b, ']')
	return b
}

func appendMap(b []byte, v reflect.Value) []byte {
	ks := v.MapKeys()
	sort.Slice(ks, func(i, j int) bool {
		ki, kj := ks[i], ks[j]
		switch ki.Kind() {
		case reflect.Bool:
			return !ki.Bool() && kj.Bool()
		case reflect.Int32, reflect.Int64:
			return ki.Int() < kj.Int()
		case reflect.Uint32, reflect.Uint64:
			return ki.Uint() < kj.Uint()
		case reflect.String:
			return ki.String() < kj.String()
		default:
			panic(fmt.Sprintf("invalid kind: %v", ki.Kind()))
		}
	})

	b = append(b, '{')
	for _, k := range ks {
		b = appendValue(b, k.Interface())
		b = append(b, ':')
		b = appendValue(b, v.MapIndex(k).Interface())
		b = append(b, delim()...)
	}
	b = bytes.TrimRight(b, delim())
	b = append(b, '}')
	return b
}

func transformRawFields(b protoreflect.RawFields) interface{} {
	var vs []interface{}
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		m := protowire.ConsumeFieldValue(num, typ, b[n:])
		vs = append(vs, transformRawField(typ, b[n:][:m]))
		b = b[n+m:]
	}
	if len(vs) == 1 {
		return vs[0]
	}
	return vs
}

func transformRawField(typ protowire.Type, b protoreflect.RawFields) interface{} {
	switch typ {
	case protowire.VarintType:
		v, _ := protowire.ConsumeVarint(b)
		return v
	case protowire.Fixed32Type:
		v, _ := protowire.ConsumeFixed32(b)
		return v
	case protowire.Fixed64Type:
		v, _ := protowire.ConsumeFixed64(b)
		return v
	case protowire.BytesType:
		v, _ := protowire.ConsumeBytes(b)
		return v
	case protowire.StartGroupType:
		v := Message{}
		for {
			num2, typ2, n := protowire.ConsumeTag(b)
			if typ2 == protowire.EndGroupType {
				return v
			}
			m := protowire.ConsumeFieldValue(num2, typ2, b[n:])
			s := strconv.Itoa(int(num2))
			b2, _ := v[s].(protoreflect.RawFields)
			v[s] = append(b2, b[:n+m]...)
			b = b[n+m:]
		}
	default:
		panic(fmt.Sprintf("invalid type: %v", typ))
	}
}

func delim() string {
	// Deliberately introduce instability into the message string to
	// discourage users from depending on it.
	if detrand.Bool() {
		return "  "
	}
	return ", "
}
