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
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func TestExponentialBackoff(t *testing.T) {
	eb := exponentialBackoff{Base: 1, Max: 20, Mult: 2}
	want := []time.Duration{1, 2, 4, 8, 16, 20, 20, 20, 20, 20}
	for _, w := range want {
		if d, ok := eb.Pause(); !ok {
			t.Error("exponentialBackoff should keep pausing forever")
		} else if d > w {
			t.Errorf("Backoff duration should be at most %s, got %s", w, d)
		}
	}
}

func TestBackoffRetryer(t *testing.T) {
	apiErr := grpc.Errorf(codes.Unavailable, "")
	tests := []struct {
		c     []codes.Code
		retry bool
	}{
		{nil, false},
		{[]codes.Code{codes.DeadlineExceeded}, false},
		{[]codes.Code{codes.DeadlineExceeded, codes.Unavailable}, true},
		{[]codes.Code{codes.Unavailable}, true},
	}
	for _, tst := range tests {
		var settings CallSettings
		WithBackoffRetryer(1, 2, 1.2, tst.c...).Resolve(&settings)

		var retry bool
		if settings.Retryer != nil {
			_, retry = settings.Retryer().Retry(apiErr)
		}

		if retry != tst.retry {
			t.Errorf("retryable codes: %v, error code: %s, retry: %t, want %t", tst.c, grpc.Code(apiErr), retry, tst.retry)
		}
	}
}
