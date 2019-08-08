// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package strs provides string manipulation functionality specific to protobuf.
package strs

import (
	"strings"
	"unicode"

	"google.golang.org/protobuf/internal/flags"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// EnforceUTF8 reports whether to enforce strict UTF-8 validation.
func EnforceUTF8(fd protoreflect.FieldDescriptor) bool {
	if flags.ProtoLegacy {
		if fd, ok := fd.(interface{ EnforceUTF8() bool }); ok {
			return fd.EnforceUTF8()
		}
	}
	return fd.Syntax() == protoreflect.Proto3
}

// JSONCamelCase converts a snake_case identifier to a camelCase identifier,
// according to the protobuf JSON specification.
func JSONCamelCase(s string) string {
	var b []byte
	var wasUnderscore bool
	for i := 0; i < len(s); i++ { // proto identifiers are always ASCII
		c := s[i]
		if c != '_' {
			isLower := 'a' <= c && c <= 'z'
			if wasUnderscore && isLower {
				c -= 'a' - 'A' // convert to uppercase
			}
			b = append(b, c)
		}
		wasUnderscore = c == '_'
	}
	return string(b)
}

// JSONSnakeCase converts a camelCase identifier to a snake_case identifier,
// according to the protobuf JSON specification.
func JSONSnakeCase(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ { // proto identifiers are always ASCII
		c := s[i]
		isUpper := 'A' <= c && c <= 'Z'
		if isUpper {
			b = append(b, '_')
			c += 'a' - 'A' // convert to lowercase
		}
		b = append(b, c)
	}
	return string(b)
}

// MapEntryName derives the name of the map entry message given the field name.
// See protoc v3.8.0: src/google/protobuf/descriptor.cc:254-276,6057
func MapEntryName(s string) string {
	var b []byte
	upperNext := true
	for _, c := range s {
		switch {
		case c == '_':
			upperNext = true
		case upperNext:
			b = append(b, byte(unicode.ToUpper(c)))
			upperNext = false
		default:
			b = append(b, byte(c))
		}
	}
	b = append(b, "Entry"...)
	return string(b)
}

// EnumValueName derives the camel-cased enum value name.
// See protoc v3.8.0: src/google/protobuf/descriptor.cc:297-313
func EnumValueName(s string) string {
	var b []byte
	upperNext := true
	for _, c := range s {
		switch {
		case c == '_':
			upperNext = true
		case upperNext:
			b = append(b, byte(unicode.ToUpper(c)))
			upperNext = false
		default:
			b = append(b, byte(unicode.ToLower(c)))
			upperNext = false
		}
	}
	return string(b)
}

// TrimEnumPrefix trims the enum name prefix from an enum value name,
// where the prefix is all lowercase without underscores.
// See protoc v3.8.0: src/google/protobuf/descriptor.cc:330-375
func TrimEnumPrefix(s, prefix string) string {
	s0 := s // original input
	for len(s) > 0 && len(prefix) > 0 {
		if s[0] == '_' {
			s = s[1:]
			continue
		}
		if unicode.ToLower(rune(s[0])) != rune(prefix[0]) {
			return s0 // no prefix match
		}
		s, prefix = s[1:], prefix[1:]
	}
	if len(prefix) > 0 {
		return s0 // no prefix match
	}
	s = strings.TrimLeft(s, "_")
	if len(s) == 0 {
		return s0 // avoid returning empty string
	}
	return s
}
