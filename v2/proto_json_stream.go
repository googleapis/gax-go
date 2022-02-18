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

type ProtoJsonStream interface {
	Recv() (proto.Message, error)
	Close() error
}

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

func (s *protoJsonStream) Recv() (proto.Message, error) {
	var raw json.RawMessage
	if err := s.stream.Decode(&raw); err != nil {
		return nil, err
	}
	m := s.typ.New().Interface()
	if err := protojson.Unmarshal(raw, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *protoJsonStream) Close() error {
	return s.reader.Close()
}
