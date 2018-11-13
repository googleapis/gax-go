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
	"sync"
	"math/rand"
)

// Backoff implements exponential backoff.
// The wait time between retries is a random value between 0 and the "retry envelope".
// The envelope starts at Initial and increases by the factor of Multiplier every retry,
// but is capped at Max.
type Backoff struct {
	// Initial is the initial value of the retry envelope, defaults to 1 second.
	Initial time.Duration

	// Max is the maximum value of the retry envelope, defaults to 30 seconds.
	Max time.Duration

	// Multiplier is the factor by which the retry envelope increases.
	// It should be greater than 1 and defaults to 2.
	Multiplier float64

	// cur is the current retry envelope.
	mu  *sync.RWMutex
	cur time.Duration
}

// Pause returns the duration that the caller should pause.
func (bo *Backoff) Pause() time.Duration {
	if bo.Initial == 0 {
		bo.Initial = time.Second
	}
	if bo.cur == 0 {
		bo.cur = bo.Initial
	}
	if bo.Max == 0 {
		bo.Max = 30 * time.Second
	}
	if bo.Multiplier < 1 {
		bo.Multiplier = 2
	}
	if bo.mu == nil {
		bo.mu = &sync.RWMutex{}
	}

	bo.mu.Lock()
	defer bo.mu.Unlock()
	// Select a duration between zero and the current max. It might seem counterintuitive to
	// have so much jitter, but https://www.awsarchitectureblog.com/2015/03/backoff.html
	// argues that that is the best strategy.
	d := time.Duration(rand.Int63n(int64(bo.cur)))
	bo.cur = time.Duration(float64(bo.cur) * bo.Multiplier)
	if bo.cur > bo.Max {
		bo.cur = bo.Max
	}
	return d
}

// ResetBackoff resets the current backoff so that the next call to Pause will
// start from the beginning.
func (bo *Backoff) ResetBackoff() {
	bo.mu.Lock()
	defer bo.mu.Unlock()
	bo.cur = 0
}
