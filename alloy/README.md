# alloy

Starter configs for allow with Grafana Cloud

## Requeriments
- EKS cluster 
- Grafana Alloy https://grafana.com/docs/alloy/latest/tasks/configure/configure-kubernetes/
- Prometheus Operator https://github.com/prometheus-operator/prometheus-operator?tab=readme-ov-file#quickstart

## Examples
- Prometheus exporter `unix` metrics to Prometheus metrics service [config-prometheus-metrics-basic.alloy](config-prometheus-metrics-basic.alloy)
- Prometheus exporter `unix` metrics using OTLP protocol  [config-prometheus-metrics-to-otel-basic.alloy](config-prometheus-metrics-to-otel-basic.alloy)
- Pods metrics with `ServiceMonitor` using OTLP protocol [prometheus-example-app-metrics.yaml](prometheus-example-app-metrics.yaml) and [config-pods-metrics-to-otel-basic.alloy](config-pods-metrics-to-otel-basic.alloy) requires `Prometheus Operator`
- Pod logs with `PodLogs` `monitoring.grafana.com/v1alpha2` to Loki [pods-logs-to-loki-basic.yaml](pods-logs-to-loki-basic.yaml) and [config-pods-logs-to-loki-basic.alloy](config-pods-logs-to-loki-basic.alloy) requires `Grafana Alloy`
