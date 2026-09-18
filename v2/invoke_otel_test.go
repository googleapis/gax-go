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
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/gax-go/v2/callctx"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type testRetryer struct {
	count int
}

func (r *testRetryer) Retry(err error) (time.Duration, bool) {
	if r.count == 0 {
		r.count++
		return 50 * time.Millisecond, true
	}
	return 0, false
}

func TestInvokeWithMetrics(t *testing.T) {
	t.Setenv("GOOGLE_SDK_GO_EXPERIMENTAL_METRICS", "true")
	TestOnlyResetIsFeatureEnabled()
	defer TestOnlyResetIsFeatureEnabled()

	tests := []struct {
		name         string
		setupCtx     func() (context.Context, context.CancelFunc)
		callFunc     func(context.Context, CallSettings) error
		callOpts     []CallOption
		wantDataAttr map[string]string
		wantErr      bool
		minSum       float64
	}{
		{
			name: "success",
			setupCtx: func() (context.Context, context.CancelFunc) {
				return context.Background(), func() {}
			},
			callFunc: func(ctx context.Context, settings CallSettings) error {
				time.Sleep(10 * time.Millisecond)
				return nil
			},
			wantDataAttr: map[string]string{
				"url.domain":               "test.domain",
				"rpc.system.name":          "grpc",
				"rpc.response.status_code": "OK",
			},
			wantErr: false,
			minSum:  0.01,
		},
		{
			name: "retry_with_backoff",
			setupCtx: func() (context.Context, context.CancelFunc) {
				return context.Background(), func() {}
			},
			callOpts: []CallOption{
				WithRetry(func() Retryer { return &testRetryer{} }),
			},
			callFunc: func(ctx context.Context, settings CallSettings) error {
				return status.Error(codes.DeadlineExceeded, "deadline exceeded") // Trigger retry, eventually failing after 1 retry
			},
			wantDataAttr: map[string]string{
				"url.domain":               "test.domain",
				"rpc.system.name":          "grpc",
				"error.type":               "DEADLINE_EXCEEDED",
				"rpc.response.status_code": "DEADLINE_EXCEEDED",
			},
			wantErr: true,
			minSum:  0.05, // The retryer sleeps for 50ms, so duration must be at least 0.05s
		},
		{
			name: "error_attributes",
			setupCtx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 1*time.Millisecond)
			},
			callFunc: func(ctx context.Context, settings CallSettings) error {
				<-ctx.Done()
				return ctx.Err()
			},
			wantDataAttr: map[string]string{
				"url.domain":               "test.domain",
				"rpc.system.name":          "grpc",
				"error.type":               "CLIENT_TIMEOUT",
				"rpc.response.status_code": "DEADLINE_EXCEEDED",
			},
			wantErr: true,
		},
		{
			name: "metadata_attributes",
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx := callctx.WithTelemetryContext(context.Background(), "rpc_method", "my_method", "url_template", "/v1/foo")
				return ctx, func() {}
			},
			callFunc: func(ctx context.Context, settings CallSettings) error {
				return nil
			},
			wantDataAttr: map[string]string{
				"url.domain":               "test.domain",
				"rpc.system.name":          "grpc",
				"rpc.response.status_code": "OK",
				"rpc.method":               "my_method",
				"url.template":             "/v1/foo",
			},
			wantErr: false,
		},
		{
			name: "transport_telemetry_data",
			setupCtx: func() (context.Context, context.CancelFunc) {
				return context.Background(), func() {}
			},
			callFunc: func(ctx context.Context, settings CallSettings) error {
				if td := ExtractTransportTelemetry(ctx); td != nil {
					td.SetServerAddress("127.0.0.1")
					td.SetServerPort(8080)
					td.SetHTTPStatusCode(200)
				}
				return nil
			},
			wantDataAttr: map[string]string{
				"url.domain":                "test.domain",
				"rpc.system.name":           "grpc",
				"rpc.response.status_code":  "OK",
				"server.address":            "127.0.0.1",
				"server.port":               "8080",
				"http.response.status_code": "200",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := tt.setupCtx()
			defer cancel()

			reader := metric.NewManualReader()
			provider := metric.NewMeterProvider(metric.WithReader(reader))

			opts := []TelemetryOption{
				WithMeterProvider(provider),
				WithTelemetryAttributes(map[string]string{
					URLDomain: "test.domain",
					RPCSystem: "grpc",
				}),
			}
			cm := NewClientMetrics(opts...)

			callOpts := []CallOption{WithClientMetrics(cm)}
			if tt.callOpts != nil {
				callOpts = append(callOpts, tt.callOpts...)
			}

			err := Invoke(ctx, tt.callFunc, callOpts...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Invoke() error = %v, wantErr %v", err, tt.wantErr)
			}

			var rm metricdata.ResourceMetrics
			if err := reader.Collect(context.Background(), &rm); err != nil {
				t.Fatalf("failed to collect metrics: %v", err)
			}

			if len(rm.ScopeMetrics) == 0 {
				t.Fatalf("expected at least 1 ScopeMetrics")
			}

			scopeMetric := rm.ScopeMetrics[0]
			if len(scopeMetric.Metrics) == 0 {
				t.Fatalf("expected at least 1 Metric recorded")
			}

			m := scopeMetric.Metrics[0]
			if m.Name != metricName {
				t.Errorf("expected metric.Name %q, got %q", metricName, m.Name)
			}

			histo, ok := m.Data.(metricdata.Histogram[float64])
			if !ok {
				t.Fatalf("expected metricdata.Histogram[float64], got %T", m.Data)
			}

			if len(histo.DataPoints) == 0 {
				t.Fatalf("expected at least 1 DataPoint")
			}

			point := histo.DataPoints[0]
			if point.Count != 1 {
				t.Errorf("expected count 1, got %d", point.Count)
			}
			if point.Sum < tt.minSum {
				t.Errorf("expected sum >= %f, got %f", tt.minSum, point.Sum)
			}

			gotDataAttr := make(map[string]string)
			for _, a := range point.Attributes.ToSlice() {
				gotDataAttr[string(a.Key)] = fmt.Sprintf("%v", a.Value.AsInterface())
			}

			if diff := cmp.Diff(tt.wantDataAttr, gotDataAttr); diff != "" {
				t.Errorf("DataPoint attributes mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInvokeWithTracing(t *testing.T) {
	t.Setenv("GOOGLE_SDK_GO_EXPERIMENTAL_TRACING", "true")
	TestOnlyResetIsFeatureEnabled()
	defer TestOnlyResetIsFeatureEnabled()

	tests := []struct {
		name              string
		setupCtx          func() (context.Context, context.CancelFunc)
		callFunc          func(context.Context, CallSettings) error
		callOpts          []CallOption
		wantName          string
		wantStatus        otelcodes.Code
		wantDataAttr      map[string]string
		wantExcludedAttrs []string
		wantErr           bool
	}{
		{
			name: "success_grpc",
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx := callctx.WithTelemetryContext(context.Background(), "rpc_method", "my.service.Method")
				ctx = callctx.WithTelemetryContext(ctx, "resource_name", "projects/p/locations/global")
				return ctx, func() {}
			},
			callFunc: func(ctx context.Context, settings CallSettings) error {
				span := trace.SpanFromContext(ctx)
				if !span.SpanContext().IsValid() {
					t.Error("expected valid span context in call context")
				}
				return nil
			},
			wantName:   "my.service.Method",
			wantStatus: otelcodes.Ok,
			wantDataAttr: map[string]string{
				"url.domain":                  "test.domain",
				"rpc.system.name":             "grpc",
				"rpc.response.status_code":    "OK",
				"gcp.resource.destination.id": "projects/p/locations/global",
			},
			wantErr: false,
		},
		{
			name: "success_rest",
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx := callctx.WithTelemetryContext(context.Background(), "http_method", "GET")
				ctx = callctx.WithTelemetryContext(ctx, "url_template", "/v1/projects/{project}/locations/{location}?key=secret")
				return ctx, func() {}
			},
			callFunc: func(ctx context.Context, settings CallSettings) error {
				return nil
			},
			wantName:   "GET /v1/projects/{project}/locations/{location}",
			wantStatus: otelcodes.Ok,
			wantDataAttr: map[string]string{
				"url.domain":               "test.domain",
				"rpc.system.name":          "grpc",
				"url.template":             "/v1/projects/{project}/locations/{location}",
				"rpc.response.status_code": "OK",
			},
			wantErr: false,
		},
		{
			name: "retry_with_backoff",
			setupCtx: func() (context.Context, context.CancelFunc) {
				return context.Background(), func() {}
			},
			callOpts: []CallOption{
				WithRetry(func() Retryer { return &testRetryer{} }),
			},
			callFunc: func(ctx context.Context, settings CallSettings) error {
				return status.Error(codes.DeadlineExceeded, "deadline exceeded")
			},
			wantName:   "gcp.client.request",
			wantStatus: otelcodes.Error,
			wantDataAttr: map[string]string{
				"url.domain":               "test.domain",
				"rpc.system.name":          "grpc",
				"error.type":               "DEADLINE_EXCEEDED",
				"rpc.response.status_code": "DEADLINE_EXCEEDED",
			},
			wantErr: true,
		},
		{
			name: "transport_telemetry_ignored",
			setupCtx: func() (context.Context, context.CancelFunc) {
				td := &TransportTelemetryData{}
				td.SetServerAddress("127.0.0.1")
				td.SetServerPort(8080)
				td.SetHTTPStatusCode(200)
				return InjectTransportTelemetry(context.Background(), td), func() {}
			},
			callFunc: func(ctx context.Context, settings CallSettings) error {
				return nil
			},
			wantName:   "gcp.client.request",
			wantStatus: otelcodes.Ok,
			wantDataAttr: map[string]string{
				"url.domain":               "test.domain",
				"rpc.system.name":          "grpc",
				"rpc.response.status_code": "OK",
			},
			wantExcludedAttrs: []string{"server.address", "server.port", "http.response.status_code"},
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := tt.setupCtx()
			defer cancel()

			exporter := tracetest.NewInMemoryExporter()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))

			ct := NewClientTracing(
				WithTracerProvider(tp),
				WithTracingAttributes(map[string]string{
					URLDomain: "test.domain",
					RPCSystem: "grpc",
				}),
			)

			callOpts := []CallOption{WithClientTracing(ct)}
			if tt.callOpts != nil {
				callOpts = append(callOpts, tt.callOpts...)
			}

			err := Invoke(ctx, tt.callFunc, callOpts...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Invoke() error = %v, wantErr %v", err, tt.wantErr)
			}

			spans := exporter.GetSpans()
			if len(spans) != 1 {
				t.Fatalf("expected 1 span, got %d", len(spans))
			}
			span := spans[0]

			if span.Name != tt.wantName {
				t.Errorf("span.Name = %q, want %q", span.Name, tt.wantName)
			}
			if span.SpanKind != trace.SpanKindClient {
				t.Errorf("span.SpanKind = %v, want %v", span.SpanKind, trace.SpanKindClient)
			}
			if span.Status.Code != tt.wantStatus {
				t.Errorf("span.Status.Code = %v, want %v", span.Status.Code, tt.wantStatus)
			}
			if tt.wantStatus == otelcodes.Ok && span.Status.Description != "" {
				t.Errorf("span.Status.Description = %q, want empty on Ok", span.Status.Description)
			}

			gotAttrs := make(map[string]string)
			for _, a := range span.Attributes {
				gotAttrs[string(a.Key)] = fmt.Sprintf("%v", a.Value.AsInterface())
			}
			for k, wantVal := range tt.wantDataAttr {
				if gotVal, ok := gotAttrs[k]; !ok || gotVal != wantVal {
					t.Errorf("attr %q = %q, want %q", k, gotVal, wantVal)
				}
			}
			for _, excl := range tt.wantExcludedAttrs {
				if _, ok := gotAttrs[excl]; ok {
					t.Errorf("attr %q should not be present on client span", excl)
				}
			}
		})
	}
}

func TestInvokeWithTracing_Disabled(t *testing.T) {
	TestOnlyResetIsFeatureEnabled()
	defer TestOnlyResetIsFeatureEnabled()

	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	ct := NewClientTracing(WithTracerProvider(tp))

	err := Invoke(context.Background(), func(ctx context.Context, s CallSettings) error { return nil }, WithClientTracing(ct))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exporter.GetSpans()) != 0 {
		t.Errorf("expected 0 spans when feature flag disabled, got %d", len(exporter.GetSpans()))
	}
}
