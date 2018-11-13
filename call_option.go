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
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CallOption is an option used by Invoke to control behaviors of RPC calls.
// CallOption works by modifying relevant fields of CallSettings.
type CallOption interface {
	// Resolve applies the option by modifying cs.
	Resolve(cs *CallSettings)
}

// Retryer is used by Invoke to determine retry behavior.
type Retryer interface {
	// Retry reports whether a request should be retriedand how long to pause before retrying
	// if the previous attempt returned with err. Invoke never calls Retry with nil error.
	Retry(err error) (pause time.Duration, shouldRetry bool)
}

type retryerOption func() Retryer

func (o retryerOption) Resolve(s *CallSettings) {
	s.Retry = o
}

// WithRetry sets CallSettings.Retry to fn.
func WithRetry(fn func() Retryer) CallOption {
	return retryerOption(fn)
}

// OnCodes returns a Retryer that retries if and only if
// the previous attempt returns a GRPC error whose error code is stored in cc.
// Pause times between retries are specified by bo.
//
// bo is only used for its parameters; each Retryer has its own copy.
func OnCodes(cc []codes.Code, bo Backoff) Retryer {
	return &boRetryer{
		backoff: bo,
		codes:   append([]codes.Code(nil), cc...),
	}
}

type boRetryer struct {
	backoff Backoff
	codes   []codes.Code
}

func (r *boRetryer) Retry(err error) (time.Duration, bool) {
	st, ok := status.FromError(err)
	if !ok {
		return 0, false
	}
	c := st.Code()
	for _, rc := range r.codes {
		if c == rc {
			return r.backoff.Pause(), true
		}
	}
	return 0, false
}

type grpcOpt []grpc.CallOption

func (o grpcOpt) Resolve(s *CallSettings) {
	s.GRPC = o
}

func WithGRPCOptions(opt ...grpc.CallOption) CallOption {
	return grpcOpt(append([]grpc.CallOption(nil), opt...))
}

type CallSettings struct {
	// Retry returns a Retryer to be used to control retry logic of a method call.
	// If Retry is nil or the returned Retryer is nil, the call will not be retried.
	Retry func() Retryer

	// CallOptions to be forwarded to GRPC.
	GRPC []grpc.CallOption
}
