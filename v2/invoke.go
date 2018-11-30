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
	"context"
	"time"
)

// A user defined call stub.
type APICall func(context.Context, CallSettings) error

// Invoke calls the given APICall,
// performing retries as specified by opts, if any.
func Invoke(ctx context.Context, call APICall, opts ...CallOption) error {
	var settings CallSettings
	for _, opt := range opts {
		opt.Resolve(&settings)
	}
	return invoke(ctx, call, settings, Sleep)
}

// Sleep is similar to time.Sleep, but it can be interrupted by ctx.Done() closing.
// If interrupted, Sleep returns ctx.Err().
func Sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	select {
	case <-ctx.Done():
		t.Stop()
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

type sleeper func(ctx context.Context, d time.Duration) error

// invoke implements Invoke, taking an additional sleeper argument for testing.
func invoke(ctx context.Context, call APICall, settings CallSettings, sp sleeper) error {
	var retryer Retryer
	for {
		err := call(ctx, settings)
		if err == nil {
			return nil
		}
		if settings.Retry == nil {
			return err
		}
		if retryer == nil {
			if r := settings.Retry(); r != nil {
				retryer = r
			} else {
				return err
			}
		}
		if d, ok := retryer.Retry(err); !ok {
			return err
		} else if err = sp(ctx, d); err != nil {
			return err
		}
	}
}
