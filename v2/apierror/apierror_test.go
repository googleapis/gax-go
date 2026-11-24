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

package apierror

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	jsonerror "github.com/googleapis/gax-go/v2/apierror/internal/proto"
	"google.golang.org/api/googleapi"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
)

var update = flag.Bool("update", false, "update golden files")

func TestDetails(t *testing.T) {
	qf := &errdetails.QuotaFailure{
		Violations: []*errdetails.QuotaFailure_Violation{{Subject: "Foo", Description: "Bar"}},
	}
	qS, _ := status.New(codes.ResourceExhausted, "test").WithDetails(qf)
	apierr := &APIError{
		err:     qS.Err(),
		status:  qS,
		details: ErrDetails{QuotaFailure: qf},
	}
	got := apierr.Details()
	want := ErrDetails{QuotaFailure: qf}
	if diff := cmp.Diff(got, want, cmp.Comparer(proto.Equal)); diff != "" {
		t.Errorf("got(-), want(+):\n%s", diff)
	}
}

func TestDetails_ExtractProtoMessage(t *testing.T) {

	customError := &jsonerror.CustomError{
		Code:         jsonerror.CustomError_UNIVERSE_WAS_DESTROYED,
		Entity:       "some entity",
		ErrorMessage: "custom error message",
	}

	testCases := []struct {
		description string
		src         *status.Status
		extract     proto.Message
		want        interface{}
		wantErr     error
	}{
		{
			description: "no details",
			src:         status.New(codes.Unimplemented, "unimp"),
			extract:     &jsonerror.CustomError{},
			wantErr:     ErrMessageNotFound,
		},
		{
			description: "nil argument",
			src: func() *status.Status {
				s, _ := status.New(codes.Unauthenticated, "who are you").WithDetails(
					&descriptorpb.DescriptorProto{},
				)
				return s
			}(),
			wantErr: ErrMessageNotFound,
		},
		{
			description: "custom error success",
			src: func() *status.Status {
				s, _ := status.New(codes.Unknown, "unknown error").WithDetails(
					customError,
				)
				return s
			}(),
			extract: &jsonerror.CustomError{},
			want:    customError,
		},
	}
	for _, tc := range testCases {

		apiErr, ok := FromError(tc.src.Err())
		if !ok {
			t.Errorf("%s: FromError failure", tc.description)
		}
		val := tc.extract
		gotErr := apiErr.Details().ExtractProtoMessage(val)
		if tc.wantErr != nil {
			if !errors.Is(gotErr, tc.wantErr) {
				t.Errorf("%s: got error %v, wanted error %v", tc.description, gotErr, tc.wantErr)
			}
		} else {
			if gotErr != nil {
				t.Errorf("%s: got error %v", tc.description, gotErr)
			}
			if diff := cmp.Diff(val, tc.want, protocmp.Transform()); diff != "" {
				t.Errorf("%s: got(-), want(+):\n%s", tc.description, diff)
			}
		}
	}
}
func TestUnwrap(t *testing.T) {
	pf := &errdetails.PreconditionFailure{
		Violations: []*errdetails.PreconditionFailure_Violation{{Type: "Foo", Subject: "Bar", Description: "desc"}},
	}
	pS, _ := status.New(codes.FailedPrecondition, "test").WithDetails(pf)
	apierr := &APIError{
		err:     pS.Err(),
		status:  pS,
		details: ErrDetails{PreconditionFailure: pf},
	}
	got := apierr.Unwrap()
	want := pS.Err()
	if diff := cmp.Diff(got, want, cmpopts.EquateErrors()); diff != "" {
		t.Errorf("got(-), want(+):\n%s", diff)
	}
}
func TestError(t *testing.T) {
	ei := &errdetails.ErrorInfo{
		Reason:   "Foo",
		Domain:   "Bar",
		Metadata: map[string]string{"type": "test"},
	}
	eS, _ := status.New(codes.Unauthenticated, "ei").WithDetails(ei)

	br := &errdetails.BadRequest{FieldViolations: []*errdetails.BadRequest_FieldViolation{{
		Field:       "Foo",
		Description: "Bar",
	}},
	}
	bS, _ := status.New(codes.InvalidArgument, "br").WithDetails(br)

	qf := &errdetails.QuotaFailure{
		Violations: []*errdetails.QuotaFailure_Violation{{Subject: "Foo", Description: "Bar"}},
	}
	pf := &errdetails.PreconditionFailure{
		Violations: []*errdetails.PreconditionFailure_Violation{{Type: "Foo", Subject: "Bar", Description: "desc"}},
	}

	ri := &errdetails.RetryInfo{
		RetryDelay: &durationpb.Duration{Seconds: 10, Nanos: 10},
	}
	rq := &errdetails.RequestInfo{
		RequestId:   "Foo",
		ServingData: "Bar",
	}
	rqS, _ := status.New(codes.Canceled, "Request cancelled by client").WithDetails(rq, ri, pf, br, qf)

	rs := &errdetails.ResourceInfo{
		ResourceType: "Foo",
		ResourceName: "Bar",
		Owner:        "Client",
		Description:  "Directory not Found",
	}
	rS, _ := status.New(codes.NotFound, "rs").WithDetails(rs)

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

	var uu []interface{}
	uu = append(uu, "unknown detail 1")
	uS := status.New(codes.Unknown, "Unknown")

	httpErrInfo := &errdetails.ErrorInfo{Reason: "just because", Domain: "tests"}
	hae := &googleapi.Error{
		Message: "just because",
		Body:    marshalHTTPErrorBody(t, httpErrInfo),
	}
	haeS := status.New(codes.Unknown, "just because")

	tests := []struct {
		apierr *APIError
		name   string
	}{
		{&APIError{err: eS.Err(), status: eS, details: ErrDetails{ErrorInfo: ei}}, "error_info"},
		{&APIError{err: bS.Err(), status: bS, details: ErrDetails{BadRequest: br}}, "bad_request"},
		{&APIError{err: rqS.Err(), status: rqS, details: ErrDetails{RequestInfo: rq, RetryInfo: ri,
			PreconditionFailure: pf, QuotaFailure: qf, BadRequest: br}}, "multiple_info"},
		{&APIError{err: bS.Err(), status: rS, details: ErrDetails{ResourceInfo: rs}}, "resource_info"},
		{&APIError{err: bS.Err(), status: dS, details: ErrDetails{DebugInfo: deb}}, "debug_info"},
		{&APIError{err: bS.Err(), status: hS, details: ErrDetails{Help: hp}}, "help"},
		{&APIError{err: bS.Err(), status: lS, details: ErrDetails{LocalizedMessage: lo}}, "localized_message"},
		{&APIError{err: bS.Err(), status: uS, details: ErrDetails{Unknown: uu}}, "unknown"},
		{&APIError{err: bS.Err(), status: bS, details: ErrDetails{}}, "empty"},
		{&APIError{err: hae, httpErr: hae, status: haeS, details: ErrDetails{ErrorInfo: httpErrInfo}}, "http_err"},
	}
	for _, tc := range tests {
		t.Helper()
		got := tc.apierr.Error()
		want, err := golden(tc.name, got)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("got(-), want(+),: \n%s", diff)
		}
	}

}

func TestGRPCStatus(t *testing.T) {
	qf := &errdetails.QuotaFailure{
		Violations: []*errdetails.QuotaFailure_Violation{{Subject: "Foo", Description: "Bar"}},
	}
	want, _ := status.New(codes.ResourceExhausted, "test").WithDetails(qf)
	apierr := &APIError{
		err:     want.Err(),
		status:  want,
		details: ErrDetails{QuotaFailure: qf},
	}
	got := apierr.GRPCStatus()
	if diff := cmp.Diff(got, want, cmp.Comparer(proto.Equal), cmp.AllowUnexported(status.Status{})); diff != "" {
		t.Errorf("got(-), want(+),: \n%s", diff)
	}
}

func TestReason(t *testing.T) {
	tests := []struct {
		ei *errdetails.ErrorInfo
	}{
		{&errdetails.ErrorInfo{Reason: "Foo"}},
		{&errdetails.ErrorInfo{}},
	}
	for _, tc := range tests {
		apierr := toAPIError(tc.ei)
		if diff := cmp.Diff(apierr.Reason(), tc.ei.GetReason()); diff != "" {
			t.Errorf("got(-), want(+),: \n%s", diff)
		}
	}
}
func TestDomain(t *testing.T) {
	tests := []struct {
		ei *errdetails.ErrorInfo
	}{
		{&errdetails.ErrorInfo{Domain: "Bar"}},
		{&errdetails.ErrorInfo{}},
	}
	for _, tc := range tests {
		apierr := toAPIError(tc.ei)
		if diff := cmp.Diff(apierr.Domain(), tc.ei.GetDomain()); diff != "" {
			t.Errorf("got(-), want(+),: \n%s", diff)
		}
	}
}
func TestMetadata(t *testing.T) {
	tests := []struct {
		ei *errdetails.ErrorInfo
	}{
		{&errdetails.ErrorInfo{Metadata: map[string]string{"type": "test"}}},
		{&errdetails.ErrorInfo{}},
	}
	for _, tc := range tests {
		apierr := toAPIError(tc.ei)
		if diff := cmp.Diff(apierr.Metadata(), tc.ei.GetMetadata()); diff != "" {
			t.Errorf("got(-), want(+),: \n%s", diff)
		}
	}
}

func TestFromError(t *testing.T) {
	ei := &errdetails.ErrorInfo{
		Reason:   "Foo",
		Domain:   "Bar",
		Metadata: map[string]string{"type": "test"},
	}
	eS, _ := status.New(codes.Unauthenticated, "ei").WithDetails(ei)

	br := &errdetails.BadRequest{FieldViolations: []*errdetails.BadRequest_FieldViolation{{
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

	msg := &descriptorpb.DescriptorProto{
		Name: proto.String("Foo"),
	}
	u := []interface{}{msg}
	uS, _ := status.New(codes.Unknown, "test").WithDetails(msg)

	httpErrInfo := &errdetails.ErrorInfo{Reason: "just because", Domain: "tests"}
	hae := &googleapi.Error{
		Message: "just because",
		Body:    marshalHTTPErrorBody(t, httpErrInfo),
	}
	haeS := status.New(codes.Unknown, "just because")

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
		{&APIError{err: uS.Err(), status: uS, details: ErrDetails{Unknown: u}}, true},
		{&APIError{err: hae, httpErr: hae, status: haeS, details: ErrDetails{ErrorInfo: httpErrInfo}}, true},
		{&APIError{err: errors.New("standard error")}, false},
	}

	for _, tc := range tests {
		got, apiB := FromError(tc.apierr.err)
		if tc.b != apiB {
			t.Errorf("FromError(%s): got %v, want %v", tc.apierr.err, apiB, tc.b)
		}
		if tc.b {
			if diff := cmp.Diff(got.details, tc.apierr.details, cmp.Comparer(proto.Equal)); diff != "" {
				t.Errorf("FromError(%s): got(-), want(+),: \n%s", tc.apierr.err, diff)
			}
			if diff := cmp.Diff(got.status, tc.apierr.status, cmp.Comparer(proto.Equal), cmp.AllowUnexported(status.Status{})); diff != "" {
				t.Errorf("FromError(%s): got(-), want(+),: \n%s", tc.apierr.err, diff)
			}
			if diff := cmp.Diff(got.err, tc.apierr.err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("FromError(%s): got(-), want(+),: \n%s", tc.apierr.err, diff)
			}
		}
	}
	if err, _ := FromError(nil); err != nil {
		t.Errorf("got %s, want nil", err)
	}

	if c, _ := FromError(context.DeadlineExceeded); c != nil {
		t.Errorf("got %s, want nil", c)
	}
}

func TestParseError(t *testing.T) {
	httpErrInfo := &errdetails.ErrorInfo{Reason: "just because", Domain: "tests"}
	hae := &googleapi.Error{
		Message: "just because",
		Body:    marshalHTTPErrorBody(t, httpErrInfo),
	}
	haeS := status.New(codes.Unknown, "just because")

	se := errors.New("standard error")

	tests := []struct {
		name       string
		source     error
		wantErr    *APIError
		wantParsed bool
	}{
		{
			name:       "with http ErrorInfo details",
			source:     hae,
			wantErr:    &APIError{httpErr: hae, status: haeS, details: ErrDetails{ErrorInfo: httpErrInfo}},
			wantParsed: true,
		},
		{
			name:       "standard error no details",
			source:     se,
			wantErr:    &APIError{err: se},
			wantParsed: false,
		},
		{
			name:       "grpc status no details",
			source:     haeS.Err(),
			wantErr:    &APIError{httpErr: hae, status: haeS, details: ErrDetails{}},
			wantParsed: true,
		},
	}

	for _, tc := range tests {
		// ParseError with wrap = true is covered by TestFromError, above.
		got, gotParsed := ParseError(tc.source, false)
		if tc.wantParsed != gotParsed {
			t.Errorf("ParseError(%s, false): got %v, want %v", tc.wantErr, gotParsed, tc.wantParsed)
		}
		if got != nil {
			if got.Error() == "" {
				t.Errorf("got.Error(): got %q, want non-empty", got.Error())
			}
		}
		if tc.wantParsed {
			if diff := cmp.Diff(got.details, tc.wantErr.details, cmp.Comparer(proto.Equal)); diff != "" {
				t.Errorf("got(-), want(+),: \n%s", diff)
			}
			if diff := cmp.Diff(got.status, tc.wantErr.status, cmp.Comparer(proto.Equal), cmp.AllowUnexported(status.Status{})); diff != "" {
				t.Errorf("got(-), want(+),: \n%s", diff)
			}
			if got.err != nil {
				t.Errorf("got %s, want nil", got.err)
			}
		} else {
			if got != nil {
				t.Errorf("got %s, want nil", got)
			}
		}
	}
	if err, _ := ParseError(nil, false); err != nil {
		t.Errorf("got %s, want nil", err)
	}

	if c, _ := ParseError(context.DeadlineExceeded, false); c != nil {
		t.Errorf("got %s, want nil", c)
	}
}

func golden(name, got string) (string, error) {
	g := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.WriteFile(g, []byte(got), 0644); err != nil {
			return "", err
		}
	}
	want, err := os.ReadFile(g)
	return string(want), err
}

func toAPIError(e *errdetails.ErrorInfo) *APIError {
	st, _ := status.New(codes.Unavailable, "test").WithDetails(e)
	return &APIError{
		err:     st.Err(),
		status:  st,
		details: ErrDetails{ErrorInfo: e},
	}
}

func TestHTTPCode(t *testing.T) {
	tests := []struct {
		name   string
		apierr *APIError
		want   int
	}{
		{
			name:   "basic http error",
			apierr: &APIError{httpErr: &googleapi.Error{Code: 418}},
			want:   418,
		},
		{
			name:   "http error, with unknown status",
			apierr: &APIError{httpErr: &googleapi.Error{Code: 418}, status: status.New(codes.Unknown, "???")},
			want:   418,
		},
		{
			name:   "gRPC error",
			apierr: &APIError{status: status.New(codes.DataLoss, "where did it go?")},
			want:   -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.apierr.HTTPCode(); got != tt.want {
				t.Errorf("HTTPCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHTTPErrorAsStatus(t *testing.T) {
	testCases := []struct {
		name     string
		err      *googleapi.Error
		wantCode codes.Code
	}{
		{
			name:     "no code",
			err:      &googleapi.Error{},
			wantCode: codes.Unknown,
		},
		{
			name: "not found",
			err: &googleapi.Error{
				Code:    http.StatusNotFound,
				Message: "message",
			},
			wantCode: codes.NotFound,
		},
		{
			name: "unauthorized",
			err: &googleapi.Error{
				Code:    http.StatusUnauthorized,
				Message: "message",
			},
			wantCode: codes.Unauthenticated,
		},
		{
			name: "2xx range",
			err: &googleapi.Error{
				Code: http.StatusPartialContent,
			},
			wantCode: codes.OK,
		},
		{
			name: "4xx range",
			err: &googleapi.Error{
				Code: http.StatusLengthRequired,
			},
			wantCode: codes.FailedPrecondition,
		},
		{
			name: "5xx range",
			err: &googleapi.Error{
				Code: http.StatusLoopDetected,
			},
			wantCode: codes.Internal,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			apiErr, ok := FromError(tc.err)
			if !ok {
				t.Error("FromError failed to transform error")
			}
			st := apiErr.GRPCStatus()
			if st.Code() != tc.wantCode {
				t.Errorf("failure, got %s want %s", st.Code().String(), tc.wantCode.String())
			}
		})

	}
}

func marshalHTTPErrorBody(t *testing.T, msgs ...proto.Message) string {
	t.Helper()
	rawDetails := make([]json.RawMessage, 0, len(msgs))
	for _, m := range msgs {
		a, err := anypb.New(m)
		if err != nil {
			t.Fatal(err)
		}
		b, err := protojson.Marshal(a)
		if err != nil {
			t.Fatal(err)
		}
		rawDetails = append(rawDetails, b)
	}
	payload := struct {
		Error struct {
			Details []json.RawMessage `json:"details"`
		} `json:"error"`
	}{}
	payload.Error.Details = rawDetails
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestParseHTTPDetails(t *testing.T) {
	t.Run("legacy errors array and unregistered @type preserve standard ErrorInfo", func(t *testing.T) {
		unregisteredDetail := `{"@type":"type.googleapis.com/edge.configstore.bundle.BadBundle","violations":[{"description":"The proxy must contain a ProxyEndpoint"}]}`
		rawBody := `{
  "error": {
    "code": 400,
    "message": "bundle contains errors",
    "status": "INVALID_ARGUMENT",
    "errors": [{"message": "bundle contains errors", "domain": "global", "reason": "badRequest"}],
    "details": [
      {
        "@type": "type.googleapis.com/google.rpc.ErrorInfo",
        "reason": "BUNDLE_INVALID",
        "domain": "apigee.googleapis.com"
      },
      ` + unregisteredDetail + `
    ]
  }
}`
		httpErr := &googleapi.Error{
			Code:    http.StatusBadRequest,
			Message: "bundle contains errors",
			Body:    rawBody,
		}

		apiErr, ok := FromError(httpErr)
		if !ok {
			t.Fatal("FromError returned false, want true")
		}
		if got, want := apiErr.Reason(), "BUNDLE_INVALID"; got != want {
			t.Errorf("Reason() = %q, want %q", got, want)
		}
		if got, want := apiErr.Domain(), "apigee.googleapis.com"; got != want {
			t.Errorf("Domain() = %q, want %q", got, want)
		}
		wantInfo := &errdetails.ErrorInfo{
			Reason: "BUNDLE_INVALID",
			Domain: "apigee.googleapis.com",
		}
		if diff := cmp.Diff(apiErr.Details().ErrorInfo, wantInfo, protocmp.Transform()); diff != "" {
			t.Errorf("ErrorInfo mismatch (-got +want):\n%s", diff)
		}
		if len(apiErr.Details().Unknown) != 1 {
			t.Fatalf("len(Unknown) = %d, want 1", len(apiErr.Details().Unknown))
		}
		rawUnknown, ok := apiErr.Details().Unknown[0].(json.RawMessage)
		if !ok {
			t.Fatalf("Unknown[0] type = %T, want json.RawMessage", apiErr.Details().Unknown[0])
		}
		if diff := cmp.Diff(string(rawUnknown), unregisteredDetail); diff != "" {
			t.Errorf("Unknown[0] mismatch (-got +want):\n%s", diff)
		}
		if !strings.Contains(apiErr.Error(), "edge.configstore.bundle.BadBundle") {
			t.Errorf("Error() = %q, want it to contain unregistered @type JSON", apiErr.Error())
		}
	})

	t.Run("array envelope with leading whitespace", func(t *testing.T) {
		rawBody := `  [{"error":{"code":400,"message":"bad request","details":[{"@type":"type.googleapis.com/google.rpc.ErrorInfo","reason":"INVALID_FIELD","domain":"example.googleapis.com"}]}}]`
		httpErr := &googleapi.Error{
			Code:    http.StatusBadRequest,
			Message: "bad request",
			Body:    rawBody,
		}
		apiErr, ok := FromError(httpErr)
		if !ok {
			t.Fatal("FromError returned false, want true")
		}
		if got, want := apiErr.Reason(), "INVALID_FIELD"; got != want {
			t.Errorf("Reason() = %q, want %q", got, want)
		}
		if got, want := apiErr.Domain(), "example.googleapis.com"; got != want {
			t.Errorf("Domain() = %q, want %q", got, want)
		}
	})

	t.Run("unknown field inside known proto detail is discarded", func(t *testing.T) {
		rawBody := `{"error":{"code":400,"message":"bad request","details":[{"@type":"type.googleapis.com/google.rpc.ErrorInfo","reason":"SOME_REASON","domain":"example.googleapis.com","extraField":"ignored"}]}}`
		httpErr := &googleapi.Error{
			Code:    http.StatusBadRequest,
			Message: "bad request",
			Body:    rawBody,
		}
		apiErr, ok := FromError(httpErr)
		if !ok {
			t.Fatal("FromError returned false, want true")
		}
		if got, want := apiErr.Reason(), "SOME_REASON"; got != want {
			t.Errorf("Reason() = %q, want %q", got, want)
		}
	})

	t.Run("registered custom proto extracted alongside unregistered @type", func(t *testing.T) {
		wantCustom := &jsonerror.CustomError{
			Code:         jsonerror.CustomError_UNIVERSE_WAS_DESTROYED,
			Entity:       "world",
			ErrorMessage: "boom",
		}
		customAny, err := anypb.New(wantCustom)
		if err != nil {
			t.Fatal(err)
		}
		customJSON, err := protojson.Marshal(customAny)
		if err != nil {
			t.Fatal(err)
		}
		rawBody := `{"error":{"code":500,"message":"boom","details":[` + string(customJSON) + `,{"@type":"type.googleapis.com/unregistered.Detail","foo":"bar"}]}}`
		httpErr := &googleapi.Error{
			Code:    http.StatusInternalServerError,
			Message: "boom",
			Body:    rawBody,
		}
		apiErr, ok := FromError(httpErr)
		if !ok {
			t.Fatal("FromError returned false, want true")
		}
		gotCustom := &jsonerror.CustomError{}
		if err := apiErr.Details().ExtractProtoMessage(gotCustom); err != nil {
			t.Fatalf("ExtractProtoMessage failed: %v", err)
		}
		if diff := cmp.Diff(gotCustom, wantCustom, protocmp.Transform()); diff != "" {
			t.Errorf("CustomError mismatch (-got +want):\n%s", diff)
		}
		if len(apiErr.Details().Unknown) != 2 {
			t.Errorf("len(Unknown) = %d, want 2", len(apiErr.Details().Unknown))
		}
	})

	t.Run("malformed or empty array envelopes do not panic", func(t *testing.T) {
		for _, body := range []string{"", "[]", "[{}]", "not json"} {
			httpErr := &googleapi.Error{
				Code: http.StatusInternalServerError,
				Body: body,
			}
			apiErr, ok := FromError(httpErr)
			if !ok {
				t.Fatalf("FromError(%q) returned false, want true", body)
			}
			if diff := cmp.Diff(apiErr.Details(), ErrDetails{}); diff != "" {
				t.Errorf("FromError(%q).Details() mismatch (-got +want):\n%s", body, diff)
			}
		}
	})
}

func TestHTTPErrorNonJSONBodyFallback(t *testing.T) {
	htmlBody := "<html><title>502 Bad Gateway</title></html>"
	httpErr := &googleapi.Error{
		Code: http.StatusBadGateway,
		Body: htmlBody,
	}

	apiErr, ok := FromError(httpErr)
	if !ok {
		t.Fatal("FromError returned false, want true")
	}
	want := "googleapi: Error 502: " + htmlBody
	if got := apiErr.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	if diff := cmp.Diff(apiErr.Details(), ErrDetails{}); diff != "" {
		t.Errorf("Details() mismatch (-got +want):\n%s", diff)
	}
}
