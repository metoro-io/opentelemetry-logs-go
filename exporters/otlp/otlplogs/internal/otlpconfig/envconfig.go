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

package otlpconfig

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/metoro-io/opentelemetry-logs-go/exporters/otlp/otlplogs/internal/envconfig"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

// DefaultEnvOptionsReader is the default environments reader.
var DefaultEnvOptionsReader = envconfig.EnvOptionsReader{
	GetEnv:    os.Getenv,
	ReadFile:  os.ReadFile,
	Namespace: "OTEL_EXPORTER_OTLP",
}

func stringToProtocol(u string) Protocol {
	switch strings.ToLower(u) {
	case string(ExporterProtocolGrpc):
		return ExporterProtocolGrpc
	case string(ExporterProtocolHttpProtobuf):
		return ExporterProtocolHttpProtobuf
	case string(ExporterProtocolHttpJson):
		return ExporterProtocolHttpJson
	default:
		return ExporterProtocolHttpProtobuf
	}
}

// ApplyEnvProtocol Apply Protocol from environment to provided default value
// This function is subject to change or removal in future versions.
func ApplyEnvProtocol(protocol Protocol) Protocol {
	DefaultEnvOptionsReader.Apply(
		envconfig.WithString("PROTOCOL", func(s string) {
			protocol = stringToProtocol(s)
		}),
		envconfig.WithString("LOGS_PROTOCOL", func(s string) {
			protocol = stringToProtocol(s)
		}),
	)

	return protocol
}

// ApplyGRPCEnvConfigs applies the env configurations for gRPC.
func ApplyGRPCEnvConfigs(cfg Config) Config {
	opts := getOptionsFromEnv()
	for _, opt := range opts {
		cfg = opt.ApplyGRPCOption(cfg)
	}
	return cfg
}

// ApplyHTTPEnvConfigs applies the env configurations for HTTP.
func ApplyHTTPEnvConfigs(cfg Config) Config {
	opts := getOptionsFromEnv()
	for _, opt := range opts {
		cfg = opt.ApplyHTTPOption(cfg)
	}
	return cfg
}

func getOptionsFromEnv() []GenericOption {
	opts := []GenericOption{}

	tlsConf := &tls.Config{}
	DefaultEnvOptionsReader.Apply(
		envconfig.WithURL("ENDPOINT", func(u *url.URL) {
			opts = append(opts, withEndpointScheme(u))
			opts = append(opts, newSplitOption(func(cfg Config) Config {
				cfg.Logs.Endpoint = u.Host
				// For OTLP/HTTP endpoint URLs without a per-signal
				// configuration, the passed endpoint is used as a base URL
				// and the signals are sent to these paths relative to that.
				cfg.Logs.URLPath = path.Join(u.Path, DefaultLogsPath)
				return cfg
			}, withEndpointForGRPC(u)))
		}),
		envconfig.WithURL("LOGS_ENDPOINT", func(u *url.URL) {
			opts = append(opts, withEndpointScheme(u))
			opts = append(opts, newSplitOption(func(cfg Config) Config {
				cfg.Logs.Endpoint = u.Host
				// For endpoint URLs for OTLP/HTTP per-signal variables, the
				// URL MUST be used as-is without any modification. The only
				// exception is that if an URL contains no path part, the root
				// path / MUST be used.
				path := u.Path
				if path == "" {
					path = "/"
				}
				cfg.Logs.URLPath = path
				return cfg
			}, withEndpointForGRPC(u)))
		}),
		envconfig.WithString("PROTOCOL", func(s string) {
			opts = append(opts, withProtocol(s))
		}),
		envconfig.WithString("LOGS_PROTOCOL", func(s string) {
			opts = append(opts, withProtocol(s))
		}),
		envconfig.WithCertPool("CERTIFICATE", func(p *x509.CertPool) { tlsConf.RootCAs = p }),
		envconfig.WithCertPool("LOGS_CERTIFICATE", func(p *x509.CertPool) { tlsConf.RootCAs = p }),
		envconfig.WithClientCert("CLIENT_CERTIFICATE", "CLIENT_KEY", func(c tls.Certificate) { tlsConf.Certificates = []tls.Certificate{c} }),
		envconfig.WithClientCert("LOGS_CLIENT_CERTIFICATE", "LOGS_CLIENT_KEY", func(c tls.Certificate) { tlsConf.Certificates = []tls.Certificate{c} }),
		withTLSConfig(tlsConf, func(c *tls.Config) { opts = append(opts, WithTLSClientConfig(c)) }),
		envconfig.WithBool("INSECURE", func(b bool) { opts = append(opts, withInsecure(b)) }),
		envconfig.WithBool("LOGS_INSECURE", func(b bool) { opts = append(opts, withInsecure(b)) }),
		envconfig.WithHeaders("HEADERS", func(h map[string]string) { opts = append(opts, WithHeaders(h)) }),
		envconfig.WithHeaders("LOGS_HEADERS", func(h map[string]string) { opts = append(opts, WithHeaders(h)) }),
		WithEnvCompression("COMPRESSION", func(c Compression) { opts = append(opts, WithCompression(c)) }),
		WithEnvCompression("LOGS_COMPRESSION", func(c Compression) { opts = append(opts, WithCompression(c)) }),
		envconfig.WithDuration("TIMEOUT", func(d time.Duration) { opts = append(opts, WithTimeout(d)) }),
		envconfig.WithDuration("LOGS_TIMEOUT", func(d time.Duration) { opts = append(opts, WithTimeout(d)) }),
	)

	return opts
}

func withEndpointScheme(u *url.URL) GenericOption {
	switch strings.ToLower(u.Scheme) {
	case "http", "unix":
		return WithInsecure()
	default:
		return WithSecure()
	}
}

func withEndpointForGRPC(u *url.URL) func(cfg Config) Config {
	return func(cfg Config) Config {
		// For OTLP/gRPC endpoints, this is the target to which the
		// exporter is going to send telemetry.
		cfg.Logs.Endpoint = path.Join(u.Host, u.Path)
		return cfg
	}
}

// WithEnvCompression retrieves the specified config and passes it to ConfigFn as a Compression.
func WithEnvCompression(n string, fn func(Compression)) func(e *envconfig.EnvOptionsReader) {
	return func(e *envconfig.EnvOptionsReader) {
		if v, ok := e.GetEnvValue(n); ok {
			cp := NoCompression
			if v == "gzip" {
				cp = GzipCompression
			}

			fn(cp)
		}
	}
}

// revive:disable-next-line:flag-parameter
func withInsecure(b bool) GenericOption {
	if b {
		return WithInsecure()
	}
	return WithSecure()
}

func withTLSConfig(c *tls.Config, fn func(*tls.Config)) func(e *envconfig.EnvOptionsReader) {
	return func(e *envconfig.EnvOptionsReader) {
		if c.RootCAs != nil || len(c.Certificates) > 0 {
			fn(c)
		}
	}
}

func withProtocol(b string) GenericOption {
	return newGenericOption(func(cfg Config) Config {
		cfg.Logs.Protocol = stringToProtocol(b)
		return cfg
	})
}
