// Copyright 2016, Google Inc.
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
	"bytes"
	"unicode"

	"google.golang.org/grpc/codes"
)

// The relationship between the string name of grpc code to codes.Code
// constants. This is generated in init().
var grpcCodeNames map[string]codes.Code

// Stores the retry config.
type RetryParams struct {
	InitialRetryDelayMillis int64
	RetryDelayMultiplier    float64
	MaxRetryDelayMillis     int64
	InitialRPCTimeoutMillis int64
	RpcTimeoutMultiplier    float64
	MaxRPCTimeoutMillis     int64
	TotalTimeoutMillis      int64
}

// Stores the config for each method.
type MethodConfig struct {
	RetryCodeName   string
	RetryParamsName string
}

// Stores a call config.
type OptionStore struct {
	RetryCodes  map[string][]RetryCode
	RetryParams map[string][]RetryParams
	Methods     map[string]MethodConfig
}

type RetryCode string

// GetGRPCCode returns the codes.Code corresponding to the RetryCode.
func (r RetryCode) GRPCCode() codes.Code {
	if c, ok := grpcCodeNames[string(r)]; ok {
		return c
	}
	return codes.Unknown
}

func camelCaseToUpperUnderscore(s string) string {
	var underscore bytes.Buffer
	previousUpperscore := true
	for _, r := range s {
		// Do not insert underscore for subsequent upper cases.
		// (i.e. "OK" should be "OK", not "O_K").
		if !previousUpperscore && unicode.IsUpper(r) {
			underscore.WriteRune('_')
		}
		underscore.WriteRune(unicode.ToUpper(r))
		previousUpperscore = unicode.IsUpper(r)
	}
	return underscore.String()
}

func init() {
	// Initialize grpcCodeNames from the list of codes.Code and its
	// string representation. String representations are camel-cased
	// in Go, thus convert it into uppercase-underscored constant
	// representation.
	grpcCodeNames = map[string]codes.Code{}
	for c := codes.OK; c <= codes.DataLoss; c++ {
		grpcCodeNames[camelCaseToUpperUnderscore(c.String())] = c
	}
}
