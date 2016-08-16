package gax

import (
	"fmt"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// A user defined call stub.
type APICall func(context.Context) error

// scaleDuration scales the duration d by mult.Multiplier,
// making sure to not exceed mult.Max.
func scaleDuration(d time.Duration, mult MultipliableDuration) time.Duration {
	nd := time.Duration(float64(d) * mult.Multiplier)
	if nd > mult.Max {
		nd = mult.Max
	}
	return nd
}

// ensureTimeout returns a context with the given timeout applied if there
// is no deadline on the context.
func ensureTimeout(ctx context.Context, timeout time.Duration) context.Context {
	if _, ok := ctx.Deadline(); !ok {
		ctx, _ = context.WithTimeout(ctx, timeout)
	}
	return ctx
}

// invokeWithRetry calls stub using an exponential backoff retry mechanism
// based on the values provided in retrySettings.
func invokeWithRetry(ctx context.Context, stub APICall, callSettings CallSettings) error {
	retrySettings := callSettings.RetrySettings
	backoffSettings := callSettings.RetrySettings.BackoffSettings
	delay := backoffSettings.DelayTimeoutSettings.Initial
	timeout := backoffSettings.RPCTimeoutSettings.Initial

	for {
		timeoutCtx, _ := context.WithTimeout(ctx, timeout)
		err := stub(timeoutCtx)
		code := grpc.Code(err)
		if code == codes.OK {
			return nil
		}
		if !retrySettings.RetryCodes[code] {
			return InvokeError{grpcErr: err}
		}

		select {
		case <-ctx.Done():
			return InvokeError{ctxErr: ctx.Err(), grpcErr: err}
		case <-time.After(delay):
		}

		delay = scaleDuration(delay, backoffSettings.DelayTimeoutSettings)
		timeout = scaleDuration(timeout, backoffSettings.RPCTimeoutSettings)
	}
}

// Invoke calls stub with a child of context modified by the specified options.
// If the returned error is not nil, it will be an InvokeError.
func Invoke(ctx context.Context, stub APICall, opts ...CallOption) error {
	var settings CallSettings
	callOptions(opts).Resolve(&settings)
	ctx = ensureTimeout(ctx, settings.Timeout)
	if len(settings.RetrySettings.RetryCodes) > 0 {
		return invokeWithRetry(ctx, stub, settings)
	}
	if err := stub(ctx); err != nil {
		return InvokeError{grpcErr: err}
	}
	return nil
}

type InvokeError struct {
	// grpcErr is always non-nil
	ctxErr, grpcErr error
}

func (e InvokeError) Error() string {
	if e.ctxErr != nil {
		return fmt.Sprintf("%s (last retry error: %s)", e.ctxErr, e.grpcErr)
	}
	return e.grpcErr.Error()
}

func (e InvokeError) GRPCError() error { return e.grpcErr }
