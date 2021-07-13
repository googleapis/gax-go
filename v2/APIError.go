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

// Present the details to the user in as type ErrDetails
func (a *APIError) Details() ErrDetails {
	return a.details
}

// Implement the error Unwrap API
func (a *APIError) Unwrap() error {
	return a.err
}

// Implement the error interface(cleaned up version of Details())
func (a *APIError) Error() string {

	strr, _ := json.Marshal(a.details)
	return a.err.Error() + "\n" + "Here are the details: " + "\n" + string(strr)

}

// Implement the GRPCStatus() method
func (a *APIError) GRPCStatus() *status.Status {
	return a.status
}

// Implement extracting Status from error
func FromError(err error) (*APIError, bool) {
	if err == nil {
		return nil, false
	}
	msg := ErrDetails{}
	//default value of false
	api := true
	//convert err to status
	st, ok := status.FromError(err)
	if ok {
		//append each index in status details to appropriate field in msg
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
			case *errdetails.ResourceInfo:
				msg.ResourceInfo = d
			case *errdetails.RequestInfo:
				msg.RequestInfo = d
			case *errdetails.DebugInfo:
				msg.DebugInfo = d
			case *errdetails.LocalizedMessage:
				msg.LocalizedMesage = d
			}
		}
	} else {
		msg.Unknown = append(msg.Unknown, err.Error())
		api = false
	}
	if !api {
		return &APIError{
			details: msg,
			err:     err,
			status:  st,
		}, false
	}
	return &APIError{
		details: msg,
		err:     err,
		status:  st,
	}, true
}
