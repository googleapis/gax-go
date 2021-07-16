package gax

// code snippet credit to @ndietz
import (
	"encoding/json"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/status"
)

//hold the google/rpc/error_details.proto messages.
type ErrDetails struct {
	BadRequest          *errdetails.BadRequest          `json:",omitempty"`
	PreconditionFailure *errdetails.PreconditionFailure `json:",omitempty"`
	QuotaFailure        *errdetails.QuotaFailure        `json:",omitempty"`
	RetryInfo           *errdetails.RetryInfo           `json:",omitempty"`
	ResourceInfo        *errdetails.ResourceInfo        `json:",omitempty"`
	RequestInfo         *errdetails.RequestInfo         `json:",omitempty"`
	DebugInfo           *errdetails.DebugInfo           `json:",omitempty"`
	Help                *errdetails.Help                `json:",omitempty"`
	LocalizedMesage     *errdetails.LocalizedMessage    `json:",omitempty"`

	//store unidentifiable error details
	Unknown []interface{} `json:",omitempty"`
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

	strr, _ := json.Marshal(a.details)
	return a.err.Error() + "\n" + "Here are the details: " + "\n" + string(strr)

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
	if ok {
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
	return nil, false
}
