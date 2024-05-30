// Copyright 2024, Google Inc.
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//   - Redistributions of source code must retain the above copyright
//
// notice, this list of conditions and the following disclaimer.
//   - Redistributions in binary form must reproduce the above
//
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//   - Neither the name of Google Inc. nor the names of its
//
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

package cloggrpc

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"testing"

	"github.com/googleapis/gax-go/v2/clog"
	internalclog "github.com/googleapis/gax-go/v2/clog/internal"
	"github.com/googleapis/gax-go/v2/clog/internal/bookpb"
	"github.com/googleapis/gax-go/v2/clog/internal/clogtest"
	"google.golang.org/grpc/metadata"
)

// To update conformance tests in this package run `go test -update_golden`
func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

func TestLog_protoMessageRequestInfo(t *testing.T) {
	golden := "request-info.log"
	logger, f := setupLogger(t, golden, nil)
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("foo", "bar"))
	book := &bookpb.Book{
		Title:  "The book",
		Author: "The author",
	}
	logger.Log(ctx, clog.DynamicLevel(), "msg", "request", ProtoMessageRequest(ctx, book))
	f.Close()
	clogtest.DiffTest(t, f.Name(), golden)
}

func TestLog_protoMessageRequestDebug(t *testing.T) {
	golden := "request-debug.log"
	logger, f := setupLogger(t, golden, slog.LevelDebug)
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("foo", "bar"))
	book := &bookpb.Book{
		Title:  "The book",
		Author: "The author",
	}
	logger.Log(ctx, clog.DynamicLevel(), "msg", "request", ProtoMessageRequest(ctx, book))
	f.Close()
	clogtest.DiffTest(t, f.Name(), golden)
}

func TestLog_protoMessageResponseInfo(t *testing.T) {
	golden := "response-info.log"
	logger, f := setupLogger(t, golden, nil)
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("foo", "bar"))
	book := &bookpb.Book{
		Title:  "The book",
		Author: "The author",
	}
	logger.Log(ctx, clog.DynamicLevel(), "msg", "response", ProtoMessageRequest(ctx, book))
	f.Close()
	clogtest.DiffTest(t, f.Name(), golden)
}

func TestLog_protoMessageResponseDebug(t *testing.T) {
	golden := "response-debug.log"
	logger, f := setupLogger(t, golden, slog.LevelDebug)
	ctx := context.Background()
	book := &bookpb.Book{
		Title:  "The book",
		Author: "The author",
	}
	logger.Log(ctx, clog.DynamicLevel(), "msg", "response", ProtoMessageResponse(book))
	f.Close()
	clogtest.DiffTest(t, f.Name(), golden)
}

func setupLogger(t *testing.T, golden string, leveler slog.Leveler) (*slog.Logger, *os.File) {
	t.Helper()
	internalclog.State = internalclog.LoggerState{}
	f, err := os.CreateTemp(t.TempDir(), golden)
	if err != nil {
		t.Fatal(err)
	}
	clog.SetDefaults(&clog.DefaultOptions{
		EnableLogging: true,
		Writer:        f,
		Level:         leveler,
	})
	logger := clog.New()
	return logger, f
}
