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
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	internalclog "github.com/googleapis/gax-go/v2/clog/internal"
)

const (
	// determines if logging is enabled
	enabledEnvVar = "GOOGLE_SDK_DEBUG_LOGGING"
	// logging level
	levelEnvVar = "GOOGLE_SDK_DEBUG_LOGGING_GO_LEVEL"

	googLvlKey    = "severity"
	googMsgKey    = "message"
	googSourceKey = "sourceLocation"
	googTimeKey   = "timestamp"
)

// DefaultOptions used to configure global logger settings.
type DefaultOptions struct {
	// Level configures what the log level is. Defaults to [slog.LevelInfo] or
	// the value specified by the environment variable
	// GOOGLE_SDK_DEBUG_LOGGING_GO_LEVEL.
	Level slog.Leveler
	// Writer configures where logs are written to. Defaults to [os.Stderr].
	Writer io.Writer
	// Handler configure the underlying handler used to format the logs. If
	// specified all other options are ignored. Defaults to a
	// [slog.JSONHandler].
	Handler slog.Handler
	// EnableLogging turns on logging. Defaults to false or the value specified
	// by the environment variable GOOGLE_SDK_DEBUG_LOGGING.
	EnableLogging bool
}

// SetDefaults configures all logging that originates from this package. This
// function must be called before any logger are instantiated with [New].
// This function may be called only once, calling it subsequent times will have
// no effect.
func SetDefaults(opts *DefaultOptions) {
	internalclog.State.ConfigureLoggingOnce.Do(func() {
		// Set Logger Defaults
		if opts == nil {
			opts = &DefaultOptions{}
		}
		level := opts.Level
		writer := opts.Writer
		internalclog.State.Handler = opts.Handler

		if level == nil {
			sLevel := strings.ToLower(os.Getenv(levelEnvVar))
			switch sLevel {
			case "debug":
				level = slog.LevelDebug
			case "info":
				level = slog.LevelInfo
			case "warn":
				level = slog.LevelWarn
			case "error":
				level = slog.LevelError
			default:
				level = slog.LevelInfo
			}
		}

		if writer == nil {
			writer = os.Stderr
		}
		if internalclog.State.Handler == nil {
			internalclog.State.Handler = slog.NewJSONHandler(writer, &slog.HandlerOptions{
				AddSource:   true,
				Level:       level,
				ReplaceAttr: replaceAttr,
			})
		}

		// Parse environment variables
		internalclog.State.LoggingEnabled, _ = strconv.ParseBool(os.Getenv(enabledEnvVar))
		// Also honor code settings
		internalclog.State.LoggingEnabled = internalclog.State.LoggingEnabled || opts.EnableLogging

		internalclog.State.Lvl = level.Level()
	})
}

// New returns a new [slog.Logger] configured with the provided [Options]. The
// returned logger will be a noop logger unless the environment variable
// GOOGLE_SDK_DEBUG_LOGGING is set to true. See package documentation for more
// details.
func New() *slog.Logger {
	// configures package defaults
	SetDefaults(nil)
	return slog.New(gcHandler{
		h: internalclog.State.Handler,
	})
}

// HTTPRequest returns a lazily evaluated [slog.LogValuer] for a [http.Request].
func HTTPRequest(req *http.Request, body []byte) slog.LogValuer {
	return &request{
		req:     req,
		payload: body,
	}
}

// HTTPResponse returns a lazily evaluated [slog.LogValuer] for a
// [http.Response].
func HTTPResponse(resp *http.Response, body []byte) slog.LogValuer {
	return &response{
		resp:    resp,
		payload: body,
	}
}

// IsDebugLoggingEnabled reports if the detected/configured level is less than
// or equal to [slog.LevelDebug].
func IsDebugLoggingEnabled() bool {
	return internalclog.State.Lvl <= slog.LevelDebug
}

type request struct {
	req     *http.Request
	payload []byte
}

func (r *request) LogValue() slog.Value {
	if r == nil || r.req == nil {
		return slog.Value{}
	}
	var groupValueAtts []slog.Attr
	groupValueAtts = append(groupValueAtts, slog.String("method", r.req.Method))

	if IsDebugLoggingEnabled() {
		groupValueAtts = append(groupValueAtts, slog.String("url", r.req.URL.String()))

		var headerAttr []slog.Attr
		for k, val := range r.req.Header {
			headerAttr = append(headerAttr, slog.String(k, strings.Join(val, ",")))
		}
		if len(headerAttr) > 0 {
			groupValueAtts = append(groupValueAtts, slog.Any("headers", headerAttr))
		}

		if len(r.payload) > 0 {
			buf := &bytes.Buffer{}
			if err := json.Compact(buf, r.payload); err != nil {
				// Write raw payload incase of error
				buf.Write(r.payload)
			}
			groupValueAtts = append(groupValueAtts, slog.String("payload", buf.String()))
		}
	}
	return slog.GroupValue(groupValueAtts...)
}

type response struct {
	resp    *http.Response
	payload []byte
}

func (r *response) LogValue() slog.Value {
	if r == nil {
		return slog.Value{}
	}
	var groupValueAtts []slog.Attr
	groupValueAtts = append(groupValueAtts, slog.String("status", fmt.Sprint(r.resp.StatusCode)))

	if IsDebugLoggingEnabled() {
		var headerAttr []slog.Attr
		for k, val := range r.resp.Header {
			headerAttr = append(headerAttr, slog.String(k, strings.Join(val, ",")))
		}
		if len(headerAttr) > 0 {
			groupValueAtts = append(groupValueAtts, slog.Any("headers", headerAttr))
		}

		if len(r.payload) > 0 {
			buf := &bytes.Buffer{}
			if err := json.Compact(buf, r.payload); err != nil {
				// Write raw payload incase of error
				buf.Write(r.payload)
			}
			groupValueAtts = append(groupValueAtts, slog.String("payload", buf.String()))
		}
	}
	return slog.GroupValue(groupValueAtts...)
}

// DynamicLevel returns the level things should be logged at in client libraries.
// This is only meant to be used when using logging helpers like [HTTPRequest]
// as they redact certain info at certain levels.
func DynamicLevel() slog.Level {
	if internalclog.State.Lvl <= slog.LevelDebug {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}

// replaceAttr remaps default Go logging keys to match what is expected in
// cloud logging.
func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if groups == nil {
		if a.Key == slog.LevelKey {
			a.Key = googLvlKey
			return a
		} else if a.Key == slog.MessageKey {
			a.Key = googMsgKey
			return a
		} else if a.Key == slog.SourceKey {
			a.Key = googSourceKey
			return a
		} else if a.Key == slog.TimeKey {
			a.Key = googTimeKey
			if a.Value.Kind() == slog.KindTime {
				a.Value = slog.StringValue(a.Value.Time().Format(time.RFC3339))
			}
			return a
		}
	}
	return a
}

type gcHandler struct {
	h slog.Handler
}

// Enabled determines if logging should be enabled in the Go Cloud SDK by checking
// if:
//   - GOOGLE_SDK_DEBUG_LOGGING` is true
//   - the log level should be logged
//   - the system is configured to log
func (g gcHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return internalclog.State.LoggingEnabled && g.h.Enabled(ctx, level)
}

func (g gcHandler) Handle(ctx context.Context, r slog.Record) error {
	return g.h.Handle(ctx, r)
}

func (g gcHandler) WithAttrs(a []slog.Attr) slog.Handler { return g.h.WithAttrs(a) }

func (g gcHandler) WithGroup(name string) slog.Handler { return g.h.WithGroup(name) }
