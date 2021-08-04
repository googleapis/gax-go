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

/*
Package apierror implements a wrapper error for parsing error details from
gRPC calls. Currently, only errors representing a gRPC status are supported.
*/
package apierror

import (
	"fmt"
	"strings"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/status"
)

// ErrDetails holds the google/rpc/error_details.proto messages.
type ErrDetails struct {
	ErrorInfo           *errdetails.ErrorInfo
	BadRequest          *errdetails.BadRequest
	PreconditionFailure *errdetails.PreconditionFailure
	QuotaFailure        *errdetails.QuotaFailure
	RetryInfo           *errdetails.RetryInfo
	ResourceInfo        *errdetails.ResourceInfo
	RequestInfo         *errdetails.RequestInfo
	DebugInfo           *errdetails.DebugInfo
	Help                *errdetails.Help
	LocalizedMessage    *errdetails.LocalizedMessage

	// Unknown stores unidentifiable error details.
	Unknown []interface{}
}

// APIError wraps a gRPC Status error. It implements error and Status.
type APIError struct {
	err     error
	status  *status.Status
	details ErrDetails
}

// Details presents the error details of the APIError.
func (a *APIError) Details() ErrDetails {
	return a.details
}

// Unwrap extracts the original error.
func (a *APIError) Unwrap() error {
	return a.err
}

// Error returns a readable representation of the APIError.
func (a *APIError) Error() string {
	var d strings.Builder
	d.WriteString(a.err.Error() + "\n")

	if a.details.ErrorInfo != nil {
		d.WriteString(fmt.Sprintf("error details: name = ErrorInfo reason = %s domain = %s metadata = %s\n",
			a.details.ErrorInfo.GetReason(), a.details.ErrorInfo.GetDomain(), a.details.ErrorInfo.GetMetadata()))
	}

	if a.details.BadRequest != nil {
		v := a.details.BadRequest.GetFieldViolations()
		var f []string
		var desc []string
		for _, x := range v {
			f = append(f, x.GetField())
			desc = append(desc, x.GetDescription())
		}
		d.WriteString(fmt.Sprintf("error details: name = BadRequest field = %s desc = %s\n",
			strings.Join(f, " "), strings.Join(desc, " ")))
	}

	if a.details.PreconditionFailure != nil {
		v := a.details.PreconditionFailure.GetViolations()
		var t []string
		var s []string
		var desc []string
		for _, x := range v {
			t = append(t, x.GetType())
			s = append(s, x.GetSubject())
			desc = append(desc, x.GetDescription())
		}
		d.WriteString(fmt.Sprintf("error details: name = PreconditionFailure type = %s subj = %s desc = %s\n", strings.Join(t, " "),
			strings.Join(s, " "), strings.Join(desc, " ")))
	}

	if a.details.QuotaFailure != nil {
		v := a.details.QuotaFailure.GetViolations()
		var s []string
		var desc []string
		for _, x := range v {
			s = append(s, x.GetSubject())
			desc = append(desc, x.GetDescription())
		}
		d.WriteString(fmt.Sprintf("error details: name = QuotaFailure subj = %s desc = %s\n",
			strings.Join(s, " "), strings.Join(desc, " ")))
	}

	if a.details.RequestInfo != nil {
		d.WriteString(fmt.Sprintf("error details: name = RequestInfo id = %s data = %s\n",
			a.details.RequestInfo.GetRequestId(), a.details.RequestInfo.GetServingData()))
	}

	if a.details.ResourceInfo != nil {
		d.WriteString(fmt.Sprintf("error details: name = ResourceInfo type = %s resourcename = %s owner = %s desc = %s\n",
			a.details.ResourceInfo.GetResourceType(), a.details.ResourceInfo.GetResourceName(),
			a.details.ResourceInfo.GetOwner(), a.details.ResourceInfo.GetDescription()))

	}
	if a.details.RetryInfo != nil {
		d.WriteString(fmt.Sprintf("error details: retry in %s\n", a.details.RetryInfo.GetRetryDelay().AsDuration()))

	}
	if a.details.Unknown != nil {
		var s []string
		for _, x := range a.details.Unknown {
			s = append(s, fmt.Sprintf("%v", x))
		}
		d.WriteString(fmt.Sprintf("error details: name = Unknown  desc = %s\n", strings.Join(s, " ")))
	}

	if a.details.DebugInfo != nil {
		d.WriteString(fmt.Sprintf("error details: name = DebugInfo detail = %s stack = %s\n", a.details.DebugInfo.GetDetail(),
			strings.Join(a.details.DebugInfo.GetStackEntries(), " ")))
	}
	if a.details.Help != nil {
		var desc []string
		var url []string
		for _, x := range a.details.Help.Links {
			desc = append(desc, x.GetDescription())
			url = append(url, x.GetUrl())
		}
		d.WriteString(fmt.Sprintf("error details: name = Help desc = %s url = %s\n",
			strings.Join(desc, " "), strings.Join(url, " ")))
	}
	if a.details.LocalizedMessage != nil {
		d.WriteString(fmt.Sprintf("error details: name = LocalizedMessge locale = %s msg = %s\n",
			a.details.LocalizedMessage.GetLocale(), a.details.LocalizedMessage.GetMessage()))
	}
	return strings.TrimSpace(d.String())
}

// GRPCStatus extracts the underlying gRPC Status error.
func (a *APIError) GRPCStatus() *status.Status {
	return a.status
}

// Reason returns the reason in an ErrorInfo.
// If ErrorInfo is nil, it returns an empty string.
func (a *APIError) Reason() string {
	return a.details.ErrorInfo.GetReason()
}

// Domain returns the domain in an ErrorInfo.
// If ErrorInfo is nil, it returns an empty string.
func (a *APIError) Domain() string {
	return a.details.ErrorInfo.GetDomain()
}

// MetaData returns the metadata in an ErrorInfo.
// If ErroInfo is nil, it returns nil.
func (a *APIError) Metadata() map[string]string {
	return a.details.ErrorInfo.GetMetadata()

}

// FromError parses a Status error and builds an APIError.
func FromError(err error) (*APIError, bool) {
	if err == nil {
		return nil, false
	}
	st, ok := status.FromError(err)
	if !ok {
		return nil, false
	}
	msg := ErrDetails{}
	for _, d := range st.Details() {
		switch d := d.(type) {
		case *errdetails.ErrorInfo:
			msg.ErrorInfo = d
		case *errdetails.BadRequest:
			msg.BadRequest = d
		case *errdetails.PreconditionFailure:
			msg.PreconditionFailure = d
		case *errdetails.QuotaFailure:
			msg.QuotaFailure = d
		case *errdetails.RetryInfo:
			msg.RetryInfo = d
		case *errdetails.ResourceInfo:
			msg.ResourceInfo = d
		case *errdetails.RequestInfo:
			msg.RequestInfo = d
		case *errdetails.DebugInfo:
			msg.DebugInfo = d
		case *errdetails.Help:
			msg.Help = d
		case *errdetails.LocalizedMessage:
			msg.LocalizedMessage = d
		default:
			msg.Unknown = append(msg.Unknown, d)
		}
	}
	return &APIError{
		details: msg,
		err:     err,
		status:  st,
	}, true

}
