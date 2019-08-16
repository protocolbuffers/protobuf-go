// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This binary implements the conformance test subprocess protocol as documented
// in conformance.proto.
package main

import (
	"encoding/binary"
	"io"
	"log"
	"os"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"

	pb "google.golang.org/protobuf/internal/testprotos/conformance"
)

func main() {
	var sizeBuf [4]byte
	inbuf := make([]byte, 0, 4096)
	for {
		_, err := io.ReadFull(os.Stdin, sizeBuf[:])
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("conformance: read request: %v", err)
		}
		size := binary.LittleEndian.Uint32(sizeBuf[:])
		if int(size) > cap(inbuf) {
			inbuf = make([]byte, size)
		}
		inbuf = inbuf[:size]
		if _, err := io.ReadFull(os.Stdin, inbuf); err != nil {
			log.Fatalf("conformance: read request: %v", err)
		}

		req := &pb.ConformanceRequest{}
		if err := proto.Unmarshal(inbuf, req); err != nil {
			log.Fatalf("conformance: parse request: %v", err)
		}
		res := handle(req)

		out, err := proto.Marshal(res)
		if err != nil {
			log.Fatalf("conformance: marshal response: %v", err)
		}
		binary.LittleEndian.PutUint32(sizeBuf[:], uint32(len(out)))
		if _, err := os.Stdout.Write(sizeBuf[:]); err != nil {
			log.Fatalf("conformance: write response: %v", err)
		}
		if _, err := os.Stdout.Write(out); err != nil {
			log.Fatalf("conformance: write response: %v", err)
		}
	}
}

func handle(req *pb.ConformanceRequest) (res *pb.ConformanceResponse) {
	var msg proto.Message = &pb.TestAllTypesProto2{}
	if req.GetMessageType() == "protobuf_test_messages.proto3.TestAllTypesProto3" {
		msg = &pb.TestAllTypesProto3{}
	}

	// Unmarshal the test message.
	var err error
	switch p := req.Payload.(type) {
	case *pb.ConformanceRequest_ProtobufPayload:
		err = proto.Unmarshal(p.ProtobufPayload, msg)
	case *pb.ConformanceRequest_JsonPayload:
		err = protojson.UnmarshalOptions{
			DiscardUnknown: req.TestCategory == pb.TestCategory_JSON_IGNORE_UNKNOWN_PARSING_TEST,
		}.Unmarshal([]byte(p.JsonPayload), msg)
	case *pb.ConformanceRequest_TextPayload:
		err = prototext.Unmarshal([]byte(p.TextPayload), msg)
	default:
		return &pb.ConformanceResponse{
			Result: &pb.ConformanceResponse_RuntimeError{
				RuntimeError: "unknown request payload type",
			},
		}
	}
	if err != nil {
		return &pb.ConformanceResponse{
			Result: &pb.ConformanceResponse_ParseError{
				ParseError: err.Error(),
			},
		}
	}

	// Marshal the test message.
	var b []byte
	switch req.RequestedOutputFormat {
	case pb.WireFormat_PROTOBUF:
		b, err = proto.Marshal(msg)
		res = &pb.ConformanceResponse{
			Result: &pb.ConformanceResponse_ProtobufPayload{
				ProtobufPayload: b,
			},
		}
	case pb.WireFormat_JSON:
		b, err = protojson.Marshal(msg)
		res = &pb.ConformanceResponse{
			Result: &pb.ConformanceResponse_JsonPayload{
				JsonPayload: string(b),
			},
		}
	case pb.WireFormat_TEXT_FORMAT:
		b, err = prototext.Marshal(msg)
		res = &pb.ConformanceResponse{
			Result: &pb.ConformanceResponse_TextPayload{
				TextPayload: string(b),
			},
		}
	default:
		return &pb.ConformanceResponse{
			Result: &pb.ConformanceResponse_RuntimeError{
				RuntimeError: "unknown output format",
			},
		}
	}
	if err != nil {
		return &pb.ConformanceResponse{
			Result: &pb.ConformanceResponse_SerializeError{
				SerializeError: err.Error(),
			},
		}
	}
	return res
}
