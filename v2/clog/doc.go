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

// Package clog exposes configuration and helpers for all logging done in the
// Go Cloud Client libraries.
//
// # Logging Configuration
//
// Enabling logging in the client libraries can be done either by setting
// environment variables or by explicit configuration in code. To turn on
// logging set the environment variable `GOOGLE_SDK_DEBUG_LOGGING` to `true`.
// To get even more detailed logs you may set the environment variable
// `GOOGLE_SDK_DEBUG_LOGGING_GO_LEVEL` to `debug`. Note, setting the logging
// level to `debug` will cause request/response payloads to be logged.
// Additionally, sensitive items like authorization headers will be logged at
// this level as well.
//
// If you want to configure logging in code the following example is equivalent
// to setting both environment variables above:
//
//	clog.SetDefaults(&clog.DefaultOptions{
//		EnableLogging: true,
//		Level:         slog.LevelDebug,
//	})
//
// For more examples of how to configure the loggers used by the Go Cloud Client
// libraries see function examples for [SetDefaults].
//
// # Default Logger
//
// The default logger used by the client libraries has the following
// characteristics if enabled:
//   - The default destination is [os.Stderr].
//   - The default level is [slog.LevelInfo]
//   - The default format is JSON with a handful of keys overwritten so that
//     logs can be easily parsed by Cloud Logging.
package clog
