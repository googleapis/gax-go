package gax

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/protobuf/ptypes/duration"
	"github.com/stretchr/testify/assert"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestEmptyDetails(t *testing.T) {
	errdetails := ErrDetails{}
	apierr := APIError{
		err:     nil,
		status:  nil,
		details: errdetails,
	}
	var actualDetails = apierr.Details()
	assert.Equal(t, apierr.details, actualDetails)
}

func TestNonEmptyDetails(t *testing.T) {
	qf := &errdetails.QuotaFailure{}
	v := &errdetails.QuotaFailure_Violation{
		Subject:     "Quotafailure",
		Description: "You have exhuasted the quota",
	}
	qf.Violations = append(qf.Violations, v)
	msg := ErrDetails{}
	st, _ := status.New(codes.ResourceExhausted, "Per user quota has been exhausted").WithDetails(qf)
	err := st.Err()
	msg.QuotaFailure = qf
	d := APIError{
		err:     err,
		status:  st,
		details: msg,
	}
	assert.Equal(t, d.details, d.Details())
}

func TestUnwrap(t *testing.T) {
	br := &errdetails.BadRequest{}
	desc := "The argument is invalid"
	v := &errdetails.BadRequest_FieldViolation{
		Field:       "username",
		Description: desc,
	}
	br.FieldViolations = append(br.FieldViolations, v)
	st, _ := status.New(codes.InvalidArgument, "Invalid argument").WithDetails(br)
	err := st.Err()
	msg := ErrDetails{}
	msg.BadRequest = br
	d := APIError{
		err:     err,
		status:  st,
		details: msg,
	}
	assert.Equal(t, d.err, d.Unwrap())
}
func TestError(t *testing.T) {
	pf := &errdetails.PreconditionFailure{}
	viol := &errdetails.PreconditionFailure_Violation{
		Type:    "Precondition Failure",
		Subject: "This is a test",
	}
	pf.Violations = append(pf.Violations, viol)
	mesg := ErrDetails{}
	mesg.PreconditionFailure = pf
	stat, _ := status.New(codes.FailedPrecondition, "System's state is not suitable for operation execution").WithDetails(pf)
	er := stat.Err()
	d := APIError{
		err:     er,
		status:  stat,
		details: mesg,
	}
	//str := err.Error() + "\n" + br.String()
	assert.Contains(t, d.Error(), "System's state is not suitable for operation execution")
}

func TestGRPCStatus(t *testing.T) {
	qf := &errdetails.QuotaFailure{}
	v := &errdetails.QuotaFailure_Violation{
		Subject:     "Quotafailure",
		Description: "You have exhuasted the quota",
	}
	qf.Violations = append(qf.Violations, v)
	st, _ := status.New(codes.ResourceExhausted, "Per user quota has been exhausted").WithDetails(qf)
	err := st.Err()
	msg := ErrDetails{}
	msg.QuotaFailure = qf
	d := APIError{
		err:     err,
		status:  st,
		details: msg,
	}
	assert.Equal(t, d.status, d.GRPCStatus())

}
func TestNilFromError(t *testing.T) {
	err := error(nil)
	_, ok := FromError(err)
	if ok == true {
		fmt.Println("Test Failed. Nil error should not be wrapped as API")
	}
}

func TestNonAPIFromError(t *testing.T) {
	err := context.DeadlineExceeded
	_, ok := FromError(err)
	if ok == true {
		fmt.Println("Test Failed. Do not wrap nonAPI errors")
	}
}

func TestAPIFromError(t *testing.T) {
	type test struct {
		name string
		want string
		got  string
	}
	br := &errdetails.BadRequest{}
	br_violation := &errdetails.BadRequest_FieldViolation{
		Field:       "field",
		Description: "desc",
	}
	br.FieldViolations = append(br.FieldViolations, br_violation)
	br_stat, _ := status.New(codes.InvalidArgument, "bad request").WithDetails(br)
	br_err := br_stat.Err()
	br_msg := ErrDetails{}
	br_msg.BadRequest = br
	br_expected := APIError{
		err:     br_err,
		status:  br_stat,
		details: br_msg,
	}
	br_actual, _ := FromError(br_err)

	qf := &errdetails.QuotaFailure{}
	qf_violation := &errdetails.QuotaFailure_Violation{
		Subject:     "Quotafailure",
		Description: "You have exhuasted the quota",
	}
	qf.Violations = append(qf.Violations, qf_violation)
	qf_msg := ErrDetails{}
	qf_stat, _ := status.New(codes.ResourceExhausted, "Per user quota has been exhausted").WithDetails(qf)
	qf_err := qf_stat.Err()
	qf_msg.QuotaFailure = qf
	qf_expected := APIError{
		err:     qf_err,
		status:  qf_stat,
		details: qf_msg,
	}
	qf_actual, _ := FromError(qf_err)

	pf := &errdetails.PreconditionFailure{}
	pf_violation := &errdetails.PreconditionFailure_Violation{
		Type:    "Precondition Failure",
		Subject: "This is a test",
	}
	pf.Violations = append(pf.Violations, pf_violation)
	pf_msg := ErrDetails{}
	pf_msg.PreconditionFailure = pf
	pf_stat, _ := status.New(codes.FailedPrecondition, "System's state is not suitable for operation execution").WithDetails(pf)
	pf_err := pf_stat.Err()
	pf_expected := APIError{
		err:     pf_err,
		status:  pf_stat,
		details: pf_msg,
	}
	pf_actual, _ := FromError(pf_err)

	ri := &errdetails.RetryInfo{}
	dur := &duration.Duration{}
	dur.Seconds = 10
	ri.RetryDelay = dur
	ri_stat, _ := status.New(codes.Unavailable, "foo").WithDetails(ri)
	ri_msg := ErrDetails{}
	ri_msg.RetryInfo = ri
	ri_err := ri_stat.Err()
	ri_expected := APIError{
		err:     ri_err,
		status:  ri_stat,
		details: ri_msg,
	}
	ri_actual, _ := FromError(ri_err)

	res := &errdetails.ResourceInfo{}
	res.ResourceName = "ResoureInfo"
	res.ResourceType = "Random"
	res.Owner = "Client"
	res.Description = "Directory not found"
	res_msg := ErrDetails{}
	res_msg.ResourceInfo = res
	res_stat, _ := status.New(codes.NotFound, "Missing directory").WithDetails(res)
	res_err := res_stat.Err()
	res_expected := APIError{
		err:     res_err,
		status:  res_stat,
		details: res_msg,
	}
	res_actual, _ := FromError(res_err)

	req := &errdetails.RequestInfo{}
	req.RequestId = "foo"
	req.ServingData = "bar"
	req_msg := ErrDetails{}
	req_msg.RequestInfo = req
	req_stat, _ := status.New(codes.Canceled, "Request cancelled by client").WithDetails(req)
	req_err := req_stat.Err()
	req_expected := APIError{
		err:     req_err,
		status:  req_stat,
		details: req_msg,
	}
	req_actual, _ := FromError(req_err)

	deb := &errdetails.DebugInfo{}
	deb_msg := ErrDetails{}
	var stack []string = []string{"foo", "bar"}
	deb.StackEntries = stack
	deb.Detail = "stack details"
	deb_msg.DebugInfo = deb
	deb_stat, _ := status.New(codes.Internal, "Here is the debug info").WithDetails(deb)
	deb_err := deb_stat.Err()
	deb_expected := APIError{
		err:     deb_err,
		status:  deb_stat,
		details: deb_msg,
	}
	deb_actual, _ := FromError(deb_err)

	help := &errdetails.Help{}
	help_msg := ErrDetails{}
	help_link := &errdetails.Help_Link{}
	help_link.Description = "foo"
	help_link.Url = "https://bar"
	help.Links = append(help.Links, help_link)
	help_stat, _ := status.New(codes.ResourceExhausted, "Help Info").WithDetails(help)
	help_err := help_stat.Err()
	help_expected := APIError{
		err:     help_err,
		status:  help_stat,
		details: help_msg,
	}
	help_actual, _ := FromError(help_err)

	lo := &errdetails.LocalizedMessage{}
	lo.Locale = "foo"
	lo.Message = "bar"
	lo_stat, _ := status.New(codes.Unknown, "Localized Message").WithDetails(lo)
	lo_msg := ErrDetails{}
	lo_msg.LocalizedMesage = lo
	lo_err := lo_stat.Err()
	lo_expected := APIError{
		err:     lo_err,
		status:  lo_stat,
		details: lo_msg,
	}
	lo_actual, _ := FromError(lo_err)

	tests := []test{
		{name: "BadRequest", want: br_expected.Error(), got: br_actual.Error()},
		{name: "QuotaFailure", want: qf_expected.Error(), got: qf_actual.Error()},
		{name: "PreconditionFailure", want: pf_expected.Error(), got: pf_actual.Error()},
		{name: "RetryInfo", want: ri_expected.Error(), got: ri_actual.Error()},
		{name: "ResourceInfo", want: res_expected.Error(), got: res_actual.Error()},
		{name: "RequestInfo", want: req_expected.Error(), got: req_actual.Error()},
		{name: "DebugInfo", want: deb_expected.Error(), got: deb_actual.Error()},
		{name: "Help", want: help_expected.Error(), got: help_actual.Error()},
		{name: "LocalizedMessage", want: lo_expected.Error(), got: lo_actual.Error()},
	}
	for _, tc := range tests {
		if !assert.Equal(t, tc.want, tc.got) {
			fmt.Println("I want", tc.want, "but I got:", tc.got)
		}
	}
}
