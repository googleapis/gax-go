// Copyright 2016 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gax

import (
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ Retryer = &boRetryer{}

func TestBackofDefault(t *testing.T) {
	backoff := Backoff{}

	max := []time.Duration{1, 2, 4, 8, 16, 30, 30, 30, 30, 30}
	for i, m := range max {
		max[i] = m * time.Second
	}

	for i, w := range max {
		if d := backoff.Pause(); d > w {
			t.Errorf("Backoff duration should be at most %s, got %s", w, d)
		} else if i < len(max)-1 && backoff.cur != max[i+1] {
			t.Errorf("current envelope is %s, want %s", backoff.cur, max[i+1])
		}
	}
}

func TestBackoffExponential(t *testing.T) {
	backoff := Backoff{Initial: 1, Max: 20, Multiplier: 2}
	want := []time.Duration{1, 2, 4, 8, 16, 20, 20, 20, 20, 20}
	for _, w := range want {
		if d := backoff.Pause(); d > w {
			t.Errorf("Backoff duration should be at most %s, got %s", w, d)
		}
	}
}

func TestOnCodes(t *testing.T) {
	// Lint errors grpc.Errorf in 1.6. It mistakenly expects the first arg to Errorf to be a string.
	errf := status.Errorf
	apiErr := errf(codes.Unavailable, "")
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
		b := OnCodes(tst.c, Backoff{})
		if _, retry := b.Retry(apiErr); retry != tst.retry {
			t.Errorf("retriable codes: %v, error: %s, retry: %t, want %t", tst.c, apiErr, retry, tst.retry)
		}
	}
}
