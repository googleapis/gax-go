package gax

import (
	"time"

	"google.golang.org/grpc/codes"
)

type CallOption interface {
	Resolve(*callSettings)
}

type callOptions []CallOption

func (opts callOptions) Resolve(s *callSettings) *callSettings {
	for _, opt := range opts {
		opt.Resolve(s)
	}
	return s
}

// Encapsulates the call settings for a particular API call.
type callSettings struct {
	timeout       time.Duration
	retrySettings retrySettings
}

// Per-call configurable settings for retrying upon transient failure.
type retrySettings struct {
	retryCodes      map[codes.Code]bool
	backoffSettings backoffSettings
}

// Parameters to the exponential backoff algorithm for retrying.
type backoffSettings struct {
	delayTimeoutSettings multipliableDuration
	rpcTimeoutSettings   multipliableDuration
	totalTimeout         time.Duration
}

type multipliableDuration struct {
	initial    time.Duration
	max        time.Duration
	multiplier float64
}

type withTimeout time.Duration

func (w withTimeout) Resolve(s *callSettings) {
	s.timeout = time.Duration(w)
}

// WithTimeout sets the client-side timeout for API calls if the call isn't
// retrying.
func WithTimeout(timeout time.Duration) CallOption {
	return withTimeout(timeout)
}

type withRetryCodes []codes.Code

func (w withRetryCodes) Resolve(s *callSettings) {
	s.retrySettings.retryCodes = make(map[codes.Code]bool)
	for _, code := range []codes.Code(w) {
		s.retrySettings.retryCodes[code] = true
	}
}

// WithRetryCodes sets a list of Google API canonical error codes upon which a
// retry should be attempted. If nil, the call will not retry.
func WithRetryCodes(retryCodes []codes.Code) CallOption {
	return withRetryCodes(retryCodes)
}

type withDelayTimeoutSettings multipliableDuration

func (w withDelayTimeoutSettings) Resolve(s *callSettings) {
	s.retrySettings.backoffSettings.delayTimeoutSettings = multipliableDuration(w)
}

// WithDelayTimeoutSettings specifies:
// - The initial delay time, in milliseconds, between the completion of
//   the first failed request and the initiation of the first retrying
//   request.
// - The multiplier by which to increase the delay time between the
//   completion of failed requests, and the initiation of the subsequent
//   retrying request.
// - The maximum delay time, in milliseconds, between requests. When this
//   value is reached, `RetryDelayMultiplier` will no longer be used to
//   increase delay time.
func WithDelayTimeoutSettings(initial time.Duration, max time.Duration, multiplier float64) CallOption {
	return withDelayTimeoutSettings(multipliableDuration{initial, max, multiplier})
}

type withRPCTimeoutSettings multipliableDuration

func (w withRPCTimeoutSettings) Resolve(s *callSettings) {
	s.retrySettings.backoffSettings.rpcTimeoutSettings = multipliableDuration(w)
}

// WithRPCTimeoutSettings specifies:
// - The initial timeout parameter to the request.
// - The multiplier by which to increase the timeout parameter between
//   failed requests.
// - The maximum timeout parameter, in milliseconds, for a request. When
//   this value is reached, `RPCTimeoutMultiplier` will no longer be used
//   to increase the timeout.
func WithRPCTimeoutSettings(initial time.Duration, max time.Duration, multiplier float64) CallOption {
	return withRPCTimeoutSettings(multipliableDuration{initial, max, multiplier})
}

type withTotalRetryTimeout time.Duration

func (w withTotalRetryTimeout) Resolve(s *callSettings) {
	s.retrySettings.backoffSettings.totalTimeout = time.Duration(w)
}

// WithTotalRetryTimeout sets the total time, in milliseconds, starting from
// when the initial request is sent, after which an error will be returned
// regardless of the retrying attempts made meanwhile.
func WithTotalRetryTimeout(totalRetryTimeout time.Duration) CallOption {
	return withTotalRetryTimeout(totalRetryTimeout)
}
