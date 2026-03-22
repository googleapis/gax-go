// Copyright 2026, Google Inc.
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//     * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package gax

import (
	"bytes"
	"context"
	"log/slog"
	"os/exec"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestNewClientMetrics(t *testing.T) {
	// Setup a dummy provider for testing.
	customProvider := noop.NewMeterProvider()

	tests := []struct {
		name          string
		opts          []TelemetryOption
		wantScopeAttr map[string]string
		wantDataAttr  map[string]string
		useCustom     bool
	}{
		{
			name: "default boundaries and global provider",
			opts: []TelemetryOption{
				WithTelemetryAttributes(map[string]string{
					ClientArtifact: "test-lib",
					ClientVersion:  "v1.0.0",
				}),
			},
			wantScopeAttr: map[string]string{},
			wantDataAttr:  map[string]string{},
		},
		{
			name: "custom provider and custom boundaries",
			opts: []TelemetryOption{
				WithTelemetryAttributes(map[string]string{
					ClientArtifact: "test-lib-2",
					ClientVersion:  "v1.0.1",
				}),
				WithExplicitBucketBoundaries([]float64{10, 20, 30}),
			},
			useCustom:     true,
			wantScopeAttr: map[string]string{},
			wantDataAttr:  map[string]string{},
		},
		{
			name: "with static attributes",
			opts: []TelemetryOption{
				WithTelemetryAttributes(map[string]string{
					ClientArtifact: "test-lib-3",
					ClientVersion:  "v1.0.2",
					ClientService:  "myservice",
					RPCSystem:      "grpc",
					URLDomain:      "test.domain",
					"ignored.key":  "ignored",
				}),
			},
			wantScopeAttr: map[string]string{
				"gcp.client.service": "myservice",
			},
			wantDataAttr: map[string]string{
				"rpc.system.name": "grpc",
				"url.domain":      "test.domain",
			},
		},
		{
			name: "with logger",
			opts: []TelemetryOption{
				WithTelemetryAttributes(map[string]string{
					ClientArtifact: "test-lib",
					ClientVersion:  "v1.0.0",
				}),
				WithTelemetryLogger(slog.Default()),
			},
			wantScopeAttr: map[string]string{},
			wantDataAttr:  map[string]string{},
		},
		{
			name: "with nil logger",
			opts: []TelemetryOption{
				WithTelemetryAttributes(map[string]string{
					ClientArtifact: "test-lib",
					ClientVersion:  "v1.0.0",
				}),
				WithTelemetryLogger(nil),
			},
			wantScopeAttr: map[string]string{},
			wantDataAttr:  map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmOpts := tt.opts
			if tt.useCustom {
				cmOpts = append(cmOpts, WithMeterProvider(customProvider))
				cm := NewClientMetrics(cmOpts...)
				if cm == nil || cm.durationHistogram() == nil {
					t.Fatalf("expected initialized metrics")
				}
				return // we can't observe noop provider, so just verify it doesn't panic
			}

			// Setup SDK for collection
			reader := sdkmetric.NewManualReader()
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			// Always override provider to our test provider for observability
			cmOpts = append(cmOpts, WithMeterProvider(provider))

			cm := NewClientMetrics(cmOpts...)
			if cm == nil {
				t.Fatalf("NewClientMetrics returned nil")
			}
			if cm.durationHistogram() == nil {
				t.Fatalf("expected Float64Histogram to be initialized, got nil")
			}

			// Record a dummy value so we can collect the metrics and inspect Scope attributes
			cm.durationHistogram().Record(context.Background(), 1.0, metric.WithAttributes(cm.attributes()...))

			var rm metricdata.ResourceMetrics
			if err := reader.Collect(context.Background(), &rm); err != nil {
				t.Fatalf("Failed to collect metrics: %v", err)
			}

			if len(rm.ScopeMetrics) != 1 {
				t.Fatalf("Expected 1 ScopeMetrics, got %d", len(rm.ScopeMetrics))
			}
			sm := rm.ScopeMetrics[0]

			// Verify Exact Scope Attributes
			scopeAttrs := make(map[string]string)
			for _, set := range sm.Scope.Attributes.ToSlice() {
				scopeAttrs[string(set.Key)] = set.Value.AsString()
			}

			if len(scopeAttrs) != len(tt.wantScopeAttr) {
				t.Errorf("expected %d scope attributes, got %d (%v)", len(tt.wantScopeAttr), len(scopeAttrs), scopeAttrs)
			}
			for wantK, wantV := range tt.wantScopeAttr {
				if gotV, ok := scopeAttrs[wantK]; !ok || gotV != wantV {
					t.Errorf("expected scope attribute %s=%s, got %s", wantK, wantV, gotV)
				}
			}

			// Verify Exact Datapoint Attributes from the collected metric
			if len(sm.Metrics) != 1 {
				t.Fatalf("Expected 1 Metric, got %d", len(sm.Metrics))
			}
			data := sm.Metrics[0].Data.(metricdata.Histogram[float64])
			if len(data.DataPoints) != 1 {
				t.Fatalf("Expected 1 DataPoint, got %d", len(data.DataPoints))
			}
			dp := data.DataPoints[0]

			dpAttrs := make(map[string]string)
			for _, set := range dp.Attributes.ToSlice() {
				dpAttrs[string(set.Key)] = set.Value.AsString()
			}

			if len(dpAttrs) != len(tt.wantDataAttr) {
				t.Errorf("expected %d datapoint attributes, got %d (%v)", len(tt.wantDataAttr), len(dpAttrs), dpAttrs)
			}
			for wantK, wantV := range tt.wantDataAttr {
				if gotV, ok := dpAttrs[wantK]; !ok || gotV != wantV {
					t.Errorf("expected datapoint attribute %s=%s, got %s", wantK, wantV, gotV)
				}
			}
		})
	}
}

func TestNewClientMetrics_GlobalFallback(t *testing.T) {
	opts := []TelemetryOption{
		WithTelemetryAttributes(map[string]string{
			ClientArtifact: "test-global-fallback",
		}),
	}
	cm := NewClientMetrics(opts...)
	if cm == nil {
		t.Fatalf("expected non-nil ClientMetrics")
	}
	if cm.durationHistogram() == nil {
		t.Errorf("expected non-nil duration histogram")
	}
}

func TestNewClientMetrics_InitializationError(t *testing.T) {
	// Setup SDK
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	t.Run("with logger", func(t *testing.T) {
		// Capture log output
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		opts := []TelemetryOption{
			WithMeterProvider(provider),
			WithTelemetryAttributes(map[string]string{
				ClientArtifact: "test-error",
			}),
			WithExplicitBucketBoundaries([]float64{10.0, 5.0}), // Invalid boundaries trigger error
			WithTelemetryLogger(logger),
		}

		cm := NewClientMetrics(opts...)
		if cm == nil {
			t.Fatalf("expected non-nil ClientMetrics")
		}

		// Trigger lazy initialization, which should fail and log
		cm.durationHistogram()

		logOutput := buf.String()
		if !strings.Contains(logOutput, "failed to initialize OTel duration histogram") {
			t.Errorf("expected initialization error to be logged, got: %s", logOutput)
		}
	})

	t.Run("without logger", func(t *testing.T) {
		opts := []TelemetryOption{
			WithMeterProvider(provider),
			WithTelemetryAttributes(map[string]string{
				ClientArtifact: "test-error",
			}),
			WithExplicitBucketBoundaries([]float64{10.0, 5.0}), // Invalid boundaries trigger error
		}

		cm := NewClientMetrics(opts...)
		if cm == nil {
			t.Fatalf("expected non-nil ClientMetrics")
		}

		// Trigger lazy initialization, which should fail but NOT panic
		cm.durationHistogram()
	})
}

func TestTelemetryConfigKeys(t *testing.T) {
	// These keys are referenced by generated client code. They must not be changed,
	// otherwise generated code will fail to compile or pass attributes incorrectly.
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"ClientService", ClientService, "client_service"},
		{"ClientVersion", ClientVersion, "client_version"},
		{"ClientArtifact", ClientArtifact, "client_artifact"},
		{"RPCSystem", RPCSystem, "rpc_system"},
		{"URLDomain", URLDomain, "url_domain"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("Config key %s changed: got %q, want %q. This will break generated clients.", tt.name, tt.got, tt.want)
			}
		})
	}
}

// TestNoSDKImport verifies that the go.opentelemetry.io/otel/sdk package
// is not imported by the production code in this module.
// It is perfectly fine for test code (*_test.go) to import the SDK.
func TestNoSDKImport(t *testing.T) {
	cmd := exec.Command("go", "list", "-f", "{{.Imports}}", ".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list failed: %v\nOutput: %s", err, string(out))
	}

	imports := string(out)
	if strings.Contains(imports, "go.opentelemetry.io/otel/sdk") {
		t.Errorf("Production code imports the OpenTelemetry SDK (go.opentelemetry.io/otel/sdk). This is forbidden.")
	}
}

func TestTransportTelemetry(t *testing.T) {
	ctx := context.Background()
	data := &TransportTelemetryData{}
	data.SetServerAddress("localhost")
	data.SetServerPort(8080)
	data.SetResponseStatusCode(200)

	ctx = InjectTransportTelemetry(ctx, data)
	got := ExtractTransportTelemetry(ctx)
	if got == nil {
		t.Errorf("ExtractTransportTelemetry() = nil, want %v", data)
	}
	if got != data {
		t.Errorf("ExtractTransportTelemetry() = %v, want %v", got, data)
	}
	if got.ServerAddress() != "localhost" {
		t.Errorf("got.ServerAddress() = %q, want %q", got.ServerAddress(), "localhost")
	}
	if got.ServerPort() != 8080 {
		t.Errorf("got.ServerPort() = %d, want %d", got.ServerPort(), 8080)
	}
	if got.ResponseStatusCode() != 200 {
		t.Errorf("got.ResponseStatusCode() = %d, want %d", got.ResponseStatusCode(), 200)
	}
}

func TestOpenTelemetryUnaryClientInterceptor(t *testing.T) {
	interceptor := OpenTelemetryUnaryClientInterceptor()

	// 1. Without TransportTelemetryData in context
	ctx := context.Background()
	invokerCalled := false
	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		invokerCalled = true
		return nil
	}
	err := interceptor(ctx, "/TestService/TestMethod", nil, nil, nil, invoker)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !invokerCalled {
		t.Errorf("expected invoker to be called")
	}

	// 2. With TransportTelemetryData in context
	data := &TransportTelemetryData{}
	ctx = InjectTransportTelemetry(context.Background(), data)
	invokerCalled = false
	invokerWithTarget := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		invokerCalled = true
		return nil
	}

	// We need a real ClientConn to call cc.Target()
	cc, err := grpc.NewClient("127.0.0.1:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer cc.Close()

	err = interceptor(ctx, "/TestService/TestMethod", nil, nil, cc, invokerWithTarget)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !invokerCalled {
		t.Errorf("expected invoker to be called")
	}

	if data.ServerAddress() != "127.0.0.1" {
		t.Errorf("expected ServerAddress to be 127.0.0.1, got %q", data.ServerAddress())
	}
	if data.ServerPort() != 8080 {
		t.Errorf("expected ServerPort to be 8080, got %d", data.ServerPort())
	}
}

func TestExtractHostPort(t *testing.T) {
	tests := []struct {
		target   string
		wantHost string
		wantPort int
	}{
		{"127.0.0.1:8080", "127.0.0.1", 8080},
		{"[::1]:9090", "::1", 9090},
		{"dns:///localhost:443", "localhost", 443},
		{"xds:///my-service:80", "my-service", 80},
		{"https://example.com:443", "example.com", 443},
		{"invalid-target", "invalid-target", 0},             // missing port
		{"localhost:notanumber", "localhost:notanumber", 0}, // port is not int
	}

	for _, tt := range tests {
		gotHost, gotPort := extractHostPort(tt.target)
		if gotHost != tt.wantHost {
			t.Errorf("extractHostPort(%q) host = %q, want %q", tt.target, gotHost, tt.wantHost)
		}
		if gotPort != tt.wantPort {
			t.Errorf("extractHostPort(%q) port = %d, want %d", tt.target, gotPort, tt.wantPort)
		}
	}
}

func TestClientMetrics_NilReceiver(t *testing.T) {
	var cm *ClientMetrics
	if cm.durationHistogram() != nil {
		t.Errorf("expected nil durationHistogram for nil receiver")
	}
	if cm.attributes() != nil {
		t.Errorf("expected nil attributes for nil receiver")
	}

	cm = &ClientMetrics{} // nil .get func
	if cm.durationHistogram() != nil {
		t.Errorf("expected nil durationHistogram for uninitialized ClientMetrics")
	}
	if cm.attributes() != nil {
		t.Errorf("expected nil attributes for uninitialized ClientMetrics")
	}
}
