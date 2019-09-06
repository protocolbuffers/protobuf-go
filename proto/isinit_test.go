// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style.
// license that can be found in the LICENSE file.

package proto_test

import (
	"fmt"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	testpb "google.golang.org/protobuf/internal/testprotos/test"
)

func TestIsInitializedErrors(t *testing.T) {
	for _, test := range []struct {
		m    proto.Message
		want string
	}{
		{
			&testpb.TestRequired{},
			`goproto.proto.test.TestRequired.required_field`,
		},
		{
			&testpb.TestRequiredForeign{
				OptionalMessage: &testpb.TestRequired{},
			},
			`goproto.proto.test.TestRequired.required_field`,
		},
		{
			&testpb.TestRequiredForeign{
				RepeatedMessage: []*testpb.TestRequired{
					{RequiredField: proto.Int32(1)},
					{},
				},
			},
			`goproto.proto.test.TestRequired.required_field`,
		},
		{
			&testpb.TestRequiredForeign{
				MapMessage: map[int32]*testpb.TestRequired{
					1: {},
				},
			},
			`goproto.proto.test.TestRequired.required_field`,
		},
	} {
		err := proto.IsInitialized(test.m)
		got := "<nil>"
		if err != nil {
			got = fmt.Sprintf("%q", err)
		}
		if !strings.Contains(got, test.want) {
			t.Errorf("IsInitialized(m):\n got: %v\nwant contains: %v\nMessage:\n%v", got, test.want, marshalText(test.m))
		}
	}
}
