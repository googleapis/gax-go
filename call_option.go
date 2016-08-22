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

type retryerOption func() Retryer

func (o retryerOption) Resolve(s *CallSettings) {
	s.Retryer = o
}

func WithBackoffRetryer(base, max time.Duration, mult float64, codes ...codes.Code) CallOption {
	if len(codes) == 0 {
		return retryerOption(nil)
	}
	return retryerOption(func() Retryer {
		return &codeRetryer{
			b: &exponentialBackoff{Base: base, Max: max, Mult: mult},
			c: codes,
		}
	})
}

type CallSettings struct {
	// Retryer returns a Retryer to be used to control retry logic of a method call.
	// If retry is undesirable, simply set the function to nil.
	// If the function is not nil, it must return non-nil Retryer.
	Retryer func() Retryer
}

type Retryer interface {
	// Retry decides whether a request should be retried and how long to wait before retrying
	// if the previous attempt returned with err. err is never nil.
	Retry(err error) (backoff time.Duration, ok bool)
}

type BackoffStrategy interface {
	Pause() (time.Duration, bool)
}

type codeRetryer struct {
	b BackoffStrategy
	c []codes.Code
}

func (r *codeRetryer) Retry(err error) (time.Duration, bool) {
	c := grpc.Code(err)
	for _, rc := range r.c {
		if rc == c {
			return r.b.Pause()
		}
	}
	return 0, false
}

// exponentialBackoff implements exponential backoff.
// It is similar to gensupport.ExponentialBackoff,
// but allows arbitrary Mult
// and retries forever within Max.
type exponentialBackoff struct {
	Base time.Duration
	Max  time.Duration
	Mult float64

	d time.Duration
}

func (b *exponentialBackoff) Pause() (time.Duration, bool) {
	if b.d < b.Base {
		b.d = b.Base
	}
	d := b.d
	b.d = time.Duration(float64(b.d) * b.Mult)
	if b.d > b.Max {
		b.d = b.Max
	}
	return time.Duration(rand.Int63n(int64(d))), true
}
