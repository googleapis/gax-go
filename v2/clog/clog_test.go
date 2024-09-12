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

package clog

import (
	"bytes"
	"context"
	"flag"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	internalclog "github.com/googleapis/gax-go/v2/clog/internal"
	"github.com/googleapis/gax-go/v2/clog/internal/clogtest"
)

// To update conformance tests in this package run `go test -update_golden`
func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

func TestLog_basic(t *testing.T) {
	golden := "basic.log"
	logger, f := setupLogger(t, golden, nil)
	logger.Info("one")
	logger.Info("two")
	logger.Info("three")
	f.Close()
	clogtest.DiffTest(t, f.Name(), golden)
}

func TestLog_noDebug(t *testing.T) {
	golden := "no-debug.log"
	logger, f := setupLogger(t, golden, nil)
	logger.Info("one")
	logger.Debug("two")
	logger.Info("three")
	f.Close()
	clogtest.DiffTest(t, f.Name(), golden)
}

func TestLog_withDebug(t *testing.T) {
	golden := "with-debug.log"
	logger, f := setupLogger(t, golden, slog.LevelDebug)
	logger.Info("one")
	logger.Debug("two")
	logger.Info("three")
	f.Close()
	clogtest.DiffTest(t, f.Name(), golden)
}

func TestLog_envarError(t *testing.T) {
	golden := "envar-error.log"
	t.Setenv(levelEnvVar, "eRrOr")
	logger, f := setupLogger(t, golden, nil)
	logger.Error("one")
	logger.Info("two")
	logger.Warn("three")
	logger.Debug("four")
	f.Close()
	clogtest.DiffTest(t, f.Name(), golden)
}

func TestLog_envarInfo(t *testing.T) {
	golden := "envar-info.log"
	t.Setenv(levelEnvVar, "info")
	logger, f := setupLogger(t, golden, nil)
	logger.Error("one")
	logger.Info("two")
	logger.Warn("three")
	logger.Debug("four")
	f.Close()
	clogtest.DiffTest(t, f.Name(), golden)
}

func TestLog_envarWarn(t *testing.T) {
	golden := "envar-warn.log"
	t.Setenv(levelEnvVar, "warn")
	logger, f := setupLogger(t, golden, nil)
	logger.Error("one")
	logger.Info("two")
	logger.Warn("three")
	logger.Debug("four")
	f.Close()
	clogtest.DiffTest(t, f.Name(), golden)
}

func TestLog_envarDebug(t *testing.T) {
	golden := "envar-debug.log"
	t.Setenv(levelEnvVar, "debug")
	logger, f := setupLogger(t, golden, nil)
	logger.Error("one")
	logger.Info("two")
	logger.Warn("three")
	logger.Debug("four")
	f.Close()
	clogtest.DiffTest(t, f.Name(), golden)
}

func TestLog_dynamicHTTPRequest_info(t *testing.T) {
	golden := "httpRequest-info.log"
	logger, f := setupLogger(t, golden, nil)
	ctx := context.Background()
	body := []byte(`{"secret":"shh, it's a secret"}`)
	request, err := http.NewRequest(http.MethodPost, "https://example.com", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Add("foo", "bar")
	logger.Log(ctx, DynamicLevel(), "msg", "request", HTTPRequest(request, body))
	f.Close()
	clogtest.DiffTest(t, f.Name(), golden)
}

func TestLog_dynamicHTTPRequest_debug(t *testing.T) {
	golden := "httpRequest-debug.log"
	logger, f := setupLogger(t, golden, slog.LevelDebug)
	ctx := context.Background()
	body := []byte(`{"secret":"shh, it's a secret"}`)
	request, err := http.NewRequest(http.MethodPost, "https://example.com", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Add("foo", "bar")
	logger.Log(ctx, DynamicLevel(), "msg", "request", HTTPRequest(request, body))
	f.Close()
	clogtest.DiffTest(t, f.Name(), golden)
}

func TestLog_dynamicHTTPResponse_info(t *testing.T) {
	golden := "httpResponse-info.log"
	logger, f := setupLogger(t, golden, nil)
	ctx := context.Background()
	body := []byte(`{"secret":"shh, it's a secret"}`)
	response := &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Foo": []string{"bar"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
	logger.Log(ctx, DynamicLevel(), "msg", "response", HTTPResponse(response, body))
	f.Close()
	clogtest.DiffTest(t, f.Name(), golden)
}

func TestLog_dynamicHTTPResponse_debug(t *testing.T) {
	golden := "httpResponse-debug.log"
	logger, f := setupLogger(t, golden, slog.LevelDebug)
	ctx := context.Background()
	body := []byte(`{"secret":"shh, it's a secret"}`)
	response := &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Foo": []string{"bar"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
	logger.Log(ctx, DynamicLevel(), "msg", "response", HTTPResponse(response, body))
	f.Close()
	clogtest.DiffTest(t, f.Name(), golden)
}

func TestLog_HTTPRequest_formData(t *testing.T) {
	golden := "httpRequest-form.log"
	logger, f := setupLogger(t, golden, slog.LevelDebug)
	ctx := context.Background()
	form := url.Values{}
	form.Add("foo", "bar")
	form.Add("baz", "qux")
	request, err := http.NewRequest(http.MethodPost, "https://example.com", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Add("foo", "bar")
	logger.Log(ctx, DynamicLevel(), "msg", "request", HTTPRequest(request, []byte(form.Encode())))
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
	SetDefaults(&DefaultOptions{
		EnableLogging: true,
		Writer:        f,
		Level:         leveler,
	})
	logger := New()
	return logger, f
}
