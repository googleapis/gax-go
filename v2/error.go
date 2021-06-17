// Copyright 2021, Google Inc.
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
	"fmt"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/status"
)

// APIError is a wrapper type for API call error statuses. It extracts
// well-known error detail information from the status and presents it to the
// user in a Go idiomatic way.
//
// Example:
//
//   response, err := client.GetFooRpc()
//   if ae, ok := errors.As(err, *APIError); ok {
//	   // print ae.Details()
//   }
//
// Generally, an APICall should return an APIError if possible.
type APIError struct {
	// I don't like this.
	details *details

	err    error
	status *status.Status
	// Could expand to HTTP errors as well...
}

// TODO: I don't like this
type details struct {
	BadRequest          *errdetails.BadRequest
	ErrorInfo           *errdetails.ErrorInfo
	QuotaFailure        *errdetails.QuotaFailure
	Help                *errdetails.Help
	DebugInfo           *errdetails.DebugInfo
	PreconditionFailure *errdetails.PreconditionFailure
}

func (d *details) String() string {
	s := ""

	// I don't like this.
	if d.ErrorInfo != nil {
		s = fmt.Sprintf("%s: %+v", d.ErrorInfo.GetReason(), d.ErrorInfo.GetMetadata())
	}

	return s
}

// I don't like this.
func d(st *status.Status) *details {
	dets := &details{}
	for _, det := range st.Details() {
		switch detail := det.(type) {
		case *errdetails.ErrorInfo:
			dets.ErrorInfo = detail
		case *errdetails.BadRequest:
			dets.BadRequest = detail
		case *errdetails.QuotaFailure:
			dets.QuotaFailure = detail
		case *errdetails.Help:
			dets.Help = detail
		case *errdetails.DebugInfo:
			dets.DebugInfo = detail
		case *errdetails.PreconditionFailure:
			dets.PreconditionFailure = detail
		}
	}
	return dets
}

// Unwrap returns the original error wrapped by APIError.
func (a *APIError) Unwrap() error {
	return a.err
}

// Error returns the error message.
func (a *APIError) Error() string {
	return a.details.String()
}

func (a *APIError) Details() string {
	return a.details.String()
}

// GRPCStatus returns the gRPC Status parsed from the wrapped error. If the
// original wasn't a gRPC status, this returns nil.
// This enables the use of APIError as input to status.FromError().
func (a *APIError) GRPCStatus() *status.Status {
	return a.status
}

// FromError attempts to extract an API call status from err and wrap it in an
// APIError. If err contains an API call status, an APIError and true are
// returned. If present, the error details are parsed into the APIError. If
// err is nil or it is not an API call status, nil and false are returned.
func FromError(err error) (*APIError, bool) {
	if st, ok := status.FromError(err); ok {
		return &APIError{
			err:     err,
			status:  st,
			details: d(st),
		}, true
	}

	return nil, false
}

func FromStatus(st *status.Status) *APIError {
	return &APIError{
		details: d(st),
		err:     st.Err(),
		status:  st,
	}
}
