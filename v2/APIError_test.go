package gax

import (
	"context"
	"encoding/json"
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
	stat, _ := status.New(codes.ResourceExhausted, "Per user quota has been exhausted").WithDetails(qf)
	apierr := &APIError{
		err:     stat.Err(),
		status:  stat,
		details: ErrDetails{QuotaFailure: qf},
	}
	if diff := cmp.Diff(apierr.details, apierr.Details(), cmp.Comparer(proto.Equal)); diff != "" {
		t.Errorf("Expected(+) but got(-):\n%s", diff)
	}
}

func TestError(t *testing.T) {
	pf := &errdetails.PreconditionFailure{
		Violations: []*errdetails.PreconditionFailure_Violation{{Type: "Foo", Subject: "Bar"}},
	}
	stat, _ := status.New(codes.FailedPrecondition, "System's state is not suitable for operation execution").WithDetails(pf)
	apierr := &APIError{
		err:     stat.Err(),
		status:  stat,
		details: ErrDetails{PreconditionFailure: pf},
	}
	strRep, _ := json.Marshal(apierr.details)
	expected := apierr.err.Error() + "\n" + "Here are the details: " + "\n" + string(strRep)
	if expected != apierr.Error() {
		t.Errorf("Expected: %s but got: %s", expected, apierr.Error())
	}
}

func TestGRPCStatus(t *testing.T) {
	qf := &errdetails.QuotaFailure{
		Violations: []*errdetails.QuotaFailure_Violation{{Subject: "Foo", Description: "Bar"}},
	}
	stat, _ := status.New(codes.ResourceExhausted, "Per user quota has been exhausted").WithDetails(qf)
	apierr := &APIError{
		err:     stat.Err(),
		status:  stat,
		details: ErrDetails{QuotaFailure: qf},
	}
	if apierr.status != apierr.GRPCStatus() {
		t.Errorf("Expected: %v but got: %v", apierr.status, apierr.GRPCStatus())
	}
}

func TestFromError(t *testing.T) {
	type test struct {
		name string
		got  *APIError
		want *APIError
	}
	err, _ := FromError(nil)
	if err != nil {
		t.Errorf("Expected nil but got: %s", err)
	}

	ctxErr, _ := FromError(context.DeadlineExceeded)
	if ctxErr != nil {
		t.Errorf("Expected %s: but got %s:", context.DeadlineExceeded, ctxErr)
	}

	br := &errdetails.BadRequest{
		FieldViolations: []*errdetails.BadRequest_FieldViolation{{
			Field:       "Foo",
			Description: "Bar",
		}},
	}
	brStat, _ := status.New(codes.InvalidArgument, "bad request").WithDetails(br)
	brExpected := &APIError{
		err:    brStat.Err(),
		status: brStat,
		details: ErrDetails{
			BadRequest: br},
	}
	brActual, _ := FromError(brStat.Err())

	qf := &errdetails.QuotaFailure{
		Violations: []*errdetails.QuotaFailure_Violation{{Subject: "Foo", Description: "Bar"}},
	}
	qfStat, _ := status.New(codes.ResourceExhausted, "Per user quota has been exhausted").WithDetails(qf)
	qfExpected := &APIError{
		err:     qfStat.Err(),
		status:  qfStat,
		details: ErrDetails{QuotaFailure: qf},
	}
	qfActual, _ := FromError(qfStat.Err())

	pf := &errdetails.PreconditionFailure{
		Violations: []*errdetails.PreconditionFailure_Violation{{Type: "Foo", Subject: "Bar"}},
	}
	pfStat, _ := status.New(codes.FailedPrecondition, "System's state is not suitable for operation execution").WithDetails(pf)
	pfExpected := &APIError{
		err:     pfStat.Err(),
		status:  pfStat,
		details: ErrDetails{PreconditionFailure: pf},
	}
	pfActual, _ := FromError(pfStat.Err())

	ri := &errdetails.RetryInfo{
		RetryDelay: &durationpb.Duration{Seconds: 10},
	}
	riStat, _ := status.New(codes.Unavailable, "foo").WithDetails(ri)
	riExpected := &APIError{
		err:     riStat.Err(),
		status:  riStat,
		details: ErrDetails{RetryInfo: ri},
	}
	riActual, _ := FromError(riStat.Err())

	res := &errdetails.ResourceInfo{
		ResourceType: "Foo",
		ResourceName: "Bar",
		Owner:        "Client",
		Description:  "Directory not Found",
	}
	resStat, _ := status.New(codes.NotFound, "Missing directory").WithDetails(res)
	resExpected := &APIError{
		err:     resStat.Err(),
		status:  resStat,
		details: ErrDetails{ResourceInfo: res},
	}
	resActual, _ := FromError(resStat.Err())

	req := &errdetails.RequestInfo{
		RequestId:   "Foo",
		ServingData: "Bar",
	}
	reqStat, _ := status.New(codes.Canceled, "Request cancelled by client").WithDetails(req)
	reqExpected := &APIError{
		err:     reqStat.Err(),
		status:  reqStat,
		details: ErrDetails{RequestInfo: req},
	}
	reqActual, _ := FromError(reqStat.Err())

	deb := &errdetails.DebugInfo{
		StackEntries: []string{"Foo", "Bar"},
		Detail:       "Stack Details",
	}
	debStat, _ := status.New(codes.DataLoss, "Here is the debug info").WithDetails(deb)
	debExpected := &APIError{
		err:     debStat.Err(),
		status:  debStat,
		details: ErrDetails{DebugInfo: deb},
	}
	debActual, _ := FromError(debStat.Err())

	hp := &errdetails.Help{
		Links: []*errdetails.Help_Link{{Description: "Foo", Url: "Bar"}},
	}
	hpStat, _ := status.New(codes.Unimplemented, "Help Info").WithDetails(hp)
	hpExpected := &APIError{
		err:     hpStat.Err(),
		status:  hpStat,
		details: ErrDetails{Help: hp},
	}
	hpActual, _ := FromError(hpStat.Err())
	lo := &errdetails.LocalizedMessage{
		Locale:  "Foo",
		Message: "Bar",
	}
	loStat, _ := status.New(codes.Unknown, "Localized Message").WithDetails(lo)
	loExpected := &APIError{
		err:     loStat.Err(),
		status:  loStat,
		details: ErrDetails{LocalizedMesage: lo},
	}
	loActual, _ := FromError(loStat.Err())

	tests := []test{
		{name: "BadRequest", want: brExpected, got: brActual},
		{name: "QuotaFailure", want: qfExpected, got: qfActual},
		{name: "PreconditionFailure", want: pfExpected, got: pfActual},
		{name: "RetryInfo", want: riExpected, got: riActual},
		{name: "ResourceInfo", want: resExpected, got: resActual},
		{name: "RequestInfo", want: reqExpected, got: reqActual},
		{name: "DebugInfo", want: debExpected, got: debActual},
		{name: "Help", want: hpExpected, got: hpActual},
		{name: "LocalizedMessage", want: loExpected, got: loActual},
	}
	for _, tc := range tests {
		if diff := cmp.Diff(tc.got.details, tc.want.details, cmp.Comparer(proto.Equal)); diff != "" {
			t.Errorf("Actual(-), Expected(+): \n%s", diff)
		}
		if diff := cmp.Diff(tc.got.status, tc.want.status, cmp.Comparer(proto.Equal), cmp.AllowUnexported(status.Status{})); diff != "" {
			t.Errorf("Actual(-), Expected(+): \n%s", diff)
		}
		if diff := cmp.Diff(tc.got.err, tc.want.err, cmpopts.EquateErrors()); diff != "" {
			t.Errorf("Actual(-), Expected(+): \n%s", diff)
		}
	}
}
