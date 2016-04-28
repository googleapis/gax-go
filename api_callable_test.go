package gax

import (
	"testing"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var (
	testCallSettings = []CallOption{
		WithRetryCodes([]codes.Code{codes.Unavailable, codes.DeadlineExceeded}),
		// initial, max, multiplier
		WithDelayTimeoutSettings(100*time.Millisecond, 300*time.Millisecond, 1.5),
		WithRPCTimeoutSettings(50*time.Millisecond, 500*time.Millisecond, 3.0),
		WithTotalRetryTimeout(1000 * time.Millisecond),
	}
)

func TestCreateAPICallWithTimeout(t *testing.T) {
	ctx := context.Background()
	var ok bool
	CreateAPICall(func(ctx context.Context, req interface{}) (interface{}, error) {
		_, ok = ctx.Deadline()
		return nil, nil
	}, WithTimeout(10000*time.Millisecond))(ctx, nil)
	if !ok {
		t.Errorf("expected call to have an assigned timeout")
	}
}

func TestCreateApiCallWithOKResponseWithTimeout(t *testing.T) {
	ctx := context.Background()
	resp, err := CreateAPICall(func(ctx context.Context, req interface{}) (interface{}, error) {
		return 42, nil
	}, WithTimeout(10000*time.Millisecond))(ctx, nil)
	if resp.(int) != 42 || err != nil {
		t.Errorf("expected call to return (42, nil)")
	}
}

func TestCreateApiCallWithDeadlineAfterRetries(t *testing.T) {
	ctx := context.Background()
	count := 0

	now := time.Now()
	expectedTimeout := []time.Duration{
		0,
		150 * time.Millisecond,
		450 * time.Millisecond,
	}

	_, err := CreateAPICall(func(ctx context.Context, req interface{}) (interface{}, error) {
		t.Log("delta:", time.Now().Sub(now.Add(expectedTimeout[count])))
		if !time.Now().After(now.Add(expectedTimeout[count])) {
			t.Errorf("expected %s to pass before this call", expectedTimeout[count])
		}
		count += 1
		<-ctx.Done()
		return nil, grpc.Errorf(codes.DeadlineExceeded, "")
	}, testCallSettings...)(ctx, nil)
	if count != 3 || err == nil {
		t.Errorf("expected call to retry 3 times and return an error")
	}
}

func TestCreateApiCallWithOKResponseAfterRetries(t *testing.T) {
	ctx := context.Background()
	count := 0

	resp, err := CreateAPICall(func(ctx context.Context, req interface{}) (interface{}, error) {
		count += 1
		if count == 3 {
			return 42, nil
		}
		<-ctx.Done()
		return nil, grpc.Errorf(codes.DeadlineExceeded, "")
	}, testCallSettings...)(ctx, nil)
	if count != 3 || resp.(int) != 42 || err != nil {
		t.Errorf("expected call to retry 3 times and return (42, nil)")
	}
}
