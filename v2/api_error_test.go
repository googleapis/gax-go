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
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestDetails(t *testing.T) {
	qf := &errdetails.QuotaFailure{
		Violations: []*errdetails.QuotaFailure_Violation{{Subject: "Foo", Description: "Bar"}},
	}
	st, _ := status.New(codes.ResourceExhausted, "qf").WithDetails(qf)
	apierr := &APIError{
		err:     st.Err(),
		status:  st,
		details: ErrDetails{QuotaFailure: qf},
	}
	if diff := cmp.Diff(ErrDetails{QuotaFailure: qf}, apierr.Details(), cmp.Comparer(proto.Equal)); diff != "" {
		t.Errorf("Expected(+), Actual(-):\n%s", diff)
	}
}

func TestError(t *testing.T) {
	pf := &errdetails.PreconditionFailure{
		Violations: []*errdetails.PreconditionFailure_Violation{{Type: "Foo", Subject: "Bar", Description: "test"}},
	}
	st, _ := status.New(codes.FailedPrecondition, "pf").WithDetails(pf)
	apierr := &APIError{
		err:     st.Err(),
		status:  st,
		details: ErrDetails{PreconditionFailure: pf},
	}
	if !strings.Contains(apierr.Error(), "Foo") {
		t.Errorf("Status message not included!")
	}
}

func TestGRPCStatus(t *testing.T) {
	qf := &errdetails.QuotaFailure{
		Violations: []*errdetails.QuotaFailure_Violation{{Subject: "Foo", Description: "Bar"}},
	}
	st, _ := status.New(codes.ResourceExhausted, "Per user quota has been exhausted").WithDetails(qf)
	apierr := &APIError{
		err:     st.Err(),
		status:  st,
		details: ErrDetails{QuotaFailure: qf},
	}
	if st != apierr.GRPCStatus() {
		t.Errorf("Expected: %v but got: %v", st, apierr.GRPCStatus())
	}
}

func TestFromError(t *testing.T) {
	m := make(map[string]string)
	m["type"] = "ErrorInfo"
	ei := &errdetails.ErrorInfo{
		Reason:   "Foo",
		Domain:   "Bar",
		Metadata: m,
	}
	eS, _ := status.New(codes.Unauthenticated, "ei").WithDetails(ei)

	br := &errdetails.BadRequest{
		FieldViolations: []*errdetails.BadRequest_FieldViolation{{
			Field:       "Foo",
			Description: "Bar",
		}},
	}
	bS, _ := status.New(codes.InvalidArgument, "br").WithDetails(br)

	qf := &errdetails.QuotaFailure{
		Violations: []*errdetails.QuotaFailure_Violation{{Subject: "Foo", Description: "Bar"}},
	}
	qS, _ := status.New(codes.ResourceExhausted, "qf").WithDetails(qf, br)

	pf := &errdetails.PreconditionFailure{
		Violations: []*errdetails.PreconditionFailure_Violation{{Type: "Foo", Subject: "Bar", Description: "desc"}},
	}
	pS, _ := status.New(codes.FailedPrecondition, "pf").WithDetails(pf)

	ri := &errdetails.RetryInfo{
		RetryDelay: &durationpb.Duration{Seconds: 10, Nanos: 10},
	}
	riS, _ := status.New(codes.Unavailable, "foo").WithDetails(ri)

	rs := &errdetails.ResourceInfo{
		ResourceType: "Foo",
		ResourceName: "Bar",
		Owner:        "Client",
		Description:  "Directory not Found",
	}
	rS, _ := status.New(codes.NotFound, "rs").WithDetails(rs)

	rq := &errdetails.RequestInfo{
		RequestId:   "Foo",
		ServingData: "Bar",
	}
	rqS, _ := status.New(codes.Canceled, "Request cancelled by client").WithDetails(rq)

	deb := &errdetails.DebugInfo{
		StackEntries: []string{"Foo", "Bar"},
		Detail:       "Stack",
	}
	dS, _ := status.New(codes.DataLoss, "Here is the debug info").WithDetails(deb)

	hp := &errdetails.Help{
		Links: []*errdetails.Help_Link{{Description: "Foo", Url: "Bar"}},
	}
	hS, _ := status.New(codes.Unimplemented, "Help Info").WithDetails(hp)

	lo := &errdetails.LocalizedMessage{
		Locale:  "Foo",
		Message: "Bar",
	}
	lS, _ := status.New(codes.Unknown, "Localized Message").WithDetails(lo)

	tests := []struct {
		apierr *APIError
		b      bool
	}{
		{&APIError{err: eS.Err(), status: eS, details: ErrDetails{ErrorInfo: ei}}, true},
		{&APIError{err: bS.Err(), status: bS, details: ErrDetails{BadRequest: br}}, true},
		{&APIError{err: qS.Err(), status: qS, details: ErrDetails{QuotaFailure: qf, BadRequest: br}}, true},
		{&APIError{err: pS.Err(), status: pS, details: ErrDetails{PreconditionFailure: pf}}, true},
		{&APIError{err: riS.Err(), status: riS, details: ErrDetails{RetryInfo: ri}}, true},
		{&APIError{err: rS.Err(), status: rS, details: ErrDetails{ResourceInfo: rs}}, true},
		{&APIError{err: rqS.Err(), status: rqS, details: ErrDetails{RequestInfo: rq}}, true},
		{&APIError{err: dS.Err(), status: dS, details: ErrDetails{DebugInfo: deb}}, true},
		{&APIError{err: hS.Err(), status: hS, details: ErrDetails{Help: hp}}, true},
		{&APIError{err: lS.Err(), status: lS, details: ErrDetails{LocalizedMessage: lo}}, true},
	}

	for _, tc := range tests {
		actual, apiB := FromError(tc.apierr.err)

		if tc.b != apiB {
			t.Errorf("Expected: %v but got: %v", tc.b, apiB)
		}
		if diff := cmp.Diff(tc.apierr.details, actual.details, cmp.Comparer(proto.Equal)); diff != "" {
			t.Errorf("Actual(-), Expected(+): \n%s", diff)
		}
		if diff := cmp.Diff(tc.apierr.status, actual.status, cmp.Comparer(proto.Equal), cmp.AllowUnexported(status.Status{})); diff != "" {
			t.Errorf("Actual(-), Expected(+): \n%s", diff)
		}
		if diff := cmp.Diff(tc.apierr.err, actual.err, cmpopts.EquateErrors()); diff != "" {
			t.Errorf("Actual(-), Expected(+): \n%s", diff)
		}
	}

	if err, _ := FromError(nil); err != nil {
		t.Errorf("Expected nil but got: %v", err)
	}

	if c, _ := FromError(context.DeadlineExceeded); c != nil {
		t.Errorf("Expected nil but got: %v", c)
	}
}