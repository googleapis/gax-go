package gax

// code snippet credit to @ndietz
import (
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

//Hold the google/rpc/error_details.proto messages.
type ErrDetails struct {
	BadRequest          *errdetails.BadRequest
	PreconditionFailure *errdetails.PreconditionFailure
	QuotaFailure        *errdetails.QuotaFailure
	RetryInfo           *errdetails.RetryInfo
	ResourceInfo        *errdetails.ResourceInfo
	RequestInfo         *errdetails.RequestInfo
	DebugInfo           *errdetails.DebugInfo
	Help                *errdetails.Help
	LocalizedMesage     *errdetails.LocalizedMessage

	Unknown []interface{}
}

type APIError struct {
	err     error
	status  *status.Status
	details ErrDetails
}

func (a *APIError) Details() ErrDetails {
	return a.details
}

func (a *APIError) Unwrap() error {
	return a.err
}

func (a *APIError) Error() string {

	s, _ := protojson.Marshal(a.details.BadRequest)

	return string(s)
}

func (a *APIError) GRPCStatus() *status.Status {
	return a.status
}

func FromError(err error) (*APIError, bool) {
	if err == nil {
		return nil, false
	}
	msg := ErrDetails{}
	st, ok := status.FromError(err)
	if !ok {
		return nil, false
	}
	for _, d := range st.Details() {
		switch d := d.(type) {
		case *errdetails.BadRequest:
			msg.BadRequest = d
		case *errdetails.PreconditionFailure:
			msg.PreconditionFailure = d
		case *errdetails.QuotaFailure:
			msg.QuotaFailure = d
		case *errdetails.RetryInfo:
			msg.RetryInfo = d
		case *errdetails.Help:
			msg.Help = d
		case *errdetails.ResourceInfo:
			msg.ResourceInfo = d
		case *errdetails.RequestInfo:
			msg.RequestInfo = d
		case *errdetails.DebugInfo:
			msg.DebugInfo = d
		case *errdetails.LocalizedMessage:
			msg.LocalizedMesage = d
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
