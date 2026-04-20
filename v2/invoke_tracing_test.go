package gax

import (
	"context"
	"testing"

	"github.com/googleapis/gax-go/v2/callctx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	grpccodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestInvokeClientRequestSpan(t *testing.T) {
	t.Setenv("GOOGLE_SDK_GO_EXPERIMENTAL_TRACING", "true")
	TestOnlyResetIsFeatureEnabled()
	defer TestOnlyResetIsFeatureEnabled()

	tests := []struct {
		name         string
		setupCtx     func() context.Context
		callFunc     func(context.Context, CallSettings) error
		wantSpanName string
		wantAttrs    map[string]string
		wantStatus   codes.Code
		wantErr      bool
	}{
		{
			name: "success_with_client_span_name",
			setupCtx: func() context.Context {
				return callctx.WithTelemetryContext(context.Background(), "client_span_name", "MyClient.MyMethod")
			},
			callFunc: func(ctx context.Context, settings CallSettings) error {
				return nil
			},
			wantSpanName: "MyClient.MyMethod",
			wantAttrs:    map[string]string{},
			wantStatus:   codes.Ok,
			wantErr:      false,
		},
		{
			name: "success_with_fallback_rpc_method",
			setupCtx: func() context.Context {
				return callctx.WithTelemetryContext(context.Background(), "rpc_method", "my.pkg.Service/Method")
			},
			callFunc: func(ctx context.Context, settings CallSettings) error {
				return nil
			},
			wantSpanName: "my.pkg.Service/Method",
			wantAttrs: map[string]string{
				"rpc.method": "my.pkg.Service/Method",
			},
			wantStatus: codes.Ok,
			wantErr:    false,
		},
		{
			name: "failure_records_error_and_attributes",
			setupCtx: func() context.Context {
				return callctx.WithTelemetryContext(context.Background(), "client_span_name", "MyMethod")
			},
			callFunc: func(ctx context.Context, settings CallSettings) error {
				return status.Error(grpccodes.Unavailable, "service unavailable")
			},
			wantSpanName: "MyMethod",
			wantAttrs: map[string]string{
				"error.type":               "UNAVAILABLE",
				"rpc.response.status_code": "UNAVAILABLE",
				"status.message":           "service unavailable",
			},
			wantStatus: codes.Error,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()

			sr := tracetest.NewSpanRecorder()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

			oldProvider := otel.GetTracerProvider()
			otel.SetTracerProvider(tp)
			defer otel.SetTracerProvider(oldProvider)

			err := Invoke(ctx, tt.callFunc)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Invoke() error = %v, wantErr %v", err, tt.wantErr)
			}

			spans := sr.Ended()
			if len(spans) == 0 {
				t.Fatalf("expected at least 1 span recorded")
			}

			span := spans[0]
			if span.Name() != tt.wantSpanName {
				t.Errorf("expected span name %q, got %q", tt.wantSpanName, span.Name())
			}
			if span.Status().Code != tt.wantStatus {
				t.Errorf("expected span status %v, got %v", tt.wantStatus, span.Status().Code)
			}

			gotAttrs := make(map[string]string)
			for _, a := range span.Attributes() {
				gotAttrs[string(a.Key)] = a.Value.AsString()
			}

			for k, v := range tt.wantAttrs {
				if gotAttrs[k] != v {
					t.Errorf("expected attribute %q=%q, got %q", k, v, gotAttrs[k])
				}
			}
		})
	}
}
