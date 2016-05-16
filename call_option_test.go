package gax

import (
	"reflect"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
)

func TestCallOptions(t *testing.T) {
	expected := &CallSettings{
		time.Second * 1,
		RetrySettings{
			map[codes.Code]bool{codes.Unavailable: true, codes.DeadlineExceeded: true},
			BackoffSettings{
				MultipliableDuration{time.Second * 2, time.Second * 4, 3.0},
				MultipliableDuration{time.Second * 5, time.Second * 7, 6.0},
			},
		},
	}

	settings := &CallSettings{}
	opts := []CallOption{
		WithTimeout(time.Second * 1),
		WithRetryCodes([]codes.Code{codes.Unavailable, codes.DeadlineExceeded}),
		WithDelayTimeoutSettings(time.Second*2, time.Second*4, 3.0),
		WithRPCTimeoutSettings(time.Second*5, time.Second*7, 6.0),
	}
	callOptions(opts).Resolve(settings)

	if !reflect.DeepEqual(settings, expected) {
		t.Errorf("piece-by-piece settings don't match their expected configuration")
	}

	settings = &CallSettings{}
	expected.Resolve(settings)

	if !reflect.DeepEqual(settings, expected) {
		t.Errorf("whole settings don't match their expected configuration")
	}

	expected.RetrySettings.RetryCodes[codes.FailedPrecondition] = true
	if _, ok := settings.RetrySettings.RetryCodes[codes.FailedPrecondition]; ok {
		t.Errorf("unexpected modification in the RetryCodes map")
	}
}
