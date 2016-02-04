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

import "time"

// CallOption holds properties passed to individual invocations of the API calls.
type CallOption interface {
	Resolve(*callOpt)
}

// WithMaxAttempts specifies the maximum number of retries if it fails temporarily.
func WithMaxAttempts(attempts int) CallOption {
	return withMaxAttempts(attempts)
}

type withMaxAttempts int

func (w withMaxAttempts) Resolve(o *callOpt) {
	o.maxAttempts = int(w)
}

// WithTimeout specifies the timeout duration for each API invocation attempt.
func WithTimeout(timeout time.Duration) CallOption {
	return withTimeout(timeout)
}

type withTimeout time.Duration

func (w withTimeout) Resolve(o *callOpt) {
	o.timeout.initialDuration = time.Duration(w)
}

// WithMaxTimeout specifies the maximum timeout duration for each API invocation attempt.
func WithMaxTimeout(max time.Duration) CallOption {
	return withMaxTimeout(max)
}

type withMaxTimeout time.Duration

func (w withMaxTimeout) Resolve(o *callOpt) {
	o.timeout.maxDuration = time.Duration(w)
}

// WithTimeoutMultiplier specifies the multiplier to increase the timeout of further retries.
func WithTimeoutMultiplier(mult float64) CallOption {
	return withTimeoutMultiplier(mult)
}

type withTimeoutMultiplier float64

func (w withTimeoutMultiplier) Resolve(o *callOpt) {
	o.timeout.multiplier = float64(w)
}

type callOptions []CallOption

func (opts callOptions) Resolve(o *callOpt) {
	for _, opt := range opts {
		opt.Resolve(o)
	}
}

// WithTimeoutInfo specifies the initial timeout, maximum timeout, and the mutliplier
// at the same time.
func WithTimeoutInfo(initial time.Duration, max time.Duration, mult float64) CallOption {
	return callOptions([]CallOption{WithTimeout(initial), WithMaxTimeout(max), WithTimeoutMultiplier(mult)})
}

// WithRetryInterval specifies the interval between API invocation attempts.
func WithRetryInterval(interval time.Duration) CallOption {
	return withInterval(interval)
}

type withInterval time.Duration

func (w withInterval) Resolve(o *callOpt) {
	o.retryInterval.initialDuration = time.Duration(w)
}

// WithMaxInterval specifies the maximum interval between API invocation attempts.
func WithMaxInterval(max time.Duration) CallOption {
	return withMaxInterval(max)
}

type withMaxInterval time.Duration

func (w withMaxInterval) Resolve(o *callOpt) {
	o.retryInterval.maxDuration = time.Duration(w)
}

// WithIntervalMultiplier specifies the multiplier to increase the interval between
// API invocation attempts.
func WithIntervalMultiplier(mult float64) CallOption {
	return withIntervalMultiplier(mult)
}

type withIntervalMultiplier float64

func (w withIntervalMultiplier) Resolve(o *callOpt) {
	o.retryInterval.multiplier = float64(w)
}

// WithIntervalInfo specifies the initial interval, maximum interval, and multipliers of
// the interval between API invocation attempts at the same time.
func WithIntervalInfo(initial time.Duration, max time.Duration, mult float64) CallOption {
	return callOptions([]CallOption{WithRetryInterval(initial), WithMaxInterval(max), WithIntervalMultiplier(mult)})
}

type multipliableDuration struct {
	initialDuration time.Duration
	maxDuration     time.Duration
	multiplier      float64
}

func (m multipliableDuration) initial() time.Duration {
	if m.initialDuration > m.maxDuration {
		m.initialDuration = m.maxDuration
	}
	return m.initialDuration
}

func (m multipliableDuration) next(duration time.Duration) time.Duration {
	if duration < m.initial() {
		return m.initial()
	}
	duration = time.Duration(float64(duration) * m.multiplier)
	if duration > m.maxDuration {
		return m.maxDuration
	}
	return duration
}

// callOpt is the struct to hold the properties for individual invocations in a single place.
type callOpt struct {
	maxAttempts   int
	timeout       multipliableDuration
	retryInterval multipliableDuration
}

func defaultCallOpt() *callOpt {
	return &callOpt{
		maxAttempts: 3,
		timeout: multipliableDuration{
			initialDuration: 3 * time.Second,
			maxDuration:     10 * time.Second,
			multiplier:      1.2,
		},
		retryInterval: multipliableDuration{
			initialDuration: 10 * time.Millisecond,
			maxDuration:     time.Second,
			multiplier:      1.2,
		},
	}
}

func buildCallOpt(opts ...CallOption) *callOpt {
	callOpt := defaultCallOpt()
	callOptions(opts).Resolve(callOpt)
	return callOpt
}
