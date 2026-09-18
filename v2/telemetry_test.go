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
	"errors"
	"log/slog"
	"math"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/gax-go/v2/apierror"
	"github.com/googleapis/gax-go/v2/callctx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/api/googleapi"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
				scopeAttrs[string(set.Key)] = set.Value.String()
			}

			if diff := cmp.Diff(tt.wantScopeAttr, scopeAttrs); diff != "" {
				t.Errorf("Scope attributes mismatch (-want +got):\n%s", diff)
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
				dpAttrs[string(set.Key)] = set.Value.String()
			}

			if diff := cmp.Diff(tt.wantDataAttr, dpAttrs); diff != "" {
				t.Errorf("DataPoint attributes mismatch (-want +got):\n%s", diff)
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
	data.SetHTTPStatusCode(200)

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
	if got.HTTPStatusCode() != 200 {
		t.Errorf("got.HTTPStatusCode() = %d, want %d", got.HTTPStatusCode(), 200)
	}
}

func TestExtractTelemetryErrorInfo(t *testing.T) {
	// Helper to construct a real apierror.APIError with an ErrorInfo
	st := status.New(codes.PermissionDenied, "disabled")
	stWithDetails, _ := st.WithDetails(&errdetails.ErrorInfo{Reason: "SERVICE_DISABLED", Domain: "googleapis.com"})
	apiErr, _ := apierror.FromError(stWithDetails.Err())

	tests := []struct {
		name     string
		setupCtx func() (context.Context, context.CancelFunc)
		err      error
		wantInfo TelemetryErrorInfo
	}{
		{
			name:     "success",
			setupCtx: func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			err:      nil,
			wantInfo: TelemetryErrorInfo{ErrorType: "", StatusCode: "OK"},
		},
		{
			name: "error_cancelled",
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
			err: context.Canceled,
			wantInfo: TelemetryErrorInfo{
				ErrorType:     "CLIENT_CANCELLED",
				StatusCode:    "CANCELED",
				StatusMessage: "context canceled",
			},
		},
		{
			name: "error_deadline",
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithTimeout(context.Background(), 0)
				return ctx, cancel
			},
			err: context.DeadlineExceeded,
			wantInfo: TelemetryErrorInfo{
				ErrorType:     "CLIENT_TIMEOUT",
				StatusCode:    "DEADLINE_EXCEEDED",
				StatusMessage: "context deadline exceeded",
			},
		},
		{
			name:     "error_apierror_reason",
			setupCtx: func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			err:      apiErr,
			wantInfo: TelemetryErrorInfo{
				ErrorType:     "SERVICE_DISABLED",
				StatusCode:    "PERMISSION_DENIED",
				StatusMessage: "disabled",
				Domain:        "googleapis.com",
				Metadata:      nil,
			},
		},
		{
			name:     "error_unknown_type",
			setupCtx: func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			err:      errors.New("random io error"),
			wantInfo: TelemetryErrorInfo{
				ErrorType:     "*errors.errorString",
				StatusCode:    "UNKNOWN",
				StatusMessage: "random io error",
			},
		},
		{
			name:     "error_invalid_grpc_code",
			setupCtx: func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			err:      status.Error(codes.Code(999), "unknown code"),
			wantInfo: TelemetryErrorInfo{
				ErrorType:     "UNKNOWN",
				StatusCode:    "UNKNOWN",
				StatusMessage: "unknown code",
			},
		},
		{
			name:     "error_http_with_message",
			setupCtx: func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			err:      &googleapi.Error{Code: 404, Message: "not found"},
			wantInfo: TelemetryErrorInfo{
				ErrorType:     "404",
				StatusCode:    "UNKNOWN",
				StatusMessage: "not found",
			},
		},
		{
			name:     "error_http_without_message",
			setupCtx: func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			err:      &googleapi.Error{Code: 500, Message: ""},
			wantInfo: TelemetryErrorInfo{
				ErrorType:     "500",
				StatusCode:    "UNKNOWN",
				StatusMessage: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := tt.setupCtx()
			defer cancel()

			got := ExtractTelemetryErrorInfo(ctx, tt.err)
			if diff := cmp.Diff(tt.wantInfo, got, cmp.AllowUnexported(TelemetryErrorInfo{})); diff != "" {
				t.Errorf("ExtractTelemetryErrorInfo() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRecordMetric(t *testing.T) {
	tests := []struct {
		name          string
		method        string
		template      string
		err           error
		wantSum       float64
		wantDataAttr  map[string]string
		nilMetrics    bool
		transportData *TransportTelemetryData
		rpcSystem     string
	}{
		{
			name:       "nil_metrics",
			nilMetrics: true, // Should return early and not panic
		},
		{
			name:     "success",
			method:   "my.service.Method",
			template: "/v1/test/{id}",
			err:      nil,
			wantSum:  1.5,
			wantDataAttr: map[string]string{
				"url.domain":               "test.domain",
				"rpc.system.name":          "grpc",
				"rpc.response.status_code": "OK",
				"rpc.method":               "my.service.Method",
				"url.template":             "/v1/test/{id}",
			},
		},
		{
			name:     "error_recorded",
			method:   "my.service.Method",
			template: "/v1/test/{id}",
			err:      errors.New("random io error"),
			wantSum:  1.5,
			wantDataAttr: map[string]string{
				"url.domain":               "test.domain",
				"rpc.system.name":          "grpc",
				"rpc.response.status_code": "UNKNOWN",
				"error.type":               "*errors.errorString",
				"rpc.method":               "my.service.Method",
				"url.template":             "/v1/test/{id}",
			},
		},
		{
			name:      "http_success_with_status_code",
			method:    "my.service.Method",
			template:  "/v1/test/{id}",
			err:       nil,
			wantSum:   1.5,
			rpcSystem: "http",
			transportData: &TransportTelemetryData{
				serverAddress:  "localhost",
				serverPort:     8080,
				httpStatusCode: 200,
			},
			wantDataAttr: map[string]string{
				"url.domain":                "test.domain",
				"rpc.system.name":           "http",
				"rpc.response.status_code":  "OK",
				"rpc.method":                "my.service.Method",
				"url.template":              "/v1/test/{id}",
				"server.address":            "localhost",
				"server.port":               "8080",
				"http.response.status_code": "200",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			if tt.method != "" {
				ctx = callctx.WithTelemetryContext(ctx, "rpc_method", tt.method)
			}
			if tt.template != "" {
				ctx = callctx.WithTelemetryContext(ctx, "url_template", tt.template)
			}
			if tt.transportData != nil {
				ctx = InjectTransportTelemetry(ctx, tt.transportData)
			}

			reader := sdkmetric.NewManualReader()
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			settings := CallSettings{}
			if !tt.nilMetrics {
				rpcSys := "grpc"
				if tt.rpcSystem != "" {
					rpcSys = tt.rpcSystem
				}
				opts := []TelemetryOption{
					WithMeterProvider(provider),
					WithTelemetryAttributes(map[string]string{
						ClientArtifact: "test-artifact",
						ClientService:  "test-service",
						URLDomain:      "test.domain",
						RPCSystem:      rpcSys,
					}),
				}
				cm := NewClientMetrics(opts...)
				WithClientMetrics(cm).Resolve(&settings)
			}

			dur := time.Duration(tt.wantSum * float64(time.Second))
			recordMetric(ctx, settings, dur, tt.err)

			var rm metricdata.ResourceMetrics
			if err := reader.Collect(context.Background(), &rm); err != nil {
				t.Fatalf("failed to collect metrics: %v", err)
			}

			if tt.nilMetrics {
				if len(rm.ScopeMetrics) > 0 {
					t.Fatalf("expected 0 metrics recorded for nil clientMetrics")
				}
				return
			}

			if len(rm.ScopeMetrics) == 0 {
				t.Fatalf("expected at least 1 ScopeMetrics")
			}

			scopeMetric := rm.ScopeMetrics[0]
			if len(scopeMetric.Metrics) == 0 {
				t.Fatalf("expected at least 1 Metric recorded")
			}

			metric := scopeMetric.Metrics[0]
			if metric.Name != metricName {
				t.Errorf("expected metric.Name %q, got %q", metricName, metric.Name)
			}

			histo, ok := metric.Data.(metricdata.Histogram[float64])
			if !ok {
				t.Fatalf("expected metricdata.Histogram[float64], got %T", metric.Data)
			}

			if len(histo.DataPoints) == 0 {
				t.Fatalf("expected at least 1 DataPoint")
			}

			point := histo.DataPoints[0]

			if math.Abs(point.Sum-tt.wantSum) > 1e-6 {
				t.Errorf("expected float sum %f, got %f", tt.wantSum, point.Sum)
			}
			if point.Count != 1 {
				t.Errorf("expected count 1, got %d", point.Count)
			}

			wantScopeAttr := map[string]string{
				"gcp.client.service": "test-service",
			}
			gotScopeAttr := make(map[string]string)
			for _, a := range scopeMetric.Scope.Attributes.ToSlice() {
				gotScopeAttr[string(a.Key)] = a.Value.AsString()
			}

			if diff := cmp.Diff(wantScopeAttr, gotScopeAttr); diff != "" {
				t.Errorf("Scope attributes mismatch (-want +got):\n%s", diff)
			}

			gotDataAttr := make(map[string]string)
			for _, a := range point.Attributes.ToSlice() {
				gotDataAttr[string(a.Key)] = a.Value.String()
			}

			if diff := cmp.Diff(tt.wantDataAttr, gotDataAttr); diff != "" {
				t.Errorf("DataPoint attributes mismatch (-want +got):\n%s", diff)
			}
		})
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

func TestNewClientTracing(t *testing.T) {
	customProvider := tracenoop.NewTracerProvider()

	for _, tt := range []struct {
		name      string
		opts      []TracingOption
		wantAttr  map[string]string
		useCustom bool
	}{
		{
			name: "static attributes with custom provider",
			opts: []TracingOption{
				WithTracingAttributes(map[string]string{
					ClientArtifact: "test-lib",
					ClientVersion:  "v1.0.1",
					ClientService:  "myservice",
					RPCSystem:      "grpc",
					URLDomain:      "test.domain",
					"ignored.key":  "ignored",
				}),
			},
			useCustom: true,
			wantAttr: map[string]string{
				"url.domain":         "test.domain",
				"rpc.system.name":    "grpc",
				"gcp.client.service": "myservice",
			},
		},
		{
			name: "nil entries and multiple options override",
			opts: []TracingOption{
				nil,
				WithTracingAttributes(map[string]string{ClientArtifact: "first", URLDomain: "first.domain"}),
				WithTracingAttributes(map[string]string{ClientArtifact: "second", URLDomain: "second.domain"}),
				nil,
			},
			wantAttr: map[string]string{"url.domain": "second.domain"},
		},
		{
			name:     "no options",
			opts:     nil,
			wantAttr: map[string]string{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			opts := tt.opts
			if tt.useCustom {
				opts = append(opts, WithTracerProvider(customProvider))
			}
			ct := NewClientTracing(opts...)
			if ct == nil || ct.tracer() == nil {
				t.Fatalf("expected non-nil ClientTracing and tracer")
			}
			gotAttr := make(map[string]string)
			for _, a := range ct.attributes() {
				gotAttr[string(a.Key)] = a.Value.AsString()
			}
			if diff := cmp.Diff(tt.wantAttr, gotAttr); diff != "" {
				t.Errorf("Attributes mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewClientTracing_LifecycleAndFallback(t *testing.T) {
	for _, tt := range []struct {
		name       string
		opts       []TracingOption
		tracerName string
	}{
		{
			name: "unspecified provider delegates to otel.GetTracerProvider",
			opts: []TracingOption{
				WithTracingAttributes(map[string]string{ClientArtifact: "test-global-fallback", ClientVersion: "1.2.3"}),
			},
			tracerName: "test-global-fallback",
		},
		{
			name: "explicit nil provider falls back to otel.GetTracerProvider",
			opts: []TracingOption{
				WithTracerProvider(nil),
				WithTracingAttributes(map[string]string{ClientArtifact: "test-nil-provider-fallback", ClientVersion: "1.2.3"}),
			},
			tracerName: "test-nil-provider-fallback",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			globalSpy := &spyTracerProvider{}
			prev := otel.GetTracerProvider()
			otel.SetTracerProvider(globalSpy)
			t.Cleanup(func() { otel.SetTracerProvider(prev) })

			initialCalls := globalSpy.callCount

			ct := NewClientTracing(tt.opts...)
			if ct == nil {
				t.Fatalf("expected non-nil ClientTracing")
			}
			// Verify lazy initialization: Tracer() not called until first access
			if globalSpy.callCount-initialCalls != 0 {
				t.Fatalf("expected 0 calls before tracer access, got %d", globalSpy.callCount-initialCalls)
			}
			if ct.tracer() == nil || globalSpy.callCount-initialCalls != 1 || globalSpy.tracerName != tt.tracerName {
				t.Fatalf("expected 1 call with tracer %q, got %d calls with %q", tt.tracerName, globalSpy.callCount-initialCalls, globalSpy.tracerName)
			}
			// Cached on repeated access
			_ = ct.tracer()
			if globalSpy.callCount-initialCalls != 1 {
				t.Fatalf("expected cached tracer without new provider calls")
			}
		})
	}
}

type spyTracerProvider struct {
	tracenoop.TracerProvider
	mu         sync.Mutex
	callCount  int
	tracerName string
}

func (s *spyTracerProvider) Tracer(name string, options ...trace.TracerOption) trace.Tracer {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callCount++
	s.tracerName = name
	return s.TracerProvider.Tracer(name, options...)
}

func TestClientTracing_ConcurrentAndNil(t *testing.T) {
	t.Run("concurrent access", func(t *testing.T) {
		spy := &spyTracerProvider{}
		ct := NewClientTracing(WithTracerProvider(spy), WithTracingAttributes(map[string]string{
			ClientArtifact: "my-artifact", ClientVersion: "1.0", ClientService: "srv",
		}))
		var wg sync.WaitGroup
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if ct.tracer() == nil || len(ct.attributes()) != 1 {
					t.Errorf("unexpected tracer or attributes")
				}
			}()
		}
		wg.Wait()
		if spy.callCount != 1 {
			t.Errorf("expected exactly 1 Tracer call across goroutines, got %d", spy.callCount)
		}
	})

	t.Run("nil receiver safety", func(t *testing.T) {
		var ct *ClientTracing
		if ct.tracer() != nil || ct.attributes() != nil {
			t.Errorf("expected nil for nil receiver")
		}
		ct = &ClientTracing{}
		if ct.tracer() != nil || ct.attributes() != nil {
			t.Errorf("expected nil for empty struct")
		}
	})
}

func TestSanitizeURLTemplate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "path template without query or fragment",
			input: "v1/projects/{project}/databases/{database}/documents",
			want:  "v1/projects/{project}/databases/{database}/documents",
		},
		{
			name:  "path template with query parameters",
			input: "v1/projects/{project}/databases/{database}/documents?alt=json&key=secret",
			want:  "v1/projects/{project}/databases/{database}/documents",
		},
		{
			name:  "path template with URL fragment",
			input: "v1/projects/{project}/databases/{database}/documents#section-1",
			want:  "v1/projects/{project}/databases/{database}/documents",
		},
		{
			name:  "path template with both query parameters and fragment",
			input: "v1/projects/{project}/databases/{database}/documents?alt=json#section-1",
			want:  "v1/projects/{project}/databases/{database}/documents",
		},
		{
			name:  "fragment before query character",
			input: "v1/resource#fragment?not-query",
			want:  "v1/resource",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only query parameter",
			input: "?alt=json",
			want:  "",
		},
		{
			name:  "only fragment",
			input: "#frag",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeURLTemplate(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeURLTemplate(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveSpanName(t *testing.T) {
	tests := []struct {
		name     string
		ctxSetup func(context.Context) context.Context
		want     string
	}{
		{
			name: "REST HTTP method and URL template",
			ctxSetup: func(ctx context.Context) context.Context {
				ctx = callctx.WithTelemetryContext(ctx, "http_method", "GET")
				return callctx.WithTelemetryContext(ctx, "url_template", "v1/projects/{project}/databases/{database}")
			},
			want: "GET v1/projects/{project}/databases/{database}",
		},
		{
			name: "REST HTTP method and URL template with query params sanitized",
			ctxSetup: func(ctx context.Context) context.Context {
				ctx = callctx.WithTelemetryContext(ctx, "http_method", "POST")
				return callctx.WithTelemetryContext(ctx, "url_template", "v1/projects/{project}/locations?alt=json")
			},
			want: "POST v1/projects/{project}/locations",
		},
		{
			name: "REST takes precedence over gRPC method per SemConv",
			ctxSetup: func(ctx context.Context) context.Context {
				ctx = callctx.WithTelemetryContext(ctx, "http_method", "DELETE")
				ctx = callctx.WithTelemetryContext(ctx, "url_template", "v1/projects/{project}/instances/{instance}")
				return callctx.WithTelemetryContext(ctx, "rpc_method", "google.spanner.admin.instance.v1.InstanceAdmin/DeleteInstance")
			},
			want: "DELETE v1/projects/{project}/instances/{instance}",
		},
		{
			name: "HTTP method present but missing URL template falls back to gRPC",
			ctxSetup: func(ctx context.Context) context.Context {
				ctx = callctx.WithTelemetryContext(ctx, "http_method", "POST")
				return callctx.WithTelemetryContext(ctx, "rpc_method", "google.pubsub.v1.Publisher/Publish")
			},
			want: "google.pubsub.v1.Publisher/Publish",
		},
		{
			name: "URL template present but missing HTTP method falls back to gRPC",
			ctxSetup: func(ctx context.Context) context.Context {
				ctx = callctx.WithTelemetryContext(ctx, "url_template", "v1/projects/{project}/topics")
				return callctx.WithTelemetryContext(ctx, "rpc_method", "google.pubsub.v1.Publisher/Publish")
			},
			want: "google.pubsub.v1.Publisher/Publish",
		},
		{
			name: "gRPC method present without HTTP context",
			ctxSetup: func(ctx context.Context) context.Context {
				return callctx.WithTelemetryContext(ctx, "rpc_method", "google.pubsub.v1.Publisher/Publish")
			},
			want: "google.pubsub.v1.Publisher/Publish",
		},
		{
			name: "empty context falls back to default span name",
			ctxSetup: func(ctx context.Context) context.Context {
				return ctx
			},
			want: "gcp.client.request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.ctxSetup(context.Background())
			got := resolveSpanName(ctx)
			if got != tt.want {
				t.Errorf("resolveSpanName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEndSpan(t *testing.T) {
	stWithDetails, _ := status.New(codes.PermissionDenied, "disabled").WithDetails(&errdetails.ErrorInfo{
		Reason: "SERVICE_DISABLED", Domain: "googleapis.com", Metadata: map[string]string{
			"service": "speech.googleapis.com",
			"foo":     "bar",
		},
	})
	apiErr, _ := apierror.FromError(stWithDetails.Err())

	tests := []struct {
		name       string
		setupCtx   func() context.Context
		err        error
		wantStatus otelcodes.Code
		wantAttrs  map[string]any
	}{
		{
			name: "success without transport telemetry",
			setupCtx: func() context.Context {
				return context.Background()
			},
			err:        nil,
			wantStatus: otelcodes.Ok,
			wantAttrs: map[string]any{
				"rpc.response.status_code": "OK",
			},
		},
		{
			name: "success with transport telemetry ignored by T3 span",
			setupCtx: func() context.Context {
				td := &TransportTelemetryData{}
				td.SetServerAddress("speech.googleapis.com")
				td.SetServerPort(443)
				td.SetHTTPStatusCode(200)
				return InjectTransportTelemetry(context.Background(), td)
			},
			err:        nil,
			wantStatus: otelcodes.Ok,
			wantAttrs: map[string]any{
				"rpc.response.status_code": "OK",
			},
		},
		{
			name: "terminal gRPC failure",
			setupCtx: func() context.Context {
				return context.Background()
			},
			err:        status.Error(codes.PermissionDenied, "permission denied"),
			wantStatus: otelcodes.Error,
			wantAttrs: map[string]any{
				"error.type":               "PERMISSION_DENIED",
				"rpc.response.status_code": "PERMISSION_DENIED",
			},
		},
		{
			name: "context deadline exceeded error",
			setupCtx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 0)
				cancel()
				return ctx
			},
			err:        context.DeadlineExceeded,
			wantStatus: otelcodes.Error,
			wantAttrs: map[string]any{
				"error.type":               "CLIENT_TIMEOUT",
				"rpc.response.status_code": "DEADLINE_EXCEEDED",
			},
		},
		{
			name: "context canceled error",
			setupCtx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			err:        context.Canceled,
			wantStatus: otelcodes.Error,
			wantAttrs: map[string]any{
				"error.type":               "CLIENT_CANCELLED",
				"rpc.response.status_code": "CANCELED",
			},
		},
		{
			name: "APIError unpacking details and metadata",
			setupCtx: func() context.Context {
				return context.Background()
			},
			err:        apiErr,
			wantStatus: otelcodes.Error,
			wantAttrs: map[string]any{
				"error.type":                  "SERVICE_DISABLED",
				"rpc.response.status_code":    "PERMISSION_DENIED",
				"gcp.errors.domain":           "googleapis.com",
				"gcp.errors.metadata.foo":     "bar",
				"gcp.errors.metadata.service": "speech.googleapis.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exporter := tracetest.NewInMemoryExporter()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
			tracer := tp.Tracer("test")

			ctx := tt.setupCtx()
			ctx, span := tracer.Start(ctx, "test-span")

			errInfo := ExtractTelemetryErrorInfo(ctx, tt.err)
			endSpan(ctx, span, &errInfo, tt.err)

			spans := exporter.GetSpans()
			if len(spans) != 1 {
				t.Fatalf("len(spans) = %d, want 1", len(spans))
			}
			s := spans[0]
			if s.Status.Code != tt.wantStatus {
				t.Errorf("span.Status.Code = %v, want %v", s.Status.Code, tt.wantStatus)
			}
			if tt.wantStatus == otelcodes.Ok && s.Status.Description != "" {
				t.Errorf("span.Status.Description = %q, want empty for Ok status", s.Status.Description)
			}

			gotAttrs := make(map[string]any, len(s.Attributes))
			for _, a := range s.Attributes {
				switch a.Value.Type() {
				case attribute.STRING:
					gotAttrs[string(a.Key)] = a.Value.AsString()
				case attribute.INT64:
					gotAttrs[string(a.Key)] = int(a.Value.AsInt64())
				}
			}
			if diff := cmp.Diff(tt.wantAttrs, gotAttrs); diff != "" {
				t.Errorf("Attributes mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEndSpan_NilSpan(t *testing.T) {
	errInfo := ExtractTelemetryErrorInfo(context.Background(), nil)
	endSpan(context.Background(), nil, &errInfo, nil)
}

func TestStartSpan(t *testing.T) {
	t.Run("nil ClientTracing", func(t *testing.T) {
		ctx := context.Background()
		gotCtx, span := startSpan(ctx, nil)
		if span != nil {
			t.Errorf("expected nil span, got %v", span)
		}
		if gotCtx != ctx {
			t.Errorf("expected original context, got %v", gotCtx)
		}
	})

	t.Run("uninitialized ClientTracing", func(t *testing.T) {
		ctx := context.Background()
		gotCtx, span := startSpan(ctx, &ClientTracing{})
		if span != nil {
			t.Errorf("expected nil span, got %v", span)
		}
		if gotCtx != ctx {
			t.Errorf("expected original context, got %v", gotCtx)
		}
	})

	t.Run("gRPC with resource_name", func(t *testing.T) {
		exporter := tracetest.NewInMemoryExporter()
		tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
		ct := NewClientTracing(
			WithTracerProvider(tp),
			WithTracingAttributes(map[string]string{
				ClientService:  "speech",
				ClientArtifact: "cloud.google.com/go/speech",
				RPCSystem:      "grpc",
				URLDomain:      "speech.googleapis.com",
			}),
		)

		ctx := callctx.WithTelemetryContext(context.Background(), "rpc_method", "google.cloud.speech.v1.Speech/Recognize")
		ctx = callctx.WithTelemetryContext(ctx, "resource_name", "projects/p/locations/global")

		gotCtx, span := startSpan(ctx, ct)
		if span == nil {
			t.Fatal("expected non-nil span")
		}
		if trace.SpanFromContext(gotCtx) != span {
			t.Error("expected span injected into returned context")
		}
		span.End()

		spans := exporter.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("len(spans) = %d, want 1", len(spans))
		}
		s := spans[0]
		if s.Name != "google.cloud.speech.v1.Speech/Recognize" {
			t.Errorf("span.Name = %q, want google.cloud.speech.v1.Speech/Recognize", s.Name)
		}
		if s.SpanKind != trace.SpanKindClient {
			t.Errorf("span.SpanKind = %v, want %v", s.SpanKind, trace.SpanKindClient)
		}

		gotAttrs := make(map[string]string)
		for _, a := range s.Attributes {
			gotAttrs[string(a.Key)] = a.Value.AsString()
		}
		wantAttrs := map[string]string{
			"url.domain":                  "speech.googleapis.com",
			"rpc.system.name":             "grpc",
			"gcp.client.service":          "speech",
			"gcp.resource.destination.id": "projects/p/locations/global",
		}
		if diff := cmp.Diff(wantAttrs, gotAttrs); diff != "" {
			t.Errorf("Attributes mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("REST with sanitized url_template", func(t *testing.T) {
		exporter := tracetest.NewInMemoryExporter()
		tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
		ct := NewClientTracing(
			WithTracerProvider(tp),
			WithTracingAttributes(map[string]string{
				ClientService:  "storage",
				ClientArtifact: "cloud.google.com/go/storage",
				RPCSystem:      "http",
				URLDomain:      "storage.googleapis.com",
			}),
		)

		ctx := callctx.WithTelemetryContext(context.Background(), "http_method", "GET")
		ctx = callctx.WithTelemetryContext(ctx, "url_template", "/b/{bucket}/o?key=secret#frag")

		_, span := startSpan(ctx, ct)
		if span == nil {
			t.Fatal("expected non-nil span")
		}
		span.End()

		spans := exporter.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("len(spans) = %d, want 1", len(spans))
		}
		s := spans[0]
		if s.Name != "GET /b/{bucket}/o" {
			t.Errorf("span.Name = %q, want GET /b/{bucket}/o", s.Name)
		}

		gotAttrs := make(map[string]string)
		for _, a := range s.Attributes {
			gotAttrs[string(a.Key)] = a.Value.AsString()
		}
		wantAttrs := map[string]string{
			"url.domain":         "storage.googleapis.com",
			"rpc.system.name":    "http",
			"gcp.client.service": "storage",
			"url.template":       "/b/{bucket}/o",
		}
		if diff := cmp.Diff(wantAttrs, gotAttrs); diff != "" {
			t.Errorf("Attributes mismatch (-want +got):\n%s", diff)
		}
	})
}
