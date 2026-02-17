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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type metricsOpt struct {
	libraryName    string
	libraryVersion string
	scopeAttrs     map[string]string
	metricAttrs    map[string]string
}

// WithMetrics is a CallOption that initializes OpenTelemetry metrics for the client.
// It sets up the MeterProvider, retrieves the meter for the given library,
// creates the metric instrument (gcp.client.request.duration), and stores the
// attributes to be recorded with each metric emission.
// scopeAttrs are instrumentation scope attributes.
// metricAttrs are common per-metric attributes (supplied to record).
func WithMetrics(libraryName, libraryVersion string, scopeAttrs map[string]string, metricAttrs map[string]string) CallOption {
	return &metricsOpt{
		libraryName:    libraryName,
		libraryVersion: libraryVersion,
		scopeAttrs:     scopeAttrs,
		metricAttrs:    metricAttrs,
	}
}

func (m *metricsOpt) Resolve(s *CallSettings) {
	provider := otel.GetMeterProvider()

	var otelScopeAttrs []attribute.KeyValue
	for k, v := range m.scopeAttrs {
		otelScopeAttrs = append(otelScopeAttrs, attribute.String(k, v))
	}

	meter := provider.Meter(m.libraryName, metric.WithInstrumentationVersion(m.libraryVersion), metric.WithInstrumentationAttributes(otelScopeAttrs...))

	// Create the duration histogram. Note that we don't return the error directly
	// as CallOptions cannot return errors, and we do not log them. If creation
	// fails, we simply skip metrics generation.
	hist, err := meter.Float64Histogram("gcp.client.request.duration", metric.WithUnit("s"))
	if err != nil {
		return
	}

	var otelMetricAttrs []attribute.KeyValue
	for k, v := range m.metricAttrs {
		otelMetricAttrs = append(otelMetricAttrs, attribute.String(k, v))
	}

	s.MetricInstrument = hist
	s.TelemetryAttributes = append(s.TelemetryAttributes, otelMetricAttrs...)
}
