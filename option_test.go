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
)

func TestMultipliableDuration(t *testing.T) {
	data := &multipliableDuration{initialDuration: time.Second, maxDuration: 3 * time.Second, multiplier: 1.5}
	duration := data.initial()
	if duration != data.initialDuration {
		t.Errorf("Failed to get the initial duration.")
	}
	duration = data.next(duration)
	if duration != time.Second+time.Millisecond*500 {
		t.Errorf("Failed to compute the next duration: %v", duration)
		return
	}

	nextDuration := data.next(duration)
	if nextDuration < duration {
		t.Errorf("next duration [%v] should be bigger than previous [%v]", nextDuration, duration)
	}

	if nextMax := data.next(data.maxDuration); nextMax != data.maxDuration {
		t.Errorf("next() shouldn't exceed the maximum value: %v vs %v", nextMax, data.maxDuration)
	}

	data = &multipliableDuration{initialDuration: 2 * time.Second, maxDuration: time.Second, multiplier: 1.5}
	duration = data.initial()
	if duration > data.maxDuration {
		t.Errorf("The duration should be capped by the maximum value: %v", duration)
	}
}

func withDuration(f func(time.Duration) CallOption) func(interface{}) CallOption {
	return func(obj interface{}) CallOption {
		return f(obj.(time.Duration))
	}
}

func withFloat64(f func(float64) CallOption) func(interface{}) CallOption {
	return func(obj interface{}) CallOption {
		return f(obj.(float64))
	}
}

func withMultipliable(f func(time.Duration, time.Duration, float64) CallOption) func(interface{}) CallOption {
	return func(obj interface{}) CallOption {
		m := obj.(multipliableDuration)
		return f(m.initialDuration, m.maxDuration, m.multiplier)
	}
}

func TestCallOptions(t *testing.T) {
	opt := defaultCallOpt()
	testCases := []struct {
		msg        string
		value      interface{}
		optCreator func(interface{}) CallOption
		getValue   func(opt *callOpt) interface{}
	}{
		{"WithMaxAttempts", opt.maxAttempts + 1, func(obj interface{}) CallOption {
			return WithMaxAttempts(obj.(int))
		}, func(opt *callOpt) interface{} { return opt.maxAttempts }},
		{"WithTimout", opt.timeout.initialDuration + time.Second, withDuration(WithTimeout), func(opt *callOpt) interface{} {
			return opt.timeout.initialDuration
		}},
		{"WithMaxTimeout", opt.timeout.maxDuration + time.Second, withDuration(WithMaxTimeout), func(opt *callOpt) interface{} {
			return opt.timeout.maxDuration
		}},
		{"WithTimeoutMultiplier", opt.timeout.multiplier + 1, withFloat64(WithTimeoutMultiplier), func(opt *callOpt) interface{} {
			return opt.timeout.multiplier
		}},
		{"WithRetryInterval", opt.retryInterval.initialDuration + time.Second, withDuration(WithRetryInterval), func(opt *callOpt) interface{} {
			return opt.retryInterval.initialDuration
		}},
		{"WithMaxInterval", opt.retryInterval.maxDuration + time.Second, withDuration(WithMaxInterval), func(opt *callOpt) interface{} {
			return opt.retryInterval.maxDuration
		}},
		{"WithIntervalMultiplier", opt.retryInterval.multiplier + 1, withFloat64(WithIntervalMultiplier), func(opt *callOpt) interface{} {
			return opt.retryInterval.multiplier
		}},
		{"WithTimeoutInfo", multipliableDuration{
			opt.timeout.initialDuration + time.Second,
			opt.timeout.maxDuration + 2*time.Second,
			opt.timeout.multiplier + 3,
		}, withMultipliable(WithTimeoutInfo), func(opt *callOpt) interface{} { return opt.timeout }},
		{"WithIntervalInfo", multipliableDuration{
			opt.retryInterval.initialDuration + time.Second,
			opt.retryInterval.maxDuration + 2*time.Second,
			opt.retryInterval.multiplier + 3,
		}, withMultipliable(WithIntervalInfo), func(opt *callOpt) interface{} { return opt.retryInterval }},
	}

	for _, testCase := range testCases {
		builtOpt := buildCallOpt(testCase.optCreator(testCase.value))
		actualValue := testCase.getValue(builtOpt)
		if actualValue != testCase.value {
			t.Errorf("%s set the value, got [%v] but expected [%v]", testCase.msg, actualValue, testCase.value)
		}
	}
}

func TestCallOptionOrder(t *testing.T) {
	opt := defaultCallOpt()
	withNewMaxAttempt1 := WithMaxAttempts(opt.maxAttempts + 1)
	withNewMaxAttempt2 := WithMaxAttempts(opt.maxAttempts + 2)

	if builtOpt := buildCallOpt(withNewMaxAttempt1, withNewMaxAttempt2); builtOpt.maxAttempts != opt.maxAttempts+2 {
		t.Errorf("Failed to build max attempts: %v", builtOpt.maxAttempts)
	}

	if builtOpt := buildCallOpt(withNewMaxAttempt2, withNewMaxAttempt1); builtOpt.maxAttempts != opt.maxAttempts+1 {
		t.Errorf("Failed to build max attempts: %v", builtOpt.maxAttempts)
	}
}
