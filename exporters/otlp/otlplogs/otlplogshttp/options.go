/*
Copyright Agoda Services Co.,Ltd.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package otlplogshttp

import (
	"crypto/tls"
	"github.com/metoro-io/opentelemetry-logs-go/exporters/otlp/otlplogs/internal/otlpconfig"
	"github.com/metoro-io/opentelemetry-logs-go/exporters/otlp/otlplogs/internal/retry"
	"net/http"
	"time"
)

// Compression describes the compression used for payloads sent to the
// collector.
type Compression otlpconfig.Compression

const (
	// NoCompression tells the driver to send payloads without
	// compression.
	NoCompression = Compression(otlpconfig.NoCompression)
	// GzipCompression tells the driver to send payloads after
	// compressing them with gzip.
	GzipCompression = Compression(otlpconfig.GzipCompression)
)

// Option applies an option to the HTTP httpClient.
type Option interface {
	applyHTTPOption(otlpconfig.Config) otlpconfig.Config
}

func asHTTPOptions(opts []Option) []otlpconfig.HTTPOption {
	converted := make([]otlpconfig.HTTPOption, len(opts))
	for i, o := range opts {
		converted[i] = otlpconfig.NewHTTPOption(o.applyHTTPOption)
	}
	return converted
}

// RetryConfig defines configuration for retrying batches in case of export
// failure using an exponential backoff.
type RetryConfig retry.Config

type wrappedOption struct {
	otlpconfig.HTTPOption
}

func (w wrappedOption) applyHTTPOption(cfg otlpconfig.Config) otlpconfig.Config {
	return w.ApplyHTTPOption(cfg)
}

// WithEndpoint allows one to set the address of the collector
// endpoint that the driver will use to send logs. If
// unset, it will instead try to use
// the default endpoint (localhost:4318). Note that the endpoint
// must not contain any URL path.
func WithEndpoint(endpoint string) Option {
	return wrappedOption{otlpconfig.WithEndpoint(endpoint)}
}

// WithJsonProtocol will apply http/json protocol to Http client
func WithJsonProtocol() Option {
	return wrappedOption{otlpconfig.WithProtocol(otlpconfig.ExporterProtocolHttpJson)}
}

// WithProtobufProtocol will apply http/protobuf protocol to Http client
func WithProtobufProtocol() Option {
	return wrappedOption{otlpconfig.WithProtocol(otlpconfig.ExporterProtocolHttpProtobuf)}
}

// WithCompression tells the driver to compress the sent data.
func WithCompression(compression Compression) Option {
	return wrappedOption{otlpconfig.WithCompression(otlpconfig.Compression(compression))}
}

// WithHTTPClient sets the HTTP client to used by the exporter.
//
// This option will take precedence over [WithProxy], [WithTimeout],
// [WithTLSClientConfig] options as well as OTEL_EXPORTER_OTLP_CERTIFICATE,
// OTEL_EXPORTER_OTLP_TRACES_CERTIFICATE, OTEL_EXPORTER_OTLP_TIMEOUT,
// OTEL_EXPORTER_OTLP_TRACES_TIMEOUT environment variables.
//
// Timeout and all other fields of the passed [http.Client] are left intact.
//
// Be aware that passing an HTTP client with transport like
// [go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp.NewTransport] can
// cause the client to be instrumented twice and cause infinite recursion.
func WithHTTPClient(c *http.Client) Option {
	return wrappedOption{otlpconfig.WithHTTPClient(c)}
}

// WithURLPath allows one to override the default URL path used
// for sending logs. If unset, default ("/v1/logs") will be used.
func WithURLPath(urlPath string) Option {
	return wrappedOption{otlpconfig.WithURLPath(urlPath)}
}

// WithTLSClientConfig can be used to set up a custom TLS
// configuration for the httpClient used to send payloads to the
// collector. Use it if you want to use a custom certificate.
func WithTLSClientConfig(tlsCfg *tls.Config) Option {
	return wrappedOption{otlpconfig.WithTLSClientConfig(tlsCfg)}
}

// WithInsecure tells the driver to connect to the collector using the
// HTTP scheme, instead of HTTPS.
func WithInsecure() Option {
	return wrappedOption{otlpconfig.WithInsecure()}
}

// WithHeaders allows one to tell the driver to send additional HTTP
// headers with the payloads. Specifying headers like Content-Length,
// Content-Encoding and Content-Type may result in a broken driver.
func WithHeaders(headers map[string]string) Option {
	return wrappedOption{otlpconfig.WithHeaders(headers)}
}

// WithTimeout tells the driver the max waiting time for the backend to process
// each logs batch.  If unset, the default will be 10 seconds.
func WithTimeout(duration time.Duration) Option {
	return wrappedOption{otlpconfig.WithTimeout(duration)}
}

// WithRetry configures the retry policy for transient errors that may occurs
// when exporting logs. An exponential back-off algorithm is used to ensure
// endpoints are not overwhelmed with retries. If unset, the default retry
// policy will retry after 5 seconds and increase exponentially after each
// error for a total of 1 minute.
func WithRetry(rc RetryConfig) Option {
	return wrappedOption{otlpconfig.WithRetry(retry.Config(rc))}
}
