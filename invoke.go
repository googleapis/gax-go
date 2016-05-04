package gax

import (
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// A user defined call stub.
type APICall func(context.Context) error

// scaleDuration returns the product of a and mult.
func scaleDuration(a time.Duration, mult float64) time.Duration {
	ns := float64(a) * mult
	return time.Duration(ns)
}

// invokeWithRetry calls stub using an exponential backoff retry mechanism
// based on the values provided in retrySettings.
func invokeWithRetry(ctx context.Context, stub APICall, retrySettings retrySettings) error {
	backoffSettings := retrySettings.backoffSettings
	// Forces ctx to expire after a deadline.
	childCtx, _ := context.WithTimeout(ctx, backoffSettings.totalTimeout)
	delay := backoffSettings.delayTimeoutSettings.initial
	timeout := backoffSettings.rpcTimeoutSettings.initial
	for {
		// If the deadline is exceeded...
		if childCtx.Err() != nil {
			return childCtx.Err()
		}
		timeoutCtx, _ := context.WithTimeout(childCtx, backoffSettings.rpcTimeoutSettings.max)
		timeoutCtx, _ = context.WithTimeout(timeoutCtx, timeout)
		err := stub(timeoutCtx)
		code := grpc.Code(err)
		if code == codes.OK {
			return nil
		}
		if !retrySettings.retryCodes[code] {
			return err
		}
		delayCtx, _ := context.WithTimeout(childCtx, backoffSettings.delayTimeoutSettings.max)
		delayCtx, _ = context.WithTimeout(delayCtx, delay)
		<-delayCtx.Done()

		delay = scaleDuration(delay, backoffSettings.delayTimeoutSettings.multiplier)
		timeout = scaleDuration(timeout, backoffSettings.rpcTimeoutSettings.multiplier)
	}
}

// invokeWithTimeout calls stub with a timeout applied to its context.
func invokeWithTimeout(ctx context.Context, stub APICall, timeout time.Duration) error {
	childCtx, _ := context.WithTimeout(ctx, timeout)
	return stub(childCtx)
}

// Invoke calls stub with a child of context modified by the specified options.
func Invoke(ctx context.Context, stub APICall, opts ...CallOption) error {
	settings := &callSettings{}
	callOptions(opts).resolve(settings)
	if len(settings.retrySettings.retryCodes) > 0 {
		return invokeWithRetry(ctx, stub, settings.retrySettings)
	}
	return invokeWithTimeout(ctx, stub, settings.timeout)
}
