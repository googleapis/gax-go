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

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// TransportTelemetryData contains mutable telemetry information that the transport
// layer (e.g. gRPC or HTTP) populates during an RPC. This allows gax.Invoke to
// correctly emit metric data without directly importing those transport layers.
// This is an EXPERIMENTAL struct and should not be used by external consumers.
type TransportTelemetryData struct {
	serverAddress      string
	serverPort         int
	responseStatusCode int
}

// SetServerAddress sets the server address.
func (d *TransportTelemetryData) SetServerAddress(addr string) { d.serverAddress = addr }

// ServerAddress returns the server address.
func (d *TransportTelemetryData) ServerAddress() string { return d.serverAddress }

// SetServerPort sets the server port.
func (d *TransportTelemetryData) SetServerPort(port int) { d.serverPort = port }

// ServerPort returns the server port.
func (d *TransportTelemetryData) ServerPort() int { return d.serverPort }

// SetResponseStatusCode sets the response status code.
func (d *TransportTelemetryData) SetResponseStatusCode(code int) { d.responseStatusCode = code }

// ResponseStatusCode returns the response status code.
func (d *TransportTelemetryData) ResponseStatusCode() int { return d.responseStatusCode }

// transportTelemetryKey is the private context key used to inject TransportTelemetryData
type transportTelemetryKey struct{}

// InjectTransportTelemetry injects a mutable TransportTelemetryData pointer into the context.
// Experimental: This function is subject to breaking changes.
func InjectTransportTelemetry(ctx context.Context, data *TransportTelemetryData) context.Context {
	return context.WithValue(ctx, transportTelemetryKey{}, data)
}

// ExtractTransportTelemetry retrieves a mutable TransportTelemetryData pointer from the context.
// It returns nil if the data is not present.
// Experimental: This function is subject to breaking changes.
func ExtractTransportTelemetry(ctx context.Context) *TransportTelemetryData {
	data, _ := ctx.Value(transportTelemetryKey{}).(*TransportTelemetryData)
	return data
}

const (
	metricName        = "gcp.client.request.duration"
	metricDescription = "Duration of the request to the Google Cloud API"

	// Constants for ClientMetrics configuration map keys.
	// These are used by generated clients to pass attributes to the ClientMetrics option.
	// Because they are used in generated code, these values must not be changed.

	// ClientService is the Google Cloud API service name. E.g. "storage".
	ClientService = "client_service"
	// ClientVersion is the version of the client. E.g. "1.43.0".
	ClientVersion = "client_version"
	// ClientArtifact is the library name. E.g. "cloud.google.com/go/storage".
	ClientArtifact = "client_artifact"
	// RPCSystem is the RPC system type. E.g. "grpc" or "http".
	RPCSystem = "rpc_system"
	// URLDomain is the nominal service domain. E.g. "storage.googleapis.com".
	URLDomain = "url_domain"

	// Constants for telemetry attribute keys.
	keyGCPClientService = "gcp.client.service"
	keyRPCSystemName    = "rpc.system.name"
	keyURLDomain        = "url.domain"

	// SchemaURL specifies the OpenTelemetry schema version.
	schemaURL = "https://opentelemetry.io/schemas/1.39.0"
)

// Default bucket boundaries for the duration metric in seconds.
// An exponential-ish distribution.
var defaultHistogramBoundaries = []float64{
	0.0, 0.0001, 0.0005, 0.0010, 0.005, 0.010, 0.050, 0.100, 0.5, 1.0, 5.0, 10.0, 60.0, 300.0, 900.0, 3600.0,
}

// ClientMetrics contains the pre-allocated OpenTelemetry instruments and attributes
// for a specific generated Google Cloud client library.
// There should be exactly one ClientMetrics instance instantiated per generated client.
type ClientMetrics struct {
	duration metric.Float64Histogram
	attr     []attribute.KeyValue
}

type clientMetricsOptions struct {
	provider                 metric.MeterProvider
	attributes               map[string]string
	explicitBucketBoundaries []float64
}

// ClientMetricsOption is an option to configure a ClientMetrics instance.
// ClientMetricsOption works by modifying relevant fields of clientMetricsOptions.
type ClientMetricsOption interface {
	// Resolve applies the option by modifying opts.
	Resolve(opts *clientMetricsOptions)
}

type providerOpt struct {
	p metric.MeterProvider
}

func (p providerOpt) Resolve(opts *clientMetricsOptions) {
	opts.provider = p.p
}

// WithMeterProvider specifies the metric.MeterProvider to use for instruments.
func WithMeterProvider(p metric.MeterProvider) ClientMetricsOption {
	return &providerOpt{p: p}
}

type attrOpt struct {
	attrs map[string]string
}

func (a attrOpt) Resolve(opts *clientMetricsOptions) {
	opts.attributes = a.attrs
}

// WithClientMetricsAttributes specifies the static attributes attachments.
func WithClientMetricsAttributes(attr map[string]string) ClientMetricsOption {
	return &attrOpt{attrs: attr}
}

type boundariesOpt struct {
	boundaries []float64
}

func (b boundariesOpt) Resolve(opts *clientMetricsOptions) {
	opts.explicitBucketBoundaries = b.boundaries
}

// WithExplicitBucketBoundaries overrides the default histogram bucket boundaries.
func WithExplicitBucketBoundaries(boundaries []float64) ClientMetricsOption {
	return &boundariesOpt{boundaries: boundaries}
}

// NewClientMetrics initializes and returns a new ClientMetrics instance.
// It is intended to be called once per generated client during initialization.
func NewClientMetrics(opts ...ClientMetricsOption) *ClientMetrics {
	var config clientMetricsOptions
	for _, opt := range opts {
		opt.Resolve(&config)
	}

	provider := config.provider
	if provider == nil {
		provider = otel.GetMeterProvider()
	}

	var meterAttrs []attribute.KeyValue
	if val, ok := config.attributes[ClientService]; ok {
		meterAttrs = append(meterAttrs, attribute.KeyValue{Key: attribute.Key(keyGCPClientService), Value: attribute.StringValue(val)})
	}

	meterOpts := []metric.MeterOption{
		metric.WithInstrumentationVersion(config.attributes[ClientVersion]),
		metric.WithSchemaURL(schemaURL),
	}
	if len(meterAttrs) > 0 {
		meterOpts = append(meterOpts, metric.WithInstrumentationAttributes(meterAttrs...))
	}

	meter := provider.Meter(config.attributes[ClientArtifact], meterOpts...)

	boundaries := config.explicitBucketBoundaries
	if len(boundaries) == 0 {
		boundaries = defaultHistogramBoundaries
	}

	duration, _ := meter.Float64Histogram(
		metricName,
		metric.WithDescription(metricDescription),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(boundaries...),
	)

	var attr []attribute.KeyValue
	if val, ok := config.attributes[URLDomain]; ok {
		attr = append(attr, attribute.KeyValue{Key: attribute.Key(keyURLDomain), Value: attribute.StringValue(val)})
	}
	if val, ok := config.attributes[RPCSystem]; ok {
		attr = append(attr, attribute.KeyValue{Key: attribute.Key(keyRPCSystemName), Value: attribute.StringValue(val)})
	}

	return &ClientMetrics{
		duration: duration,
		attr:     attr,
	}
}
