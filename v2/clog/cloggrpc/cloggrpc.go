// Copyright 2024, Google Inc.
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

// Package cloggrpc provided helpers for creating log messages from
// gRPC/protobuf libraries. This package is intended to only be used by Google
// client code and not end-users.
package cloggrpc

import (
	"context"
	"log/slog"
	"strings"

	"github.com/googleapis/gax-go/v2/clog"
	internalclog "github.com/googleapis/gax-go/v2/clog/internal"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ProtoMessageRequest returns a lazily evaluated [slog.LogValuer] for
// the provided message. The context is used to extract outgoing headers.
func ProtoMessageRequest(ctx context.Context, msg proto.Message) slog.LogValuer {
	return &protoMessage{ctx: ctx, msg: msg}
}

// ProtoMessageResponse returns a lazily evaluated [slog.LogValuer] for
// the provided message.
func ProtoMessageResponse(msg proto.Message) slog.LogValuer {
	return &protoMessage{msg: msg}
}

type protoMessage struct {
	ctx context.Context
	msg proto.Message
}

func (m *protoMessage) LogValue() slog.Value {
	if m == nil || m.msg == nil {
		return slog.Value{}
	}

	var groupValueAtts []slog.Attr

	if clog.IsDebugLoggingEnabled() {
		if m.ctx != nil {
			var headerAttr []slog.Attr
			if m, ok := metadata.FromOutgoingContext(m.ctx); ok {
				for k, v := range m {
					headerAttr = append(headerAttr, slog.String(k, strings.Join(v, ",")))
				}
			}
			groupValueAtts = append(groupValueAtts, slog.Any("headers", headerAttr))
		}
		b, _ := protojson.MarshalOptions{AllowPartial: true, UseEnumNumbers: true}.Marshal(m.msg)
		groupValueAtts = append(groupValueAtts, slog.String("payload", string(b)))
	} else {
		return slog.StringValue(internalclog.RedactedValue)
	}
	return slog.GroupValue(groupValueAtts...)
}
