package gax

import (
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// Represents a GRPC call stub.
type APICall func(context.Context, interface{}) (interface{}, error)

// scaleDuration returns the product of a and mult.
func scaleDuration(a time.Duration, mult float64) time.Duration {
	ns := float64(a) * mult
	return time.Duration(ns)
}

// stubWithRetry returns a wrapper for stub with an exponential backoff retry
// mechanism based on the values provided in retrySettings.
func stubWithRetry(stub APICall, retrySettings retrySettings) APICall {
	return func(ctx context.Context, req interface{}) (resp interface{}, err error) {
		backoffSettings := retrySettings.backoffSettings
		// Forces ctx to expire after a deadline.
		ctx, _ = context.WithTimeout(ctx, backoffSettings.totalTimeout)

		delay := backoffSettings.delayTimeoutSettings.initial
		timeout := backoffSettings.rpcTimeoutSettings.initial

		for {
			// If the deadline is exceeded...
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			timeoutCtx, _ := context.WithTimeout(ctx, backoffSettings.rpcTimeoutSettings.max)
			timeoutCtx, _ = context.WithTimeout(timeoutCtx, timeout)
			resp, err = stub(timeoutCtx, req)
			code := grpc.Code(err)
			if code == codes.OK {
				return resp, err
			}
			if !retrySettings.retryCodes[code] {
				return nil, err
			}
			delayCtx, _ := context.WithTimeout(ctx, backoffSettings.delayTimeoutSettings.max)
			delayCtx, _ = context.WithTimeout(delayCtx, delay)
			<-delayCtx.Done()

			delay = scaleDuration(delay, backoffSettings.delayTimeoutSettings.multiplier)
			timeout = scaleDuration(timeout, backoffSettings.rpcTimeoutSettings.multiplier)
		}
		return
	}
}

// stubWithTimeout returns a wrapper for stub with a timeout applied to its
// context.
func stubWithTimeout(stub APICall, timeout time.Duration) APICall {
	return func(ctx context.Context, data interface{}) (interface{}, error) {
		childCtx, _ := context.WithTimeout(ctx, timeout)
		return stub(childCtx, data)
	}
}

// CreateAPICall returns a wrapper for stub governed by the values provided in
// settings.
func CreateAPICall(stub APICall, opts ...CallOption) APICall {
	settings := &callSettings{}
	callOptions(opts).Resolve(settings)
	if len(settings.retrySettings.retryCodes) > 0 {
		return stubWithRetry(stub, settings.retrySettings)
	}
	return stubWithTimeout(stub, settings.timeout)
}
