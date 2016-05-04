package gax

import (
	"reflect"
	"testing"
)

func TestClientOptionsPieceByPiece(t *testing.T) {
	expected := &clientSettings{
		"myapi",
		"v0.1.0",
		"https://example.com:443",
		[]string{"https://example.com/auth/helloworld", "https://example.com/auth/otherthing"},
	}

	settings := &clientSettings{}
	opts := []ClientOption{
		WithAPIName("myapi"),
		WithAPIVersion("v0.1.0"),
		WithEndpoint("https://example.com:443"),
		WithScopes("https://example.com/auth/helloworld", "https://example.com/auth/otherthing"),
	}
	clientOptions(opts).resolve(settings)

	if !reflect.DeepEqual(settings, expected) {
		t.Errorf("settings don't match their expected configuration")
	}
}
