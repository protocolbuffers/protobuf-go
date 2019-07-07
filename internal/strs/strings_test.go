// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package strs

import (
	"strconv"
	"testing"
)

func TestName(t *testing.T) {
	tests := []struct {
		in                string
		inEnumPrefix      string
		wantMapEntry      string
		wantEnumValue     string
		wantTrimValue     string
		wantJSONCamelCase string
		wantJSONSnakeCase string
	}{{
		in:                "abc",
		inEnumPrefix:      "",
		wantMapEntry:      "AbcEntry",
		wantEnumValue:     "Abc",
		wantTrimValue:     "abc",
		wantJSONCamelCase: "abc",
		wantJSONSnakeCase: "abc",
	}, {
		in:                "foo_baR_",
		inEnumPrefix:      "foo_bar",
		wantMapEntry:      "FooBaREntry",
		wantEnumValue:     "FooBar",
		wantTrimValue:     "foo_baR_",
		wantJSONCamelCase: "fooBaR",
		wantJSONSnakeCase: "foo_ba_r_",
	}, {
		in:                "snake_caseCamelCase",
		inEnumPrefix:      "snakecasecamel",
		wantMapEntry:      "SnakeCaseCamelCaseEntry",
		wantEnumValue:     "SnakeCasecamelcase",
		wantTrimValue:     "Case",
		wantJSONCamelCase: "snakeCaseCamelCase",
		wantJSONSnakeCase: "snake_case_camel_case",
	}, {
		in:                "FiZz_BuZz",
		inEnumPrefix:      "fizz",
		wantMapEntry:      "FiZzBuZzEntry",
		wantEnumValue:     "FizzBuzz",
		wantTrimValue:     "BuZz",
		wantJSONCamelCase: "FiZzBuZz",
		wantJSONSnakeCase: "_fi_zz__bu_zz",
	}}

	for _, tt := range tests {
		if got := MapEntryName(tt.in); got != tt.wantMapEntry {
			t.Errorf("MapEntryName(%q) = %q, want %q", tt.in, got, tt.wantMapEntry)
		}
		if got := EnumValueName(tt.in); got != tt.wantEnumValue {
			t.Errorf("EnumValueName(%q) = %q, want %q", tt.in, got, tt.wantEnumValue)
		}
		if got := TrimEnumPrefix(tt.in, tt.inEnumPrefix); got != tt.wantTrimValue {
			t.Errorf("ErimEnumPrefix(%q, %q) = %q, want %q", tt.in, tt.inEnumPrefix, got, tt.wantTrimValue)
		}
		if got := JSONCamelCase(tt.in); got != tt.wantJSONCamelCase {
			t.Errorf("JSONCamelCase(%q) = %q, want %q", tt.in, got, tt.wantJSONCamelCase)
		}
		if got := JSONSnakeCase(tt.in); got != tt.wantJSONSnakeCase {
			t.Errorf("JSONSnakeCase(%q) = %q, want %q", tt.in, got, tt.wantJSONSnakeCase)
		}
	}
}

var (
	srcString = "1234"
	srcBytes  = []byte(srcString)
	dst       uint64
)

func BenchmarkCast(b *testing.B) {
	b.Run("Ideal", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dst, _ = strconv.ParseUint(srcString, 0, 64)
		}
		if dst != 1234 {
			b.Errorf("got %d, want %s", dst, srcString)
		}
	})
	b.Run("Copy", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dst, _ = strconv.ParseUint(string(srcBytes), 0, 64)
		}
		if dst != 1234 {
			b.Errorf("got %d, want %s", dst, srcString)
		}
	})
	b.Run("Cast", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dst, _ = strconv.ParseUint(UnsafeString(srcBytes), 0, 64)
		}
		if dst != 1234 {
			b.Errorf("got %d, want %s", dst, srcString)
		}
	})
}
