package gax

import (
	"reflect"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
)

func TestCallOptionsPieceByPiece(t *testing.T) {
	expected := &callSettings{
		time.Second * 1,
		retrySettings{
			map[codes.Code]bool{codes.Unavailable: true, codes.DeadlineExceeded: true},
			backoffSettings{
				multipliableDuration{time.Second * 2, time.Second * 4, 3.0},
				multipliableDuration{time.Second * 5, time.Second * 7, 6.0},
				time.Second * 8,
			},
		},
	}

	settings := &callSettings{}
	opts := []CallOption{
		WithTimeout(time.Second * 1),
		WithRetryCodes([]codes.Code{codes.Unavailable, codes.DeadlineExceeded}),
		WithDelayTimeoutSettings(time.Second*2, time.Second*4, 3.0),
		WithRPCTimeoutSettings(time.Second*5, time.Second*7, 6.0),
		WithTotalRetryTimeout(time.Second * 8),
	}
	callOptions(opts).resolve(settings)

	if !reflect.DeepEqual(settings, expected) {
		t.Errorf("settings don't match their expected configuration")
	}
}
