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
	"errors"
	"testing"

	"go.opentelemetry.io/otel"
	metricapi "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestWithMetrics_Resolve(t *testing.T) {
	provider := metric.NewMeterProvider()
	otel.SetMeterProvider(provider)

	libraryName := "test-library"
	libraryVersion := "1.2.3"
	scopeAttrs := map[string]string{
		"s_key1": "s_val1",
		"s_key2": "s_val2",
	}
	metricAttrs := map[string]string{
		"m_key1": "m_val1",
		"m_key2": "m_val2",
	}

	opt := WithMetrics(libraryName, libraryVersion, scopeAttrs, metricAttrs)

	var settings CallSettings
	opt.Resolve(&settings)

	if settings.MetricInstrument == nil {
		t.Error("expected MetricInstrument to be initialized, got nil")
	}

	if len(settings.TelemetryAttributes) != len(metricAttrs) {
		t.Errorf("expected %d TelemetryAttributes, got %d", len(metricAttrs), len(settings.TelemetryAttributes))
	}

	attrMap := make(map[string]string)
	for _, attr := range settings.TelemetryAttributes {
		attrMap[string(attr.Key)] = attr.Value.AsString()
	}

	for k, v := range metricAttrs {
		if attrMap[k] != v {
			t.Errorf("expected attribute %s=%s, got %s", k, v, attrMap[k])
		}
	}
}

func TestWithMetrics_Functionality(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)

	libraryName := "test-library"
	libraryVersion := "1.2.3"
	scopeAttrs := map[string]string{
		"s_key1": "s_val1",
		"s_key2": "s_val2",
	}
	metricAttrs := map[string]string{
		"m_key1": "m_val1",
		"m_key2": "m_val2",
	}

	opt := WithMetrics(libraryName, libraryVersion, scopeAttrs, metricAttrs)

	var settings CallSettings
	opt.Resolve(&settings)

	if settings.MetricInstrument == nil {
		t.Fatal("expected MetricInstrument to be initialized, got nil")
	}

	// Record a value to ensure the metric is functional
	ctx := context.Background()
	settings.MetricInstrument.Record(ctx, 42.0, metricapi.WithAttributes(settings.TelemetryAttributes...))

	rm := &metricdata.ResourceMetrics{}
	err := reader.Collect(ctx, rm)
	if err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	if len(rm.ScopeMetrics) != 1 {
		t.Fatalf("expected 1 ScopeMetrics, got %d", len(rm.ScopeMetrics))
	}

	sm := rm.ScopeMetrics[0]
	if sm.Scope.Name != libraryName {
		t.Errorf("expected LibraryName %s, got %s", libraryName, sm.Scope.Name)
	}
	if sm.Scope.Version != libraryVersion {
		t.Errorf("expected version %s, got %s", libraryVersion, sm.Scope.Version)
	}

	// Verify the instrumentation attributes
	smAttrs := make(map[string]string)
	for _, kv := range sm.Scope.Attributes.ToSlice() {
		smAttrs[string(kv.Key)] = kv.Value.AsString()
	}
	for k, v := range scopeAttrs {
		if smAttrs[k] != v {
			t.Errorf("expected scope attribute %s=%s, got %s", k, v, smAttrs[k])
		}
	}

	if len(sm.Metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(sm.Metrics))
	}

	m := sm.Metrics[0]
	if m.Name != "gcp.client.request.duration" {
		t.Errorf("expected metric name gcp.client.request.duration, got %s", m.Name)
	}

	// Verify the recorded value and attributes
	hist, ok := m.Data.(metricdata.Histogram[float64])
	if !ok {
		t.Fatalf("expected Histogram[float64], got %T", m.Data)
	}
	if len(hist.DataPoints) != 1 {
		t.Fatalf("expected 1 datapoint, got %d", len(hist.DataPoints))
	}

	dp := hist.DataPoints[0]
	if dp.Count != 1 {
		t.Errorf("expected count 1, got %d", dp.Count)
	}
	if dp.Sum != 42.0 {
		t.Errorf("expected sum 42.0, got %f", dp.Sum)
	}

	// TelemetryAttributes are recorded via metricapi.WithAttributes
	dpAttrs := make(map[string]string)
	for _, kv := range dp.Attributes.ToSlice() {
		dpAttrs[string(kv.Key)] = kv.Value.AsString()
	}
	for k, v := range metricAttrs {
		if dpAttrs[k] != v {
			t.Errorf("expected datapoint attribute %s=%s, got %s", k, v, dpAttrs[k])
		}
	}
}

type mockMeterProvider struct {
	metricapi.MeterProvider
}

func (m *mockMeterProvider) Meter(name string, opts ...metricapi.MeterOption) metricapi.Meter {
	return &mockMeter{}
}

type mockMeter struct {
	metricapi.Meter
}

func (m *mockMeter) Float64Histogram(name string, options ...metricapi.Float64HistogramOption) (metricapi.Float64Histogram, error) {
	return nil, errors.New("mock error")
}

func TestWithMetrics_HistogramCreationFailure(t *testing.T) {
	provider := &mockMeterProvider{}
	otel.SetMeterProvider(provider)

	libraryName := "test-library"
	libraryVersion := "1.2.3"

	// Calling Resolve will trigger the mockMeter to return an error when creating the histogram
	opt := WithMetrics(libraryName, libraryVersion, nil, nil)
	var settings CallSettings
	
	// Resolve should not panic and should leave MetricInstrument as nil
	opt.Resolve(&settings)

	if settings.MetricInstrument != nil {
		t.Errorf("expected MetricInstrument to be nil on creation failure, got %v", settings.MetricInstrument)
	}
}
