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

package internallog

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

	"github.com/googleapis/gax-go/v2/internallog/internal"
	"github.com/googleapis/gax-go/v2/internallog/internal/logtest"
)

// To update conformance tests in this package run `go test -update_golden`
func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

func TestLog_off(t *testing.T) {
	golden := "off.log"
	logger, f := setupLogger(t, golden)
	logger.Error("one")
	logger.Info("two")
	logger.Warn("three")
	logger.Debug("four")
	f.Close()
	logtest.DiffTest(t, f.Name(), golden)
}

func TestLog_envarError(t *testing.T) {
	golden := "envar-error.log"
	t.Setenv(internal.LoggingLevelEnvVar, "eRrOr")
	logger, f := setupLogger(t, golden)
	logger.Error("one")
	logger.Info("two")
	logger.Warn("three")
	logger.Debug("four")
	f.Close()
	logtest.DiffTest(t, f.Name(), golden)
}

func TestLog_envarInfo(t *testing.T) {
	golden := "envar-info.log"
	t.Setenv(internal.LoggingLevelEnvVar, "info")
	logger, f := setupLogger(t, golden)
	logger.Error("one")
	logger.Info("two")
	logger.Warn("three")
	logger.Debug("four")
	f.Close()
	logtest.DiffTest(t, f.Name(), golden)
}

func TestLog_envarWarn(t *testing.T) {
	golden := "envar-warn.log"
	t.Setenv(internal.LoggingLevelEnvVar, "warn")
	logger, f := setupLogger(t, golden)
	logger.Error("one")
	logger.Info("two")
	logger.Warn("three")
	logger.Debug("four")
	f.Close()
	logtest.DiffTest(t, f.Name(), golden)
}

func TestLog_envarDebug(t *testing.T) {
	golden := "envar-debug.log"
	t.Setenv(internal.LoggingLevelEnvVar, "debug")
	logger, f := setupLogger(t, golden)
	logger.Error("one")
	logger.Info("two")
	logger.Warn("three")
	logger.Debug("four")
	f.Close()
	logtest.DiffTest(t, f.Name(), golden)
}

func TestLog_HTTPRequest(t *testing.T) {
	golden := "httpRequest.log"
	t.Setenv(internal.LoggingLevelEnvVar, "debug")
	logger, f := setupLogger(t, golden)
	ctx := context.Background()
	body := []byte(`{"secret":"shh, it's a secret"}`)
	request, err := http.NewRequest(http.MethodPost, "https://example.com", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Add("foo", "bar")
	logger.DebugContext(ctx, "msg", "request", HTTPRequest(request, body))
	f.Close()
	logtest.DiffTest(t, f.Name(), golden)
}

func TestLog_HTTPResponse(t *testing.T) {
	golden := "httpResponse.log"
	t.Setenv(internal.LoggingLevelEnvVar, "debug")
	logger, f := setupLogger(t, golden)
	ctx := context.Background()
	body := []byte(`{"secret":"shh, it's a secret"}`)
	response := &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Foo": []string{"bar"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
	logger.DebugContext(ctx, "msg", "response", HTTPResponse(response, body))
	f.Close()
	logtest.DiffTest(t, f.Name(), golden)
}

func TestLog_HTTPRequest_formData(t *testing.T) {
	golden := "httpRequest-form.log"
	t.Setenv(internal.LoggingLevelEnvVar, "debug")
	logger, f := setupLogger(t, golden)
	ctx := context.Background()
	form := url.Values{}
	form.Add("foo", "bar")
	form.Add("baz", "qux")
	request, err := http.NewRequest(http.MethodPost, "https://example.com", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Add("foo", "bar")
	logger.DebugContext(ctx, "msg", "request", HTTPRequest(request, []byte(form.Encode())))
	f.Close()
	logtest.DiffTest(t, f.Name(), golden)
}

func TestLog_HTTPRequest_jsonArray(t *testing.T) {
	golden := "httpRequest-array.log"
	t.Setenv(internal.LoggingLevelEnvVar, "debug")
	logger, f := setupLogger(t, golden)
	ctx := context.Background()
	body := []byte(`[{"secret":"shh, it's a secret"},{"secret":"and, another"}]`)
	request, err := http.NewRequest(http.MethodPost, "https://example.com", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Add("foo", "bar")
	logger.DebugContext(ctx, "msg", "request", HTTPRequest(request, body))
	f.Close()
	logtest.DiffTest(t, f.Name(), golden)
}

func setupLogger(t *testing.T, golden string) (*slog.Logger, *os.File) {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), golden)
	if err != nil {
		t.Fatal(err)
	}
	logger := internal.NewLoggerWithWriter(f)
	return logger, f
}
