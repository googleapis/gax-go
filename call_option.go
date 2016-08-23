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
	"math/rand"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type CallOption interface {
	Resolve(*CallSettings)
}

type retryFuncOption func() RetryFunc

func (o retryFuncOption) Resolve(s *CallSettings) {
	s.Retry = o
}

func WithRetry(fn func() RetryFunc) CallOption {
	return retryFuncOption(fn)
}

type Backoff struct {
	Initial time.Duration
	Max     time.Duration
	Mult    float64
	Codes   []codes.Code
}

func (r *Backoff) Retry(err error) (time.Duration, bool) {
	c := grpc.Code(err)
	for _, rc := range r.Codes {
		if c == rc {
			return r.pause(), true
		}
	}
	return 0, false
}

func (r *Backoff) pause() time.Duration {
	if r.Initial == 0 {
		r.Initial = time.Second
	}
	if r.Max == 0 {
		r.Max = 30 * time.Second
	}
	if r.Mult == 0 {
		r.Mult = 2
	}
	d := time.Duration(rand.Int63n(int64(r.Initial)))
	r.Initial = time.Duration(float64(r.Initial) * r.Mult)
	if r.Initial > r.Max {
		r.Initial = r.Max
	}
	return d
}

type CallSettings struct {
	// Retry returns a RetryFunc to be used to control retry logic of a method call.
	// If Retry is nil or the returned RetryFunc is nil, the call will not be retried.
	Retry func() RetryFunc
}

// RetryFunc decides whether a request should be retried and how long to wait before retrying
// if the previous attempt returned with err. err is never nil.
type RetryFunc func(error) (time.Duration, bool)
