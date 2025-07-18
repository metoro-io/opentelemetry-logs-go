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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/metoro-io/opentelemetry-logs-go/exporters/otlp/otlplogs/internal/envconfig"
)

const (
	WeakCertificate = `
-----BEGIN CERTIFICATE-----
MIIBhzCCASygAwIBAgIRANHpHgAWeTnLZpTSxCKs0ggwCgYIKoZIzj0EAwIwEjEQ
MA4GA1UEChMHb3RlbC1nbzAeFw0yMTA0MDExMzU5MDNaFw0yMTA0MDExNDU5MDNa
MBIxEDAOBgNVBAoTB290ZWwtZ28wWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAS9
nWSkmPCxShxnp43F+PrOtbGV7sNfkbQ/kxzi9Ego0ZJdiXxkmv/C05QFddCW7Y0Z
sJCLHGogQsYnWJBXUZOVo2MwYTAOBgNVHQ8BAf8EBAMCB4AwEwYDVR0lBAwwCgYI
KwYBBQUHAwEwDAYDVR0TAQH/BAIwADAsBgNVHREEJTAjgglsb2NhbGhvc3SHEAAA
AAAAAAAAAAAAAAAAAAGHBH8AAAEwCgYIKoZIzj0EAwIDSQAwRgIhANwZVVKvfvQ/
1HXsTvgH+xTQswOwSSKYJ1cVHQhqK7ZbAiEAus8NxpTRnp5DiTMuyVmhVNPB+bVH
Lhnm4N/QDk5rek0=
-----END CERTIFICATE-----
`
	WeakPrivateKey = `
-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgN8HEXiXhvByrJ1zK
SFT6Y2l2KqDWwWzKf+t4CyWrNKehRANCAAS9nWSkmPCxShxnp43F+PrOtbGV7sNf
kbQ/kxzi9Ego0ZJdiXxkmv/C05QFddCW7Y0ZsJCLHGogQsYnWJBXUZOV
-----END PRIVATE KEY-----
`
)

type env map[string]string

func (e *env) getEnv(env string) string {
	return (*e)[env]
}

type fileReader map[string][]byte

func (f *fileReader) readFile(filename string) ([]byte, error) {
	if b, ok := (*f)[filename]; ok {
		return b, nil
	}
	return nil, errors.New("file not found")
}

func TestConfigs(t *testing.T) {
	tlsCert, err := CreateTLSConfig([]byte(WeakCertificate))
	assert.NoError(t, err)

	tests := []struct {
		name       string
		opts       []GenericOption
		env        env
		fileReader fileReader
		asserts    func(t *testing.T, c *Config, grpcOption bool)
	}{
		{
			name: "Test default configs",
			asserts: func(t *testing.T, c *Config, grpcOption bool) {
				if grpcOption {
					assert.Equal(t, "localhost:4317", c.Logs.Endpoint)
				} else {
					assert.Equal(t, "localhost:4318", c.Logs.Endpoint)
				}
				assert.Equal(t, NoCompression, c.Logs.Compression)
				assert.Equal(t, map[string]string(nil), c.Logs.Headers)
				assert.Equal(t, 10*time.Second, c.Logs.Timeout)
			},
		},

		// Endpoint Tests
		{
			name: "Test With Endpoint",
			opts: []GenericOption{
				WithEndpoint("someendpoint"),
			},
			asserts: func(t *testing.T, c *Config, grpcOption bool) {
				assert.Equal(t, "someendpoint", c.Logs.Endpoint)
			},
		},
		{
			name: "Test Environment Endpoint",
			env: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "https://env.endpoint/prefix",
			},
			asserts: func(t *testing.T, c *Config, grpcOption bool) {
				assert.False(t, c.Logs.Insecure)
				if grpcOption {
					assert.Equal(t, "env.endpoint/prefix", c.Logs.Endpoint)
				} else {
					assert.Equal(t, "env.endpoint", c.Logs.Endpoint)
					assert.Equal(t, "/prefix/v1/logs", c.Logs.URLPath)
				}
			},
		},
		{
			name: "Test Environment Signal Specific Endpoint",
			env: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT":      "https://override.by.signal.specific/env/var",
				"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT": "http://env.logs.endpoint",
			},
			asserts: func(t *testing.T, c *Config, grpcOption bool) {
				assert.True(t, c.Logs.Insecure)
				assert.Equal(t, "env.logs.endpoint", c.Logs.Endpoint)
				if !grpcOption {
					assert.Equal(t, "/", c.Logs.URLPath)
				}
			},
		},
		{
			name: "Test Mixed Environment and With Endpoint",
			opts: []GenericOption{
				WithEndpoint("logs_endpoint"),
			},
			env: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "env_endpoint",
			},
			asserts: func(t *testing.T, c *Config, grpcOption bool) {
				assert.Equal(t, "logs_endpoint", c.Logs.Endpoint)
			},
		},
		{
			name: "Test Environment Endpoint with HTTP scheme",
			env: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://env_endpoint",
			},
			asserts: func(t *testing.T, c *Config, grpcOption bool) {
				assert.Equal(t, "env_endpoint", c.Logs.Endpoint)
				assert.Equal(t, true, c.Logs.Insecure)
			},
		},
		{
			name: "Test Environment Endpoint with HTTP scheme and leading & trailingspaces",
			env: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "      http://env_endpoint    ",
			},
			asserts: func(t *testing.T, c *Config, grpcOption bool) {
				assert.Equal(t, "env_endpoint", c.Logs.Endpoint)
				assert.Equal(t, true, c.Logs.Insecure)
			},
		},
		{
			name: "Test Environment Endpoint with HTTPS scheme",
			env: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "https://env_endpoint",
			},
			asserts: func(t *testing.T, c *Config, grpcOption bool) {
				assert.Equal(t, "env_endpoint", c.Logs.Endpoint)
				assert.Equal(t, false, c.Logs.Insecure)
			},
		},
		{
			name: "Test Environment Signal Specific Endpoint with uppercase scheme",
			env: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT":      "HTTPS://overrode_by_signal_specific",
				"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT": "HtTp://env_logs_endpoint",
			},
			asserts: func(t *testing.T, c *Config, grpcOption bool) {
				assert.Equal(t, "env_logs_endpoint", c.Logs.Endpoint)
				assert.Equal(t, true, c.Logs.Insecure)
			},
		},

		// Certificate tests
		{
			name: "Test Default Certificate",
			asserts: func(t *testing.T, c *Config, grpcOption bool) {
				if grpcOption {
					assert.NotNil(t, c.Logs.GRPCCredentials)
				} else {
					assert.Nil(t, c.Logs.TLSCfg)
				}
			},
		},
		{
			name: "Test With Certificate",
			opts: []GenericOption{
				WithTLSClientConfig(tlsCert),
			},
			asserts: func(t *testing.T, c *Config, grpcOption bool) {
				if grpcOption {
					//TODO: make sure gRPC's credentials actually works
					assert.NotNil(t, c.Logs.GRPCCredentials)
				} else {
					// nolint:staticcheck // ignoring tlsCert.RootCAs.Subjects is deprecated ERR because cert does not come from SystemCertPool.
					assert.Equal(t, tlsCert.RootCAs.Subjects(), c.Logs.TLSCfg.RootCAs.Subjects())
				}
			},
		},
		{
			name: "Test Environment Certificate",
			env: map[string]string{
				"OTEL_EXPORTER_OTLP_CERTIFICATE": "cert_path",
			},
			fileReader: fileReader{
				"cert_path": []byte(WeakCertificate),
			},
			asserts: func(t *testing.T, c *Config, grpcOption bool) {
				if grpcOption {
					assert.NotNil(t, c.Logs.GRPCCredentials)
				} else {
					// nolint:staticcheck // ignoring tlsCert.RootCAs.Subjects is deprecated ERR because cert does not come from SystemCertPool.
					assert.Equal(t, tlsCert.RootCAs.Subjects(), c.Logs.TLSCfg.RootCAs.Subjects())
				}
			},
		},
		{
			name: "Test Environment Signal Specific Certificate",
			env: map[string]string{
				"OTEL_EXPORTER_OTLP_CERTIFICATE":      "overrode_by_signal_specific",
				"OTEL_EXPORTER_OTLP_LOGS_CERTIFICATE": "cert_path",
			},
			fileReader: fileReader{
				"cert_path":    []byte(WeakCertificate),
				"invalid_cert": []byte("invalid certificate file."),
			},
			asserts: func(t *testing.T, c *Config, grpcOption bool) {
				if grpcOption {
					assert.NotNil(t, c.Logs.GRPCCredentials)
				} else {
					// nolint:staticcheck // ignoring tlsCert.RootCAs.Subjects is deprecated ERR because cert does not come from SystemCertPool.
					assert.Equal(t, tlsCert.RootCAs.Subjects(), c.Logs.TLSCfg.RootCAs.Subjects())
				}
			},
		},
		{
			name: "Test Mixed Environment and With Certificate",
			opts: []GenericOption{},
			env: map[string]string{
				"OTEL_EXPORTER_OTLP_CERTIFICATE": "cert_path",
			},
			fileReader: fileReader{
				"cert_path": []byte(WeakCertificate),
			},
			asserts: func(t *testing.T, c *Config, grpcOption bool) {
				if grpcOption {
					assert.NotNil(t, c.Logs.GRPCCredentials)
				} else {
					// nolint:staticcheck // ignoring tlsCert.RootCAs.Subjects is deprecated ERR because cert does not come from SystemCertPool.
					assert.Equal(t, tlsCert.RootCAs.Subjects(), c.Logs.TLSCfg.RootCAs.Subjects())
				}
			},
		},

		// Headers tests
		{
			name: "Test With Headers",
			opts: []GenericOption{
				WithHeaders(map[string]string{"h1": "v1"}),
			},
			asserts: func(t *testing.T, c *Config, grpcOption bool) {
				assert.Equal(t, map[string]string{"h1": "v1"}, c.Logs.Headers)
			},
		},
		{
			name: "Test Environment Headers",
			env:  map[string]string{"OTEL_EXPORTER_OTLP_HEADERS": "h1=v1,h2=v2"},
			asserts: func(t *testing.T, c *Config, grpcOption bool) {
				assert.Equal(t, map[string]string{"h1": "v1", "h2": "v2"}, c.Logs.Headers)
			},
		},
		{
			name: "Test Environment Signal Specific Headers",
			env: map[string]string{
				"OTEL_EXPORTER_OTLP_HEADERS":      "overrode_by_signal_specific",
				"OTEL_EXPORTER_OTLP_LOGS_HEADERS": "h1=v1,h2=v2",
			},
			asserts: func(t *testing.T, c *Config, grpcOption bool) {
				assert.Equal(t, map[string]string{"h1": "v1", "h2": "v2"}, c.Logs.Headers)
			},
		},
		{
			name: "Test Mixed Environment and With Headers",
			env:  map[string]string{"OTEL_EXPORTER_OTLP_HEADERS": "h1=v1,h2=v2"},
			opts: []GenericOption{},
			asserts: func(t *testing.T, c *Config, grpcOption bool) {
				assert.Equal(t, map[string]string{"h1": "v1", "h2": "v2"}, c.Logs.Headers)
			},
		},

		// Compression Tests
		{
			name: "Test With Compression",
			opts: []GenericOption{
				WithCompression(GzipCompression),
			},
			asserts: func(t *testing.T, c *Config, grpcOption bool) {
				assert.Equal(t, GzipCompression, c.Logs.Compression)
			},
		},
		{
			name: "Test Environment Compression",
			env: map[string]string{
				"OTEL_EXPORTER_OTLP_COMPRESSION": "gzip",
			},
			asserts: func(t *testing.T, c *Config, grpcOption bool) {
				assert.Equal(t, GzipCompression, c.Logs.Compression)
			},
		},
		{
			name: "Test Environment Signal Specific Compression",
			env: map[string]string{
				"OTEL_EXPORTER_OTLP_LOGS_COMPRESSION": "gzip",
			},
			asserts: func(t *testing.T, c *Config, grpcOption bool) {
				assert.Equal(t, GzipCompression, c.Logs.Compression)
			},
		},
		{
			name: "Test Mixed Environment and With Compression",
			opts: []GenericOption{
				WithCompression(NoCompression),
			},
			env: map[string]string{
				"OTEL_EXPORTER_OTLP_LOGS_COMPRESSION": "gzip",
			},
			asserts: func(t *testing.T, c *Config, grpcOption bool) {
				assert.Equal(t, NoCompression, c.Logs.Compression)
			},
		},

		// Timeout Tests
		{
			name: "Test With Timeout",
			opts: []GenericOption{
				WithTimeout(time.Duration(5 * time.Second)),
			},
			asserts: func(t *testing.T, c *Config, grpcOption bool) {
				assert.Equal(t, 5*time.Second, c.Logs.Timeout)
			},
		},
		{
			name: "Test Environment Timeout",
			env: map[string]string{
				"OTEL_EXPORTER_OTLP_TIMEOUT": "15000",
			},
			asserts: func(t *testing.T, c *Config, grpcOption bool) {
				assert.Equal(t, c.Logs.Timeout, 15*time.Second)
			},
		},
		{
			name: "Test Environment Signal Specific Timeout",
			env: map[string]string{
				"OTEL_EXPORTER_OTLP_TIMEOUT":      "15000",
				"OTEL_EXPORTER_OTLP_LOGS_TIMEOUT": "27000",
			},
			asserts: func(t *testing.T, c *Config, grpcOption bool) {
				assert.Equal(t, c.Logs.Timeout, 27*time.Second)
			},
		},
		{
			name: "Test Mixed Environment and With Timeout",
			env: map[string]string{
				"OTEL_EXPORTER_OTLP_TIMEOUT":      "15000",
				"OTEL_EXPORTER_OTLP_LOGS_TIMEOUT": "27000",
			},
			opts: []GenericOption{
				WithTimeout(5 * time.Second),
			},
			asserts: func(t *testing.T, c *Config, grpcOption bool) {
				assert.Equal(t, c.Logs.Timeout, 5*time.Second)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origEOR := DefaultEnvOptionsReader
			DefaultEnvOptionsReader = envconfig.EnvOptionsReader{
				GetEnv:    tt.env.getEnv,
				ReadFile:  tt.fileReader.readFile,
				Namespace: "OTEL_EXPORTER_OTLP",
			}
			t.Cleanup(func() { DefaultEnvOptionsReader = origEOR })

			// Tests Generic options as HTTP Options
			cfg := NewHTTPConfig(asHTTPOptions(tt.opts)...)
			tt.asserts(t, &cfg, false)

			// Tests Generic options as gRPC Options
			cfg = NewGRPCConfig(asGRPCOptions(tt.opts)...)
			tt.asserts(t, &cfg, true)
		})
	}
}

func asHTTPOptions(opts []GenericOption) []HTTPOption {
	converted := make([]HTTPOption, len(opts))
	for i, o := range opts {
		converted[i] = NewHTTPOption(o.ApplyHTTPOption)
	}
	return converted
}

func asGRPCOptions(opts []GenericOption) []GRPCOption {
	converted := make([]GRPCOption, len(opts))
	for i, o := range opts {
		converted[i] = NewGRPCOption(o.ApplyGRPCOption)
	}
	return converted
}

func TestCleanPath(t *testing.T) {
	type args struct {
		urlPath     string
		defaultPath string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "clean empty path",
			args: args{
				urlPath:     "",
				defaultPath: "DefaultPath",
			},
			want: "DefaultPath",
		},
		{
			name: "clean metrics path",
			args: args{
				urlPath:     "/prefix/v1/metrics",
				defaultPath: "DefaultMetricsPath",
			},
			want: "/prefix/v1/metrics",
		},
		{
			name: "clean logs path",
			args: args{
				urlPath:     "https://env_endpoint",
				defaultPath: "DefaultLogsPath",
			},
			want: "/https:/env_endpoint",
		},
		{
			name: "spaces trimmed",
			args: args{
				urlPath: " /dir",
			},
			want: "/dir",
		},
		{
			name: "clean path empty",
			args: args{
				urlPath:     "dir/..",
				defaultPath: "DefaultLogsPath",
			},
			want: "DefaultLogsPath",
		},
		{
			name: "make absolute",
			args: args{
				urlPath: "dir/a",
			},
			want: "/dir/a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleanPath(tt.args.urlPath, tt.args.defaultPath); got != tt.want {
				t.Errorf("CleanPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
