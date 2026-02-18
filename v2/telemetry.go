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

// ClientMetrics holds the initialized OpenTelemetry metrics state for the client.
// It is opaque to avoid leaking OTel types to generated clients.
type ClientMetrics struct {
	instrument metric.Float64Histogram
	attributes []attribute.KeyValue
}

// NewClientMetrics initializes the OpenTelemetry Meter and Histogram for the client.
// This should be called once per client instance.
func NewClientMetrics(provider metric.MeterProvider, libraryName, libraryVersion string, sharedAttrs map[string]string) (*ClientMetrics, error) {
	if provider == nil {
		provider = otel.GetMeterProvider()
	}

	meter := provider.Meter(libraryName, metric.WithInstrumentationVersion(libraryVersion))

	hist, err := meter.Float64Histogram("gcp.client.request.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}

	var otelMetricAttrs []attribute.KeyValue
	for k, v := range sharedAttrs {
		otelMetricAttrs = append(otelMetricAttrs, attribute.String(k, v))
	}

	return &ClientMetrics{
		instrument: hist,
		attributes: otelMetricAttrs,
	}, nil
}

type metricsOpt struct {
	cm *ClientMetrics
}

// WithClientMetrics is a CallOption that configures OpenTelemetry metrics for the call.
// It accepts a pre-initialized ClientMetrics struct.
func WithClientMetrics(cm *ClientMetrics) CallOption {
	return &metricsOpt{cm: cm}
}

func (m *metricsOpt) Resolve(s *CallSettings) {
	s.ClientMetrics = m.cm
}
