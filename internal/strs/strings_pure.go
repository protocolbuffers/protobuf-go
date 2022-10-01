// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build purego || appengine
// +build purego appengine

package strs

import pref "github.com/infiniteloopcloud/protoc-gen-go-types/reflect/protoreflect"

func UnsafeString(b []byte) string {
	return string(b)
}

func UnsafeBytes(s string) []byte {
	return []byte(s)
}

type Builder struct{}

func (*Builder) AppendFullName(prefix pref.FullName, name pref.Name) pref.FullName {
	return prefix.Append(name)
}

func (*Builder) MakeString(b []byte) string {
	return string(b)
}
