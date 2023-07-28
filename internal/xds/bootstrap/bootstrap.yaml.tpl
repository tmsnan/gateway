admin:
  access_log:
  - name: envoy.access_loggers.file
    typed_config:
      "@type": type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog
      path: {{ .AdminServer.AccessLogPath }}
  address:
    socket_address:
      address: {{ .AdminServer.Address }}
      port_value: {{ .AdminServer.Port }}
stats_config:
  stats_matcher:
    inclusion_list:
      patterns:
      {{- range $a, $s := .ProxyStatsMatcher.InclusionPrefixs }}
      - prefix: "{{$s}}"
      {{- end}}
      {{- range $a, $s := .ProxyStatsMatcher.InclusionSuffixs }}
      - suffix: "{{$s}}"
      {{- end}}
      {{- range $a, $s := .ProxyStatsMatcher.InclusionRegexps }}
      - safe_regex:
          google_re2: {}
          regex: "{{js $s}}"
      {{- end}}
dynamic_resources:
  ads_config:
    api_type: DELTA_GRPC
    transport_api_version: V3
    grpc_services:
    - envoy_grpc:
        cluster_name: xds_cluster
    set_node_on_first_message_only: true
  lds_config:
    ads: {}
    resource_api_version: V3
  cds_config:
    ads: {}
    resource_api_version: V3
{{- if .OtelMetricSinks }}
stats_sinks:
{{- range $idx, $sink := .OtelMetricSinks }}
- name: "envoy.stat_sinks.open_telemetry"
  typed_config:
    "@type": type.googleapis.com/envoy.extensions.stat_sinks.open_telemetry.v3.SinkConfig
    grpc_service:
      envoy_grpc:
        cluster_name: otel_metric_sink_{{ $idx }}
{{- end }}
{{- end }}
static_resources:
  listeners:
  - name: envoy-gateway-proxy-ready-{{ .ReadyServer.Address }}-{{ .ReadyServer.Port }}
    address:
      socket_address:
        address: {{ .ReadyServer.Address }}
        port_value: {{ .ReadyServer.Port }}
        protocol: TCP
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          stat_prefix: eg-ready-http
          route_config:
            name: local_route
            {{- if .EnablePrometheus }}
            virtual_hosts:
            - name: prometheus_stats
              domains:
              - "*"
              routes:
              - match:
                  prefix: /stats/prometheus
                route:
                  cluster: prometheus_stats
            {{- end }}
          http_filters:
          - name: envoy.filters.http.health_check
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.http.health_check.v3.HealthCheck
              pass_through_mode: false
              headers:
              - name: ":path"
                string_match:
                  exact: {{ .ReadyServer.ReadinessPath }}
          - name: envoy.filters.http.router
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
  clusters:
  {{- if .EnablePrometheus }}
  - name: prometheus_stats
    connect_timeout: 0.250s
    type: STATIC
    lb_policy: ROUND_ROBIN
    load_assignment:
      cluster_name: prometheus_stats
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: {{ .AdminServer.Address }}
                port_value: {{ .AdminServer.Port }}
  {{- end }}
  {{- range $idx, $sink := .OtelMetricSinks }}
  - name: otel_metric_sink_{{ $idx }}
    connect_timeout: 0.250s
    type: STRICT_DNS
    typed_extension_protocol_options:
      envoy.extensions.upstreams.http.v3.HttpProtocolOptions:
        "@type": "type.googleapis.com/envoy.extensions.upstreams.http.v3.HttpProtocolOptions"
        explicit_http_config:
          http2_protocol_options: {}
    lb_policy: ROUND_ROBIN
    load_assignment:
      cluster_name: otel_metric_sink_{{ $idx }}
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: {{ $sink.Address }}
                port_value: {{ $sink.Port }}
  {{- end }}
  - connect_timeout: 10s
    load_assignment:
      cluster_name: xds_cluster
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: {{ .XdsServer.Address }}
                port_value: {{ .XdsServer.Port }}
    typed_extension_protocol_options:
      envoy.extensions.upstreams.http.v3.HttpProtocolOptions:
        "@type": "type.googleapis.com/envoy.extensions.upstreams.http.v3.HttpProtocolOptions"
        explicit_http_config:
          http2_protocol_options: {}
    name: xds_cluster
    type: STRICT_DNS
    transport_socket:
      name: envoy.transport_sockets.tls
      typed_config:
        "@type": type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext
        common_tls_context:
          tls_params:
            tls_maximum_protocol_version: TLSv1_3
          tls_certificate_sds_secret_configs:
          - name: xds_certificate
            sds_config:
              path_config_source:
                path: "/sds/xds-certificate.json"
              resource_api_version: V3
          validation_context_sds_secret_config:
            name: xds_trusted_ca
            sds_config:
              path_config_source:
                path: "/sds/xds-trusted-ca.json"
              resource_api_version: V3
layered_runtime:
  layers:
  - name: runtime-0
    rtds_layer:
      rtds_config:
        ads: {}
        resource_api_version: V3
      name: runtime-0
