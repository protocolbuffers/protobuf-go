// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protopath

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/reflect/protoreflect"

	pb "google.golang.org/protobuf/reflect/protopath/testmessage"
)

func TestPathValues(t *testing.T) {
	simple := &pb.Test{
		Nested: &pb.Test_Nested{
			Stringfield: "stringfield",
		},
	}
	simplerefl := protoreflect.ValueOf(simple.ProtoReflect())
	simplemd := simple.ProtoReflect().Descriptor()
	rep := &pb.Test{Repeats: []*pb.Test{simple}}
	m := &pb.Test{Strkeymap: map[string]*pb.Test_Nested{"k": simple.GetNested()}}
	tcs := []struct {
		name      string
		msg       *pb.Test
		path      Path
		want      Values
		wantErr   string
		wantPanic string
	}{
		{
			name: "root is self",
			msg:  simple,
			path: Path{Root(simplemd)},
			want: Values{Path: Path{Root(simplemd)},
				Values: []protoreflect.Value{simplerefl}},
		},
		{
			name: "field access",
			msg:  simple,
			path: Path{Root(simplemd),
				FieldAccess(simplemd.Fields().ByTextName("nested"))},
			want: Values{Path: Path{Root(simplemd),
				FieldAccess(simplemd.Fields().ByTextName("nested"))},
				Values: []protoreflect.Value{simplerefl, protoreflect.ValueOf(simple.GetNested().ProtoReflect())}},
		},
		{
			name: "index access",
			msg:  rep,
			path: Path{Root(simplemd),
				FieldAccess(simplemd.Fields().ByTextName("repeats")),
				ListIndex(0),
				FieldAccess(simplemd.Fields().ByTextName("nested")),
				FieldAccess(rep.GetNested().ProtoReflect().Descriptor().Fields().ByTextName("stringfield"))},
			want: Values{Path: Path{Root(simplemd),
				FieldAccess(simplemd.Fields().ByTextName("repeats")),
				ListIndex(0),
				FieldAccess(simplemd.Fields().ByTextName("nested")),
				FieldAccess(rep.GetNested().ProtoReflect().Descriptor().Fields().ByTextName("stringfield"))},
				Values: []protoreflect.Value{
					protoreflect.ValueOf(rep.ProtoReflect()),
					rep.ProtoReflect().Get(simplemd.Fields().ByTextName("repeats")),
					simplerefl,
					protoreflect.ValueOf(simple.GetNested().ProtoReflect()),
					protoreflect.ValueOf("stringfield"),
				}},
		},
		{
			name: "map access",
			msg:  m,
			path: Path{Root(simplemd),
				FieldAccess(simplemd.Fields().ByTextName("strkeymap")),
				MapIndex(protoreflect.ValueOf("k").MapKey()),
				FieldAccess(simple.GetNested().ProtoReflect().Descriptor().Fields().ByTextName("stringfield")),
			},
			want: Values{
				Path: Path{Root(simplemd),
					FieldAccess(simplemd.Fields().ByTextName("strkeymap")),
					MapIndex(protoreflect.ValueOf("k").MapKey()),
					FieldAccess(simple.GetNested().ProtoReflect().Descriptor().Fields().ByTextName("stringfield")),
				},
				Values: []protoreflect.Value{
					protoreflect.ValueOf(m.ProtoReflect()),
					m.ProtoReflect().Get(simplemd.Fields().ByTextName("strkeymap")),
					protoreflect.ValueOf(simple.GetNested().ProtoReflect()),
					protoreflect.ValueOf("stringfield"),
				}},
		},
		{
			name:    "repeated root",
			msg:     m,
			path:    Path{Root(simplemd), Root(simplemd)},
			wantErr: "root step at index 1",
		},
		{
			name: "index strmap",
			msg:  m,
			path: Path{Root(simplemd),
				FieldAccess(simplemd.Fields().ByTextName("strkeymap")),
				MapIndex(protoreflect.ValueOf(int32(321)).MapKey())},
			wantPanic: "int32, not string",
		},
		{
			name: "index missing key",
			msg:  m,
			path: Path{Root(simplemd),
				FieldAccess(simplemd.Fields().ByTextName("strkeymap")),
				MapIndex(protoreflect.ValueOf("where").MapKey())},
			wantErr: "missing key where",
		},
		{
			name: "index out of range",
			msg:  rep,
			path: Path{Root(simplemd),
				FieldAccess(simplemd.Fields().ByTextName("repeats")),
				ListIndex(1)},
			wantErr: "out of range",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if tc.wantPanic != "" {
					if r == nil {
						t.Fatalf("expected panic did not happen: %q", tc.wantPanic)
					}
					if !strings.Contains(r.(error).Error(), tc.wantPanic) {
						t.Fatalf("panic value unexpected: %v. Want %q", r, tc.wantPanic)
					}
				} else if r != nil {
					t.Fatalf("unexpected panic: %v", r)
				}
			}()
			got, err := PathValues(tc.path, tc.msg)
			if !matchErr(err, tc.wantErr) {
				t.Fatalf("PathValues(%v, %v) = _, %v errored unexpectedly. Want %q", tc.path, tc.msg, err, tc.wantErr)
			}
			if tc.wantErr != "" {
				return
			}
			type cmpValues struct {
				Pathstr string
				Value   []protoreflect.Value
			}
			if diff := cmp.Diff(tc.want, got, cmp.Transformer("Values",
				func(p Values) cmpValues {
					return cmpValues{Pathstr: p.Path.String(), Value: p.Values}
				})); diff != "" {
				t.Errorf("PathValues(%v, %v) returned diff (-want +got):\n%s", tc.path, tc.msg, diff)
			}
		})
	}
}
