package gax

import (
	"reflect"
	"testing"
	"time"

	"google.golang.org/grpc"
)

func TestClientOptionsPieceByPiece(t *testing.T) {
	expected := &ClientSettings{
		"myapi",
		"v0.1.0",
		"https://example.com:443",
		[]string{"https://example.com/auth/helloworld", "https://example.com/auth/otherthing"},
		map[string][]CallOption{"ListWorlds": []CallOption{WithTimeout(3 * time.Second)}},
		[]grpc.DialOption{},
	}

	settings := &ClientSettings{}
	opts := []ClientOption{
		WithAppName("myapi"),
		WithAppVersion("v0.1.0"),
		WithEndpoint("https://example.com:443"),
		WithScopes("https://example.com/auth/helloworld", "https://example.com/auth/otherthing"),
		WithCallOptions(map[string][]CallOption{"ListWorlds": []CallOption{WithTimeout(3 * time.Second)}}),
		WithDialOptions(), // Can't compare function signatures for equality.
	}
	clientOptions(opts).Resolve(settings)

	if !reflect.DeepEqual(settings, expected) {
		t.Errorf("piece-by-piece settings don't match their expected configuration")
	}

	settings = &ClientSettings{}
	expected.Resolve(settings)

	if !reflect.DeepEqual(settings, expected) {
		t.Errorf("whole settings don't match their expected configuration")
	}

	expected.Scopes[0] = "hello"
	if settings.Scopes[0] == expected.Scopes[0] {
		t.Errorf("unexpected modification in Scopes array")
	}
	expected.CallOptions["Impossible"] = []CallOption{WithTimeout(42 * time.Second)}
	if _, ok := settings.CallOptions["Impossible"]; ok {
		t.Errorf("unexpected modification in CallOptions map")
	}
}
