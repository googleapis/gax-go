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

// Package gax provides Google API eXtension for Go language.
//
// This will provide utilities and common logic for the generated code of
// the API client, such as:
//    - management of common options
//    - retrying of idempotent API calls
//    - unrolling paginated APIs
package gax

import (
	"errors"
	"io"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var (
	retryableGrpcCodes []codes.Code = []codes.Code{
		codes.DeadlineExceeded,
		// TODO(mukai): Aborted means the client can retry but it's at a
		// higher-level, retrying at individual API call may not always be right.
		codes.Aborted,
	}
)

// APICall is the function type to invoke an actual gRPC call. Used by Invoke.
type APICall func(context.Context, interface{}) (interface{}, error)

func isRetryable(c codes.Code) bool {
	for _, code := range retryableGrpcCodes {
		if c == code {
			return true
		}
	}
	return false
}

// Invoke calls |apiCall| considering optional parameters. It will care retries and timeouts well.
// The entire timeout should be specified in |ctx|, otherwise it will retry the default attempts
// with the default timeout.
func Invoke(ctx context.Context, req interface{}, apiCall APICall, opts ...CallOption) (interface{}, error) {
	return invoke(ctx, req, apiCall, buildCallOpt(opts...))
}

func invoke(ctx context.Context, req interface{}, apiCall APICall, option *callOpt) (resp interface{}, err error) {
	timeout := option.timeout.initial()
	interval := option.retryInterval.initial()
	for attempts := 0; attempts < option.maxAttempts; attempts++ {
		childCtx, _ := context.WithTimeout(ctx, timeout)
		resp, err = apiCall(childCtx, req)
		code := grpc.Code(err)
		if code == codes.OK {
			return resp, nil
		}
		// Don't retry if the parent context is done (i.e. the overall deadline
		// has been exceeded, or it's canceled explicitly from outside).
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if !isRetryable(code) {
			return nil, err
		}
		// Do not invoke time.Sleep() but create a context and wait for its deadline.
		// This allows to finish the interval earlier if the interval is longer than
		// ctx's timeout, which could happen when waiting for resource exhausted.
		intervalCtx, _ := context.WithTimeout(ctx, interval)
		<-intervalCtx.Done()
		// Check if the overall context finished during the interval.
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		timeout = option.timeout.next(timeout)
		interval = option.retryInterval.next(interval)
	}
	return nil, err
}

// The interface which a page-streaming method should implement.
type PageStreamable interface {
	// ApiCall invokes the actual API call with the current request and update the response data
	// and returns its error code.
	ApiCall(ctx context.Context, opts ...CallOption) error

	// Len returns the number of elements in the current response data.
	Len() int

	// GetData returns the i-th element in the current response data.
	GetData(i int) interface{}

	// NextPage updates the page token of the current request from the response data. This should
	// return io.EOF error if it doesn't have the page token anymore, and return other non-nil
	// errors if something goes wrong.
	NextPage() error
}

// PageStream iterates over the elements in the page-streaming data. |iter| is invoked for each
// of the elements, and the iteration will stop when it returns false.
func PageStream(ctx context.Context, streamable PageStreamable, iter func(interface{}) bool, opts ...CallOption) error {
	for {
		// TODO(mukai): tweak callOpt for iteration?
		err := streamable.ApiCall(ctx, opts...)
		// io.EOF might happen when the initial ApiCall reaches to the end for streaming gRPC calls.
		// This means the requested resource is simply empty, therefore return nil instead of errors.
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
		length := streamable.Len()
		for i := 0; i < length; i++ {
			if !iter(streamable.GetData(i)) {
				return nil
			}
		}
		if err := streamable.NextPage(); err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
	}
}

// Head returns the first element from the page stream.
func Head(ctx context.Context, streamable PageStreamable, opts ...CallOption) (result interface{}, err error) {
	called := false
	err = PageStream(ctx, streamable, func(element interface{}) bool {
		called = true
		result = element
		return false
	}, opts...)
	if !called {
		return nil, errors.New("PageStreamable is empty.")
	}
	return result, err
}

// Take returns the first |count| elements in the page stream.
func Take(ctx context.Context, streamable PageStreamable, count int, opts ...CallOption) ([]interface{}, error) {
	if count <= 0 {
		return nil, nil
	}
	result := make([]interface{}, 0, count)
	i := 0
	err := PageStream(ctx, streamable, func(element interface{}) bool {
		result = append(result, element)
		i++
		return i < count
	}, opts...)
	return result, err
}

// ToArray converts the page stream into a single array.
func ToArray(ctx context.Context, streamable PageStreamable, opts ...CallOption) ([]interface{}, error) {
	var result []interface{}
	err := PageStream(ctx, streamable, func(element interface{}) bool {
		result = append(result, element)
		return true
	}, opts...)
	return result, err
}
