package gax

// code snippet credit to @ndietz
import (
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/status"
)

//ErrDetails holds the google/rpc/error_details.proto messages.
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

	Unknown []interface{}
}

type APIError struct {
	err     error
	status  *status.Status
	details ErrDetails
}

//Details presents the error details in an APIError
func (a *APIError) Details() ErrDetails {
	return a.details
}

//Unwrap extracts original error
func (a *APIError) Unwrap() error {
	return a.err
}

//Error creates a readable representation of the APIError
func (a *APIError) Error() string {
	var d strings.Builder
	d.WriteString("Error Details: ")

	if a.details.ErrorInfo != nil {
		d.WriteString("api error: name = ErrorInfo" + " reason = " + a.details.ErrorInfo.Reason +
			" domain = " + a.details.ErrorInfo.Domain + "\n")
	}

	if a.details.BadRequest != nil {
		v := a.details.BadRequest.GetFieldViolations()
		var f []string
		var desc []string
		for _, x := range v {
			f = append(f, x.Field)
			desc = append(desc, x.Description)
		}
		d.WriteString("api error: name = BadRequest" + " field = " + strings.Join(f, " ") +
			" desc = " + strings.Join(desc, " ") + "\n")
	}

	if a.details.PreconditionFailure != nil {
		v := a.details.PreconditionFailure.GetViolations()
		var t []string
		var s []string
		var desc []string
		for _, x := range v {
			t = append(t, x.Type)
			s = append(s, x.Subject)
			desc = append(desc, x.Description)
		}
		d.WriteString("api error: name = PreconditionFailure" + " type = " + strings.Join(t, " ") +
			" subj = " + strings.Join(s, " ") + " desc = " + strings.Join(desc, " ") + "\n")
	}

	if a.details.QuotaFailure != nil {
		v := a.details.QuotaFailure.GetViolations()
		var s []string
		var desc []string
		for _, x := range v {
			s = append(s, x.Subject)
			desc = append(desc, x.Description)
		}
		d.WriteString("api error: name = QuotaFailure" + " subj = " + strings.Join(s, " ") +
			" desc = " + strings.Join(desc, " ") + "\n")
	}

	if a.details.RequestInfo != nil {
		d.WriteString("api error: name = RequestInfo" + " id = " + a.details.RequestInfo.RequestId +
			" data = " + a.details.RequestInfo.ServingData + "\n")
	}

	if a.details.ResourceInfo != nil {
		d.WriteString("api error: name = ResourceInfo" + " type = " + a.details.ResourceInfo.ResourceType +
			" resourcename = " + a.details.ResourceInfo.ResourceName + " owner = " + a.details.ResourceInfo.Owner +
			" desc = " + a.details.ResourceInfo.Description + "\n")

	}
	if a.details.RetryInfo != nil {
		d.WriteString("api error: name = RequestInfo" + " seconds = " + strconv.Itoa(int(a.details.RetryInfo.RetryDelay.Seconds)) +
			" nanos = " + strconv.Itoa(int(a.details.RetryInfo.RetryDelay.Seconds)) + "\n")

	}
	if a.details.Unknown != nil {
		var s []string
		for _, x := range a.details.Unknown {
			s = append(s, fmt.Sprintf("%v", x))
		}
		d.WriteString("api error: name = Unknown" + " desc = " + strings.Join(s, " ") + "\n")

	}
	if a.details.DebugInfo != nil {
		stack := strings.Join(a.details.DebugInfo.StackEntries, " ")
		d.WriteString("api error: name = DebugInfo" + " detail = " + a.details.DebugInfo.Detail + " stack = " + stack + "\n")
	}
	if a.details.Help != nil {
		var desc string
		var url string
		for _, x := range a.details.Help.Links {
			desc = x.Description
			url = x.Url
		}
		d.WriteString("api error: name = Help" + " desc = " + desc + " url = " + url + "\n")
	}
	if a.details.LocalizedMessage != nil {
		d.WriteString("api error: name = LocalizedMessge" + " locale = " + a.details.LocalizedMessage.Locale +
			" msg = " + a.details.LocalizedMessage.Message + "\n")
	}
	return d.String()
}

//GRPCStatus extracts underlying gRPC status in APIError
func (a *APIError) GRPCStatus() *status.Status {
	return a.status
}

//FromError extracts gRPC status from an error and builds an APIError
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
