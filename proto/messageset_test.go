// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style.
// license that can be found in the LICENSE file.

package proto_test

import (
	"google.golang.org/protobuf/internal/encoding/pack"
	"google.golang.org/protobuf/internal/flags"
	"google.golang.org/protobuf/proto"

	messagesetpb "google.golang.org/protobuf/internal/testprotos/messageset/messagesetpb"
	msetextpb "google.golang.org/protobuf/internal/testprotos/messageset/msetextpb"
)

func init() {
	if flags.Proto1Legacy {
		testProtos = append(testProtos, messageSetTestProtos...)
	}
}

var messageSetTestProtos = []testProto{
	{
		desc: "MessageSet type_id before message content",
		decodeTo: []proto.Message{build(
			&messagesetpb.MessageSet{},
			extend(msetextpb.E_Ext1_MessageSetExtension, &msetextpb.Ext1{
				Ext1Field1: proto.Int32(10),
			}),
		)},
		wire: pack.Message{
			pack.Tag{1, pack.StartGroupType},
			pack.Tag{2, pack.VarintType}, pack.Varint(1000),
			pack.Tag{3, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(10),
			}),
			pack.Tag{1, pack.EndGroupType},
		}.Marshal(),
	},
	{
		desc: "MessageSet type_id after message content",
		decodeTo: []proto.Message{build(
			&messagesetpb.MessageSet{},
			extend(msetextpb.E_Ext1_MessageSetExtension, &msetextpb.Ext1{
				Ext1Field1: proto.Int32(10),
			}),
		)},
		wire: pack.Message{
			pack.Tag{1, pack.StartGroupType},
			pack.Tag{3, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(10),
			}),
			pack.Tag{2, pack.VarintType}, pack.Varint(1000),
			pack.Tag{1, pack.EndGroupType},
		}.Marshal(),
	},
	{
		desc: "MessageSet preserves unknown field",
		decodeTo: []proto.Message{build(
			&messagesetpb.MessageSet{},
			extend(msetextpb.E_Ext1_MessageSetExtension, &msetextpb.Ext1{
				Ext1Field1: proto.Int32(10),
			}),
			unknown(pack.Message{
				pack.Tag{4, pack.VarintType}, pack.Varint(30),
			}.Marshal()),
		)},
		wire: pack.Message{
			pack.Tag{1, pack.StartGroupType},
			pack.Tag{2, pack.VarintType}, pack.Varint(1000),
			pack.Tag{3, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(10),
			}),
			pack.Tag{1, pack.EndGroupType},
			// Unknown field
			pack.Tag{4, pack.VarintType}, pack.Varint(30),
		}.Marshal(),
	},
	{
		desc: "MessageSet with unknown type_id",
		decodeTo: []proto.Message{build(
			&messagesetpb.MessageSet{},
			unknown(pack.Message{
				pack.Tag{1, pack.StartGroupType},
				pack.Tag{2, pack.VarintType}, pack.Varint(1002),
				pack.Tag{3, pack.BytesType}, pack.LengthPrefix(pack.Message{
					pack.Tag{1, pack.VarintType}, pack.Varint(10),
				}),
				pack.Tag{1, pack.EndGroupType},
			}.Marshal()),
		)},
		wire: pack.Message{
			pack.Tag{1, pack.StartGroupType},
			pack.Tag{2, pack.VarintType}, pack.Varint(1002),
			pack.Tag{3, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(10),
			}),
			pack.Tag{1, pack.EndGroupType},
		}.Marshal(),
	},
	{
		desc: "MessageSet merges repeated message fields in item",
		decodeTo: []proto.Message{build(
			&messagesetpb.MessageSet{},
			extend(msetextpb.E_Ext1_MessageSetExtension, &msetextpb.Ext1{
				Ext1Field1: proto.Int32(10),
				Ext1Field2: proto.Int32(20),
			}),
		)},
		wire: pack.Message{
			pack.Tag{1, pack.StartGroupType},
			pack.Tag{2, pack.VarintType}, pack.Varint(1000),
			pack.Tag{3, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(10),
			}),
			pack.Tag{3, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{2, pack.VarintType}, pack.Varint(20),
			}),
			pack.Tag{1, pack.EndGroupType},
		}.Marshal(),
	},
	{
		desc: "MessageSet merges message fields in repeated items",
		decodeTo: []proto.Message{build(
			&messagesetpb.MessageSet{},
			extend(msetextpb.E_Ext1_MessageSetExtension, &msetextpb.Ext1{
				Ext1Field1: proto.Int32(10),
				Ext1Field2: proto.Int32(20),
			}),
			extend(msetextpb.E_Ext2_MessageSetExtension, &msetextpb.Ext2{
				Ext2Field1: proto.Int32(30),
			}),
		)},
		wire: pack.Message{
			// Ext1, field1
			pack.Tag{1, pack.StartGroupType},
			pack.Tag{2, pack.VarintType}, pack.Varint(1000),
			pack.Tag{3, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(10),
			}),
			pack.Tag{1, pack.EndGroupType},
			// Ext2, field1
			pack.Tag{1, pack.StartGroupType},
			pack.Tag{2, pack.VarintType}, pack.Varint(1001),
			pack.Tag{3, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(30),
			}),
			pack.Tag{1, pack.EndGroupType},
			// Ext2, field2
			pack.Tag{1, pack.StartGroupType},
			pack.Tag{2, pack.VarintType}, pack.Varint(1000),
			pack.Tag{3, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{2, pack.VarintType}, pack.Varint(20),
			}),
			pack.Tag{1, pack.EndGroupType},
		}.Marshal(),
	},
	{
		desc: "MessageSet with missing type_id",
		decodeTo: []proto.Message{build(
			&messagesetpb.MessageSet{},
			unknown(pack.Message{
				pack.Tag{1, pack.StartGroupType},
				pack.Tag{3, pack.BytesType}, pack.LengthPrefix(pack.Message{
					pack.Tag{1, pack.VarintType}, pack.Varint(10),
				}),
				pack.Tag{1, pack.EndGroupType},
			}.Marshal()),
		)},
		wire: pack.Message{
			pack.Tag{1, pack.StartGroupType},
			pack.Tag{3, pack.BytesType}, pack.LengthPrefix(pack.Message{
				pack.Tag{1, pack.VarintType}, pack.Varint(10),
			}),
			pack.Tag{1, pack.EndGroupType},
		}.Marshal(),
	},
	{
		desc: "MessageSet with missing message",
		decodeTo: []proto.Message{build(
			&messagesetpb.MessageSet{},
			extend(msetextpb.E_Ext1_MessageSetExtension, &msetextpb.Ext1{}),
		)},
		wire: pack.Message{
			pack.Tag{1, pack.StartGroupType},
			pack.Tag{2, pack.VarintType}, pack.Varint(1000),
			pack.Tag{1, pack.EndGroupType},
		}.Marshal(),
	},
}
