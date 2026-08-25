# Observability Runbook (Monitoring + Logging, Dev)

This runbook covers the observability stack deployed by Flux.

## Components

- kube-prometheus-stack: `kubernetes/infrastructure/base/observability/helmrelease-kube-prometheus-stack.yaml`
- Loki: `kubernetes/infrastructure/base/observability/helmrelease-loki.yaml`
- Promtail: `kubernetes/infrastructure/base/observability/helmrelease-promtail.yaml`
- Loki bootstrap: `kubernetes/infrastructure/base/observability/loki-bootstrap/`
- Prometheus Adapter: `kubernetes/infrastructure/base/observability/prometheus-adapter/`
- Kafka exporter: `kubernetes/infrastructure/base/observability/prometheus-kafka-exporter/`
- Gateway HTTPRoutes: configured in `helmrelease-kube-prometheus-stack.yaml`
- Dashboards:
  - `dashboard-gitops-flux.yaml`
  - `dashboard-visit-processing-overview.yaml`

## Current Baseline

- Monitoring namespace: `monitoring`
- Grafana exposes browser access through `http://grafana.gitops.local`
- Prometheus exposes browser access through `http://prometheus.gitops.local`
- Loki runs in distributed mode on Ceph-backed S3 object storage, with Loki S3 credentials sourced from OpenBao and bootstrapped through `loki-bootstrap/`
- Promtail provides cluster log aggregation
- Prometheus Adapter exports custom metrics for workload scaling and dashboards
- Cilium Gateway API access logs are available through Hubble Relay; application and platform logs still flow to Loki and Grafana

## Validation

Use a workstation with cluster access.

```bash
kubectl -n flux-system get kustomizations,helmreleases
kubectl -n monitoring get pods
kubectl -n monitoring get svc
task ingress:hosts_entries
task pipeline:verify
```

## Troubleshooting

- If Grafana or Prometheus is unreachable, confirm the gateway stage is Ready and the local host entries point at the current gateway service address.
- If metrics are missing, confirm `kube-prometheus-stack` and `prometheus-adapter` are both Ready.
- If application logs are missing, confirm Promtail is running on all nodes, Loki has healthy pods, and the Loki bootstrap resources have created the RGW user and buckets.
- If gateway request visibility is missing, confirm the Cilium gateway pods are healthy and query Hubble Relay for HTTP flows.
