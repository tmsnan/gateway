// Copyright Envoy Gateway Authors
// SPDX-License-Identifier: Apache-2.0
// The full text of the Apache license is available in the LICENSE file at
// the root of the repo.

package bootstrap

import (
	// Register embed
	_ "embed"
	"fmt"
	"strings"
	"text/template"

	"k8s.io/apimachinery/pkg/util/sets"

	egcfgv1a1 "github.com/envoyproxy/gateway/api/config/v1alpha1"
)

const (
	// envoyCfgFileName is the name of the Envoy configuration file.
	envoyCfgFileName = "bootstrap.yaml"
	// envoyGatewayXdsServerHost is the DNS name of the Xds Server within Envoy Gateway.
	// It defaults to the Envoy Gateway Kubernetes service.
	envoyGatewayXdsServerHost = "envoy-gateway"
	// envoyAdminAddress is the listening address of the envoy admin interface.
	envoyAdminAddress = "127.0.0.1"
	// envoyAdminPort is the port used to expose admin interface.
	envoyAdminPort = 19000
	// envoyAdminAccessLogPath is the path used to expose admin access log.
	envoyAdminAccessLogPath = "/dev/null"

	// DefaultXdsServerPort is the default listening port of the xds-server.
	DefaultXdsServerPort = 18000

	envoyReadinessAddress = "0.0.0.0"
	EnvoyReadinessPort    = 19001
	EnvoyReadinessPath    = "/ready"
	// required stats are used by readiness checks.
	RequiredEnvoyStatsMatcherInclusionPrefixes = "cluster_manager,listener_manager,server,cluster.xds-grpc"
)

//go:embed bootstrap.yaml.tpl
var bootstrapTmplStr string

var bootstrapTmpl = template.Must(template.New(envoyCfgFileName).Parse(bootstrapTmplStr))

// envoyBootstrap defines the envoy Bootstrap configuration.
type bootstrapConfig struct {
	// parameters defines configurable bootstrap configuration parameters.
	parameters bootstrapParameters
	// rendered is the rendered bootstrap configuration.
	rendered string
}

// envoyBootstrap defines the envoy Bootstrap configuration.
type bootstrapParameters struct {
	// XdsServer defines the configuration of the XDS server.
	XdsServer xdsServerParameters
	// AdminServer defines the configuration of the Envoy admin interface.
	AdminServer adminServerParameters
	// ReadyServer defines the configuration for health check ready listener
	ReadyServer readyServerParameters
	// EnablePrometheus defines whether to enable metrics endpoint for prometheus.
	EnablePrometheus bool
	// OtelMetricSinks defines the configuration of the OpenTelemetry sinks.
	OtelMetricSinks []metricSink
	// Proxy stats matcher defines configuration for reporting custom Envoy stats.
	// To reduce memory and CPU overhead from Envoy stats system, Gateway proxies by
	// default create and expose only a subset of Envoy stats. This option is to
	// control creation of additional Envoy stats with prefix, suffix, and regex
	// expressions match on the name of the stats.
	// you can specify stats matcher as follows:
	// ```yaml
	// proxyStatsMatcher:
	//
	//	inclusionRegexps:
	//	  - .*outlier_detection.*
	//	  - .*upstream_rq_retry.*
	//	  - .*upstream_cx_.*
	//	inclusionSuffixes:
	//	  - upstream_rq_timeout
	//
	// ```
	// Note including more Envoy stats might increase number of time series
	// collected by prometheus significantly. Care needs to be taken on Prometheus
	// resource provision and configuration to reduce cardinality.
	ProxyStatsMatcher ProxyStatsMatcherParameters
}

type xdsServerParameters struct {
	// Address is the address of the XDS Server that Envoy is managed by.
	Address string
	// Port is the port of the XDS Server that Envoy is managed by.
	Port int32
}

type metricSink struct {
	// Address is the address of the XDS Server that Envoy is managed by.
	Address string
	// Port is the port of the XDS Server that Envoy is managed by.
	Port int32
}

type adminServerParameters struct {
	// Address is the address of the Envoy admin interface.
	Address string
	// Port is the port of the Envoy admin interface.
	Port int32
	// AccessLogPath is the path of the Envoy admin access log.
	AccessLogPath string
}

type readyServerParameters struct {
	// Address is the address of the Envoy readiness probe
	Address string
	// Port is the port of envoy readiness probe
	Port int32
	// ReadinessPath is the path for the envoy readiness probe
	ReadinessPath string
}

type ProxyStatsMatcherParameters struct {
	// Proxy stats name prefix matcher for inclusion.
	InclusionPrefixs []string
	// Proxy stats name suffix matcher for inclusion.
	InclusionSuffixs []string
	// Proxy stats name regexps matcher for inclusion.
	InclusionRegexps []string
}

// render the stringified bootstrap config in yaml format.
func (b *bootstrapConfig) render() error {
	buf := new(strings.Builder)
	if err := bootstrapTmpl.Execute(buf, b.parameters); err != nil {
		return fmt.Errorf("failed to render bootstrap config: %v", err)
	}
	b.rendered = buf.String()

	return nil
}

// GetRenderedBootstrapConfig renders the bootstrap YAML string
func GetRenderedBootstrapConfig(proxyMetrics *egcfgv1a1.ProxyMetrics) (string, error) {
	var (
		enablePrometheus  bool
		metricSinks       []metricSink
		ProxyStatsMatcher ProxyStatsMatcherParameters
	)

	if proxyMetrics != nil {
		if proxyMetrics.Prometheus != nil {
			enablePrometheus = true
		}

		addresses := sets.NewString()
		for _, sink := range proxyMetrics.Sinks {
			if sink.OpenTelemetry == nil {
				continue
			}

			// skip duplicate sinks
			addr := fmt.Sprintf("%s:%d", sink.OpenTelemetry.Host, sink.OpenTelemetry.Port)
			if addresses.Has(addr) {
				continue
			}
			addresses.Insert(addr)

			metricSinks = append(metricSinks, metricSink{
				Address: sink.OpenTelemetry.Host,
				Port:    sink.OpenTelemetry.Port,
			})
		}

		if proxyMetrics.ProxyStatsMatcher != nil {
			ProxyStatsMatcher = ProxyStatsMatcherParameters{
				InclusionPrefixs: proxyMetrics.ProxyStatsMatcher.InclusionPrefixs,
				InclusionSuffixs: proxyMetrics.ProxyStatsMatcher.InclusionSuffixs,
				InclusionRegexps: proxyMetrics.ProxyStatsMatcher.InclusionRegexps,
			}
		}
	}
	ProxyStatsMatcher.InclusionPrefixs = append(ProxyStatsMatcher.InclusionPrefixs, strings.Split(RequiredEnvoyStatsMatcherInclusionPrefixes, ",")...)
	//ProxyStatsMatcher.InclusionRegexps = append(ProxyStatsMatcher.InclusionRegexps, strings.Split(RequiredEnvoyStatsMatcherInclusionRegexes, ",")...)

	cfg := &bootstrapConfig{
		parameters: bootstrapParameters{
			XdsServer: xdsServerParameters{
				Address: envoyGatewayXdsServerHost,
				Port:    DefaultXdsServerPort,
			},
			AdminServer: adminServerParameters{
				Address:       envoyAdminAddress,
				Port:          envoyAdminPort,
				AccessLogPath: envoyAdminAccessLogPath,
			},
			ReadyServer: readyServerParameters{
				Address:       envoyReadinessAddress,
				Port:          EnvoyReadinessPort,
				ReadinessPath: EnvoyReadinessPath,
			},
			EnablePrometheus:  enablePrometheus,
			OtelMetricSinks:   metricSinks,
			ProxyStatsMatcher: ProxyStatsMatcher,
		},
	}

	if err := cfg.render(); err != nil {
		return "", err
	}

	return cfg.rendered, nil
}
