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
	v2 "github.com/googleapis/gax-go/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// CallOption is an option used by Invoke to control behaviors of RPC calls.
// CallOption works by modifying relevant fields of CallSettings.
type CallOption = v2.CallOption

// Retryer is used by Invoke to determine retry behavior.
type Retryer = v2.Retryer

// WithRetry sets CallSettings.Retry to fn.
func WithRetry(fn func() Retryer) CallOption {
	return v2.WithRetry(fn)
}

// OnCodes returns a Retryer that retries if and only if
// the previous attempt returns a GRPC error whose error code is stored in cc.
// Pause times between retries are specified by bo.
//
// bo is only used for its parameters; each Retryer has its own copy.
func OnCodes(cc []codes.Code, bo Backoff) Retryer {
	return v2.OnCodes(cc, bo)
}

// Backoff implements exponential backoff.
// The wait time between retries is a random value between 0 and the "retry envelope".
// The envelope starts at Initial and increases by the factor of Multiplier every retry,
// but is capped at Max.
type Backoff = v2.Backoff

// WithGRPCOptions allows passing gRPC call options during client creation.
func WithGRPCOptions(opt ...grpc.CallOption) CallOption {
	return v2.WithGRPCOptions(opt...)
}

// CallSettings allow fine-grained control over how calls are made.
type CallSettings = v2.CallSettings
