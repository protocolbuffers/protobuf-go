// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protojson_test

import (
	"bytes"
	"runtime"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	pb2 "google.golang.org/protobuf/internal/testprotos/textpb2"
)

func marshalDeterministic(t *testing.T, m proto.Message) []byte {
	t.Helper()
	b, err := proto.MarshalOptions{AllowPartial: true, Deterministic: true}.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal(%T): %v", m, err)
	}
	return b
}

func anyOf(t *testing.T, m proto.Message) *anypb.Any {
	t.Helper()
	b := marshalDeterministic(t, m)
	return &anypb.Any{
		TypeUrl: "type.googleapis.com/" + string(m.ProtoReflect().Descriptor().FullName()),
		Value:   b,
	}
}

// nestAnyJSON builds a chain of depth Any messages around a StringValue.
// depth 1 is the StringValue Any itself.
func nestAnyJSON(depth int, typeLast bool) []byte {
	const (
		anyURL = "type.googleapis.com/google.protobuf.Any"
		strURL = "type.googleapis.com/google.protobuf.StringValue"
	)
	var b strings.Builder
	for i := 0; i < depth-1; i++ {
		if typeLast {
			b.WriteString(`{"value":`)
		} else {
			b.WriteString(`{"@type":"`)
			b.WriteString(anyURL)
			b.WriteString(`","value":`)
		}
	}
	if typeLast {
		b.WriteString(`{"value":"hi","@type":"`)
	} else {
		b.WriteString(`{"@type":"`)
		b.WriteString(strURL)
		b.WriteString(`","value":"hi"`)
	}
	if typeLast {
		b.WriteString(strURL)
		b.WriteString(`"}`)
	} else {
		b.WriteByte('}')
	}
	for i := 0; i < depth-1; i++ {
		if typeLast {
			b.WriteString(`,"@type":"`)
			b.WriteString(anyURL)
			b.WriteString(`"}`)
		} else {
			b.WriteByte('}')
		}
	}
	return []byte(b.String())
}

func referenceNestedAny(t *testing.T, depth int) *anypb.Any {
	t.Helper()
	var m proto.Message = &wrapperspb.StringValue{Value: "hi"}
	for i := 0; i < depth; i++ {
		m = anyOf(t, m)
	}
	return m.(*anypb.Any)
}

func TestNestedAnyJSON(t *testing.T) {
	for _, typeLast := range []bool{false, true} {
		order := "@type first"
		if typeLast {
			order = "@type last"
		}
		t.Run(order, func(t *testing.T) {
			for _, depth := range []int{1, 2, 3, 8} {
				got := &anypb.Any{}
				if err := protojson.Unmarshal(nestAnyJSON(depth, typeLast), got); err != nil {
					t.Fatalf("depth %d: %v", depth, err)
				}
				want := referenceNestedAny(t, depth)
				if !proto.Equal(got, want) {
					t.Fatalf("depth %d:\n got type %q value %x\nwant type %q value %x", depth, got.GetTypeUrl(), got.GetValue(), want.GetTypeUrl(), want.GetValue())
				}
			}
			// Field order does not change the wire bytes.
			a := &anypb.Any{}
			b := &anypb.Any{}
			if err := protojson.Unmarshal(nestAnyJSON(4, false), a); err != nil {
				t.Fatal(err)
			}
			if err := protojson.Unmarshal(nestAnyJSON(4, true), b); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(a.GetValue(), b.GetValue()) || a.GetTypeUrl() != b.GetTypeUrl() {
				t.Fatalf("field order changed Any bytes:\n first %x\n last  %x", a.GetValue(), b.GetValue())
			}
		})
	}
}

func TestAnyTypeLastMatchesMarshal(t *testing.T) {
	cases := []struct {
		name string
		json string
		want proto.Message
	}{{
		name: "Empty value before @type",
		json: `{"value":{},"@type":"type.googleapis.com/google.protobuf.Empty"}`,
		want: anyOf(t, &emptypb.Empty{}),
	}, {
		name: "StringValue value before @type",
		json: `{"value":"hi","@type":"type.googleapis.com/google.protobuf.StringValue"}`,
		want: anyOf(t, &wrapperspb.StringValue{Value: "hi"}),
	}, {
		name: "non-WKT fields before @type",
		json: `{
  "optString": "embedded inside Any",
  "optNested": {"optString": "inception"},
  "@type": "foo/pb2.Nested"
}`,
		want: func() proto.Message {
			m := &pb2.Nested{
				OptString: proto.String("embedded inside Any"),
				OptNested: &pb2.Nested{OptString: proto.String("inception")},
			}
			return &anypb.Any{TypeUrl: "foo/pb2.Nested", Value: marshalDeterministic(t, m)}
		}(),
	}, {
		name: "field before @type",
		json: `{"optString":"later","@type":"foo/pb2.Nested"}`,
		want: func() proto.Message {
			m := &pb2.Nested{OptString: proto.String("later")}
			return &anypb.Any{TypeUrl: "foo/pb2.Nested", Value: marshalDeterministic(t, m)}
		}(),
	}, {
		name: "Struct value before @type",
		json: `{"value":{"k":"v","n":1},"@type":"type.googleapis.com/google.protobuf.Struct"}`,
		want: anyOf(t, &structpb.Struct{Fields: map[string]*structpb.Value{
			"k": {Kind: &structpb.Value_StringValue{StringValue: "v"}},
			"n": {Kind: &structpb.Value_NumberValue{NumberValue: 1}},
		}}),
	}, {
		name: "Any of non-WKT, @type last at both levels",
		json: `{"value":{"optString":"z","@type":"foo/pb2.Nested"},"@type":"type.googleapis.com/google.protobuf.Any"}`,
		want: anyOf(t, &anypb.Any{
			TypeUrl: "foo/pb2.Nested",
			Value:   marshalDeterministic(t, &pb2.Nested{OptString: proto.String("z")}),
		}),
	}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := &anypb.Any{}
			if err := protojson.Unmarshal([]byte(tc.json), got); err != nil {
				t.Fatal(err)
			}
			if !proto.Equal(got, tc.want) {
				t.Fatalf("got %+v\nwant %+v", got, tc.want)
			}
		})
	}
}

func TestAnyTypeLastErrors(t *testing.T) {
	cases := []struct {
		name    string
		json    string
		opts    protojson.UnmarshalOptions
		wantErr string
		want    proto.Message
	}{{
		name:    "duplicate @type",
		json:    `{"value":"a","@type":"type.googleapis.com/google.protobuf.StringValue","@type":"foo/pb2.Nested"}`,
		wantErr: `duplicate "@type" field`,
	}, {
		name:    "empty @type",
		json:    `{"value":"a","@type":""}`,
		wantErr: `@type field contains empty value`,
	}, {
		name:    "missing @type",
		json:    `{"value":{"foo":1}}`,
		wantErr: `missing "@type" field`,
	}, {
		name: "missing @type DiscardUnknown",
		json: `{"value":{"foo":1},"unknown":true}`,
		opts: protojson.UnmarshalOptions{DiscardUnknown: true},
		want: &anypb.Any{},
	}, {
		name:    "duplicate value",
		json:    `{"value":"a","value":"b","@type":"type.googleapis.com/google.protobuf.StringValue"}`,
		wantErr: `duplicate "value" field`,
	}, {
		name:    "unknown field",
		json:    `{"optString":"hi","unknown":"x","@type":"foo/pb2.Nested"}`,
		wantErr: `unknown field "unknown"`,
	}, {
		name: "unknown field DiscardUnknown",
		json: `{"unknown":"x","@type":"foo/pb2.Nested","optString":"hi"}`,
		opts: protojson.UnmarshalOptions{DiscardUnknown: true},
		want: &anypb.Any{
			TypeUrl: "foo/pb2.Nested",
			Value:   marshalDeterministic(t, &pb2.Nested{OptString: proto.String("hi")}),
		},
	}, {
		name:    "invalid UTF-8 position preserved",
		json:    "{\n  \"optString\": \"" + "abc\xff" + "\",\n  \"@type\": \"foo/pb2.Nested\"\n}",
		wantErr: `(line 2:16): invalid UTF-8`,
	}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := &anypb.Any{}
			err := tc.opts.Unmarshal([]byte(tc.json), got)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("got nil error, want %q", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error %q, want substring %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tc.want != nil && !proto.Equal(got, tc.want) {
				t.Fatalf("got %+v\nwant %+v", got, tc.want)
			}
		})
	}
}

func TestAnyRecursionLimit(t *testing.T) {
	// N Any messages whose innermost embed is a StringValue are N+1 messages.
	// RecursionLimit counts each of them, including when "@type" is last
	// (that path used to be bounded only by skipJSONValue).
	const depth = 4
	for _, typeLast := range []bool{false, true} {
		order := "@type first"
		if typeLast {
			order = "@type last"
		}
		raw := nestAnyJSON(depth, typeLast)
		t.Run(order+" within limit", func(t *testing.T) {
			got := &anypb.Any{}
			opts := protojson.UnmarshalOptions{RecursionLimit: depth + 1}
			if err := opts.Unmarshal(raw, got); err != nil {
				t.Fatal(err)
			}
			if !proto.Equal(got, referenceNestedAny(t, depth)) {
				t.Fatalf("got %+v", got)
			}
		})
		t.Run(order+" over limit", func(t *testing.T) {
			opts := protojson.UnmarshalOptions{RecursionLimit: depth}
			err := opts.Unmarshal(raw, &anypb.Any{})
			if err == nil || !strings.Contains(err.Error(), "exceeded max recursion depth") {
				t.Fatalf("got %v, want exceeded max recursion depth", err)
			}
		})
	}

	// A non-WKT embed is counted by unmarshalMessage, not a second time.
	// Any + Nested + inner Nested = 3 messages.
	nested := `{"@type":"foo/pb2.Nested","optNested":{"optString":"x"}}`
	nestedLast := `{"optNested":{"optString":"x"},"@type":"foo/pb2.Nested"}`
	for _, raw := range []string{nested, nestedLast} {
		if err := (protojson.UnmarshalOptions{RecursionLimit: 3}).Unmarshal([]byte(raw), &anypb.Any{}); err != nil {
			t.Fatalf("RecursionLimit 3 on %s: %v", raw, err)
		}
		err := (protojson.UnmarshalOptions{RecursionLimit: 2}).Unmarshal([]byte(raw), &anypb.Any{})
		if err == nil || !strings.Contains(err.Error(), "exceeded max recursion depth") {
			t.Fatalf("RecursionLimit 2 on %s: got %v", raw, err)
		}
	}
}

// nestedAnyCost is the per-unmarshal time and bytes allocated, measured over
// enough iterations that the clock reading is not a single short sample.
type nestedAnyCost struct {
	ns    float64
	bytes float64
}

func measureNestedAny(t *testing.T, raw []byte) nestedAnyCost {
	t.Helper()
	if err := protojson.Unmarshal(raw, &anypb.Any{}); err != nil {
		t.Fatalf("warmup: %v", err)
	}
	n := 1
	var elapsed time.Duration
	var alloc uint64
	for {
		runtime.GC()
		var before runtime.MemStats
		runtime.ReadMemStats(&before)
		start := time.Now()
		for i := 0; i < n; i++ {
			if err := protojson.Unmarshal(raw, &anypb.Any{}); err != nil {
				t.Fatal(err)
			}
		}
		elapsed = time.Since(start)
		var after runtime.MemStats
		runtime.ReadMemStats(&after)
		alloc = after.TotalAlloc - before.TotalAlloc
		if elapsed >= 40*time.Millisecond || n >= 20000 {
			break
		}
		n *= 2
	}
	return nestedAnyCost{ns: float64(elapsed.Nanoseconds()) / float64(n), bytes: float64(alloc) / float64(n)}
}

// TestNestedAnyDecodeLinear fails if decoding a chain of Any messages becomes
// superlinear again. A quadratic scan or a per-level copy of the remaining
// payload is about 4x time and allocs when the depth doubles; linear growth
// stays near 2x. Depths are small enough that the check finishes quickly.
func TestNestedAnyDecodeLinear(t *testing.T) {
	const shallowDepth = 200
	const deepDepth = 400
	const maxRatio = 2.8

	for _, typeLast := range []bool{false, true} {
		name := "@type first"
		if typeLast {
			name = "@type last"
		}
		t.Run(name, func(t *testing.T) {
			shallow := measureNestedAny(t, nestAnyJSON(shallowDepth, typeLast))
			deep := measureNestedAny(t, nestAnyJSON(deepDepth, typeLast))
			timeRatio := deep.ns / shallow.ns
			byteRatio := deep.bytes / shallow.bytes
			t.Logf("depth %d → %d: time %.0f→%.0f ns (×%.2f), alloc %.0f→%.0f B (×%.2f)",
				shallowDepth, deepDepth, shallow.ns, deep.ns, timeRatio, shallow.bytes, deep.bytes, byteRatio)
			if timeRatio > maxRatio || byteRatio > maxRatio {
				t.Fatalf("depth doubled and cost grew superlinearly: time ×%.2f, alloc ×%.2f (limit ×%.1f)", timeRatio, byteRatio, maxRatio)
			}
		})
	}
}

func TestNestAnyJSONShape(t *testing.T) {
	// Guard the fixture used by the linearity test.
	first := string(nestAnyJSON(2, false))
	last := string(nestAnyJSON(2, true))
	wantFirst := `{"@type":"type.googleapis.com/google.protobuf.Any","value":{"@type":"type.googleapis.com/google.protobuf.StringValue","value":"hi"}}`
	wantLast := `{"value":{"value":"hi","@type":"type.googleapis.com/google.protobuf.StringValue"},"@type":"type.googleapis.com/google.protobuf.Any"}`
	if first != wantFirst {
		t.Fatalf("@type first:\n got %s\nwant %s", first, wantFirst)
	}
	if last != wantLast {
		t.Fatalf("@type last:\n got %s\nwant %s", last, wantLast)
	}
	if strings.Count(string(nestAnyJSON(5, false)), "@type") != 5 {
		t.Fatal("unexpected @type count")
	}
}
