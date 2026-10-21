// Copyright 2026, Google Inc.
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
	"errors"
	"testing"

	"google.golang.org/protobuf/proto"
)

func TestValidatePathExpressions(t *testing.T) {
	for _, tc := range []struct {
		desc    string
		in      []any
		wantErr bool
	}{
		{
			desc: "nil input",
		},
		{
			desc: "simple value",
			in: []any{
				func() string { return ".." }(),
			},
			wantErr: true,
		},
		{
			desc: "proto ref string",
			in: []any{
				func() *string { return proto.String(".") }(),
			},
			wantErr: false, // default value representation is reference, not value
		},
		{
			desc: "byte value",
			in: []any{
				func() []byte { return []byte{'.'} }(),
			},
			wantErr: false, // default value representation is encoded (e.g. '[46]')
		},
		{
			desc: "multiple",
			in: []any{
				func() string { return "foo" }(),
				func() int { return 8 }(),
				func() []byte { return []byte{'.'} }(),
				func() string { return ".." }(),
			},
			wantErr: true,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			err := ValidatePathExpressions(tc.in...)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected err, got success")
				}
				if !errors.Is(err, PathExpressionError) {
					t.Errorf("expected PathExpressionError, got %T: %v", err, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected success, got err: %v", err)
				}
			}
		})
	}
}
