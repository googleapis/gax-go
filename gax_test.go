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
	"io"
	"testing"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var timeUnit time.Duration = time.Millisecond / 2

func getTestOptions() []CallOption {
	return []CallOption{
		WithTimeoutInfo(timeUnit, timeUnit, 1),
		WithIntervalInfo(timeUnit, timeUnit, 1),
	}
}

func TestInvoke(t *testing.T) {
	testOptions := getTestOptions()
	defaultOpt := defaultCallOpt()
	invokeTestCases := []struct {
		expectedCalls     int
		expectedErrorCode codes.Code
		apiCall           func(callCount int) error
	}{
		{1, codes.OK, func(callCount int) error { return nil }},
		{2, codes.OK, func(callCount int) error {
			if callCount < 2 {
				return grpc.Errorf(codes.DeadlineExceeded, "")
			}
			return nil
		}},
		{defaultOpt.maxAttempts, codes.DeadlineExceeded, func(callCount int) error {
			if callCount < defaultOpt.maxAttempts+1 {
				return grpc.Errorf(codes.DeadlineExceeded, "")
			}
			return nil
		}},
		{1, codes.InvalidArgument, func(callCount int) error { return grpc.Errorf(codes.InvalidArgument, "") }},
	}
	for i, testCase := range invokeTestCases {
		callCount := 0
		_, err := Invoke(context.Background(), nil, func(ctx context.Context, req interface{}) (interface{}, error) {
			callCount++
			return nil, testCase.apiCall(callCount)
		}, testOptions...)
		if callCount != testCase.expectedCalls {
			t.Errorf("[%d]: Invoke doesn't call the function properly: %d", i, callCount)
		}
		if grpc.Code(err) != testCase.expectedErrorCode {
			t.Errorf("[%d]: Invoke returns error %v but expected error code is %v", i, err, testCase.expectedErrorCode)
		}
	}
}

func waitForContext(t *testing.T, ctx context.Context) error {
	if _, ok := ctx.Deadline(); !ok {
		t.Errorf("child context should have deadline")
		return grpc.Errorf(codes.InvalidArgument, "")
	}
	<-ctx.Done()
	return grpc.Errorf(codes.DeadlineExceeded, "")
}

func TestInvokeTimeout(t *testing.T) {
	testOptions := getTestOptions()
	opt := defaultCallOpt()
	testCases := []struct {
		parentTimeout         float64
		expectedCalls         int
		parentFailureExpected bool
		expectedErrorCode     codes.Code
	}{
		{10, opt.maxAttempts, false, codes.DeadlineExceeded},
		{3, 2, true, codes.OK},
		{1.5, 1, true, codes.OK},
		{0.25, 1, true, codes.OK},
	}

	callCount := 0
	waitFunc := func(ctx context.Context, req interface{}) (interface{}, error) {
		callCount++
		return nil, waitForContext(t, ctx)
	}

	for i, testCase := range testCases {
		callCount = 0
		parent_context, _ := context.WithTimeout(context.Background(), time.Duration(testCase.parentTimeout*float64(timeUnit)))
		_, err := Invoke(parent_context, nil, waitFunc, testOptions...)
		if callCount != testCase.expectedCalls {
			t.Errorf("[%d]: Invoke() invokes the function %d times (expected: %d)", i, callCount, testCase.expectedCalls)
		}
		if testCase.parentFailureExpected {
			if err == nil || err != parent_context.Err() {
				t.Errorf("[%d]: The overall context for Invoke() failure is expected, but got %v (ovreall context error is %v", i, err, parent_context.Err())
			}
		} else {
			if grpc.Code(err) != testCase.expectedErrorCode {
				t.Errorf("[%d]: Error code %v is expected, but got an error %v", testCase.expectedErrorCode, err)
			}
		}
	}
}

func TestLongInterval(t *testing.T) {
	testOptions := getTestOptions()
	testOptions = append(testOptions, WithIntervalInfo(time.Minute, 2*time.Minute, 1.0))
	waitFunc := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, waitForContext(t, ctx)
	}
	testCases := []func() error{
		func() error {
			parent_context, _ := context.WithTimeout(context.Background(), time.Duration(1.5*float64(timeUnit)))
			_, err := Invoke(parent_context, nil, waitFunc, testOptions...)
			return err
		},
		func() error {
			parent_context, cancel := context.WithCancel(context.Background())
			time.AfterFunc(time.Duration(1.5*float64(timeUnit)), cancel)
			_, err := Invoke(parent_context, nil, waitFunc, testOptions...)
			return err
		},
	}

	for i, testCase := range testCases {
		start := time.Now()
		err := testCase()
		callDuration := time.Since(start)
		if err == nil {
			t.Errorf("[%d]: Invoke() succeeded unexpectedly.", i)
		}
		if callDuration >= time.Minute {
			t.Errorf("[%d]: Invoke() should cancel when the parent is cancelled, but it waits for %v", i, callDuration)
		}
	}
}

type testingPageStreamable struct {
	values [][]int
	i      int
}

func (streamable *testingPageStreamable) ApiCall(ctx context.Context, opts ...CallOption) error {
	if streamable.i >= len(streamable.values) {
		return grpc.Errorf(codes.NotFound, "")
	}
	return nil
}

func (streamable *testingPageStreamable) Len() int {
	return len(streamable.values[streamable.i])
}

func (streamable *testingPageStreamable) GetData(i int) interface{} {
	return streamable.values[streamable.i][i]
}

func (streamable *testingPageStreamable) NextPage() error {
	streamable.i++
	if streamable.i >= len(streamable.values) {
		return io.EOF
	}
	return nil
}
func TestPageStreaming(t *testing.T) {
	testOptions := getTestOptions()
	streamable := &testingPageStreamable{
		values: [][]int{[]int{0, 1, 2, 3}, []int{4, 5}, []int{6}},
	}
	found := map[int]bool{}
	err := PageStream(context.Background(), streamable, func(obj interface{}) bool {
		found[obj.(int)] = true
		return true
	}, testOptions...)
	if err != nil {
		t.Errorf("PageStream() fails unexpectedly: %v", err)
	}
	for i := 0; i <= 6; i++ {
		if !found[i] {
			t.Errorf("PageStream() skips data: %v", i)
		}
	}

	streamable.i = 0
	callCount := 0
	err = PageStream(context.Background(), streamable, func(obj interface{}) bool {
		callCount++
		return callCount < 3
	}, testOptions...)
	if err != nil {
		t.Errorf("PageStream() fails unexpectedly: %v", err)
	}
	if callCount != 3 {
		t.Errorf("PageStream() failed to stop when iter function returns false: %v", callCount)
	}
}

func checkIntList(lst []interface{}, expectedLength int, t *testing.T) {
	if len(lst) != expectedLength {
		t.Errorf("Unexpected length of the list: %d vs %d (%v)", len(lst), expectedLength, lst)
		return
	}
	// Assumes lst is a list of integers of 0, 1, 2, 3...
	for i := 0; i < len(lst); i++ {
		if x, ok := lst[i].(int); !ok || x != i {
			t.Errorf("%d-th element is unexpected: %v", i, lst)
		}
	}
}

func TestHead(t *testing.T) {
	testOptions := getTestOptions()
	testCases := []struct {
		values          [][]int
		expectedFailure bool
		expected        int
	}{
		{[][]int{[]int{0, 1, 2, 3}, []int{4, 5}, []int{6}}, false, 0},
		{[][]int{[]int{}}, true, 0},
		{[][]int{}, true, 0},
	}
	for i, testCase := range testCases {
		obj, err := Head(context.Background(), &testingPageStreamable{values: testCase.values}, testOptions...)
		if testCase.expectedFailure {
			if err == nil {
				t.Errorf("[%d] Head expected failure but returns no error.", i)
			}
		} else {
			if i, ok := obj.(int); !ok {
				t.Errorf("Head returns wrong typed value: %v", obj)
			} else if i != testCase.expected {
				t.Errorf("Head returns %d but expected %d", i, testCase.expected)
			}
		}
	}
}

func TestTake(t *testing.T) {
	testOptions := getTestOptions()
	streamable := &testingPageStreamable{
		values: [][]int{[]int{0, 1, 2, 3}, []int{4, 5}, []int{6}},
	}
	testCases := []struct {
		length         int
		expectedLength int
	}{
		{5, 5},
		{15, 7},
		{0, 0},
		{-1, 0},
	}
	for _, testCase := range testCases {
		streamable.i = 0
		lst, _ := Take(context.Background(), streamable, testCase.length, testOptions...)
		checkIntList(lst, testCase.expectedLength, t)
	}
}

func TestToArray(t *testing.T) {
	testOptions := getTestOptions()
	streamable := &testingPageStreamable{
		values: [][]int{[]int{0, 1, 2, 3}, []int{4, 5}, []int{6}},
	}
	lst, _ := ToArray(context.Background(), streamable, testOptions...)
	checkIntList(lst, 7, t)
}
