// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto_test

import (
	"flag"
	"fmt"
	"reflect"
	"testing"

	"google.golang.org/protobuf/proto"
)

// The results of these microbenchmarks are unlikely to correspond well
// to real world performance. They are mainly useful as a quick check to
// detect unexpected regressions and for profiling specific cases.

var (
	allowPartial = flag.Bool("allow_partial", false, "set AllowPartial")
)

// BenchmarkEncode benchmarks encoding all the test messages.
func BenchmarkEncode(b *testing.B) {
	for _, test := range testValidMessages {
		for _, want := range test.decodeTo {
			opts := proto.MarshalOptions{AllowPartial: *allowPartial}
			b.Run(fmt.Sprintf("%s (%T)", test.desc, want), func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						_, err := opts.Marshal(want)
						if err != nil && !test.partial {
							b.Fatal(err)
						}
					}
				})
			})
		}
	}
}

// BenchmarkDecode benchmarks decoding all the test messages.
func BenchmarkDecode(b *testing.B) {
	for _, test := range testValidMessages {
		for _, want := range test.decodeTo {
			opts := proto.UnmarshalOptions{AllowPartial: *allowPartial}
			b.Run(fmt.Sprintf("%s (%T)", test.desc, want), func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						m := reflect.New(reflect.TypeOf(want).Elem()).Interface().(proto.Message)
						err := opts.Unmarshal(test.wire, m)
						if err != nil && !test.partial {
							b.Fatal(err)
						}
					}
				})
			})
		}
	}
}

// BenchmarkSize benchmarks sizing open and opaque messages across field densities.
func BenchmarkSize(b *testing.B) {
	densities := []struct {
		name   string
		stride int
	}{
		{name: "empty", stride: 0},
		{name: "sparse", stride: 32},
		{name: "medium", stride: 8},
		{name: "dense", stride: 1},
	}
	for _, apiName := range []string{"open", "opaque"} {
		b.Run(apiName, func(b *testing.B) {
			for _, d := range densities {
				open, opaque := sparseMessages(d.stride)
				m := open
				if apiName == "opaque" {
					m = opaque
				}
				b.Run(d.name, func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						_ = proto.Size(m)
					}
				})
			}
		})
	}
}

// BenchmarkMarshal benchmarks marshaling open and opaque messages across field
// densities into a preallocated buffer with cached sizes primed.
func BenchmarkMarshal(b *testing.B) {
	densities := []struct {
		name   string
		stride int
	}{
		{name: "empty", stride: 0},
		{name: "sparse", stride: 32},
		{name: "medium", stride: 8},
		{name: "dense", stride: 1},
	}
	for _, apiName := range []string{"open", "opaque"} {
		b.Run(apiName, func(b *testing.B) {
			for _, d := range densities {
				open, opaque := sparseMessages(d.stride)
				m := open
				if apiName == "opaque" {
					m = opaque
				}
				size := proto.Size(m)
				buf := make([]byte, 0, size)
				b.Run(d.name, func(b *testing.B) {
					opts := proto.MarshalOptions{UseCachedSize: true}
					for i := 0; i < b.N; i++ {
						_, err := opts.MarshalAppend(buf[:0], m)
						if err != nil {
							b.Fatal(err)
						}
					}
				})
			}
		})
	}
}
