// Copyright 2022, Google Inc.
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//     * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package gax

import (
	"context"
	"encoding/json"
	"io"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ProtoJsonStream represents an interface for consuming a stream of protobuf
// messages encoded using protobuf-JSON format. More information on this format
// can be found at https://developers.google.com/protocol-buffers/docs/proto3#json.
type ProtoJsonStream interface {
	Recv() (proto.Message, error)
	Close() error
}

// NewProtoJsonStream accepts a stream of bytes via an io.ReadCloser that are
// protobuf-JSON encoded protobuf messages of the given type. The ProtoJsonStream
// must be closed when done.
func NewProtoJsonStream(ctx context.Context, rc io.ReadCloser, typ protoreflect.MessageType) ProtoJsonStream {
	return &protoJsonStream{
		ctx:    ctx,
		reader: rc,
		stream: json.NewDecoder(rc),
		typ:    typ,
	}
}

type protoJsonStream struct {
	ctx    context.Context
	reader io.ReadCloser
	stream *json.Decoder
	typ    protoreflect.MessageType
}

// Recv decodes the next protobuf message in the stream or returns io.EOF if
// the stream is done.
func (s *protoJsonStream) Recv() (proto.Message, error) {
	// Capture the next data for the item (a JSON object) in the stream.
	var raw json.RawMessage
	if err := s.stream.Decode(&raw); err != nil {
		return nil, err
	}
	// Initialize a new instance of the protobuf message to unmarshal the
	// raw data into.
	m := s.typ.New().Interface()
	if err := protojson.Unmarshal(raw, m); err != nil {
		return nil, err
	}
	return m, nil
}

// Close closes the stream so that resources are cleaned up.
func (s *protoJsonStream) Close() error {
	return s.reader.Close()
}
