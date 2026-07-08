# Observability Runbook (Monitoring + Logging, Dev)

This runbook covers the observability stack deployed by Flux.

## Components

- kube-prometheus-stack: `kubernetes/infrastructure/base/observability/helmrelease-kube-prometheus-stack.yaml`
- Loki: `kubernetes/infrastructure/base/observability/helmrelease-loki.yaml`
- Promtail: `kubernetes/infrastructure/base/observability/helmrelease-promtail.yaml`
- Prometheus Adapter: `kubernetes/infrastructure/base/observability/prometheus-adapter/`
- Kafka exporter: `kubernetes/infrastructure/base/observability/prometheus-kafka-exporter/`
- Grafana ingress: `kubernetes/infrastructure/base/observability/grafana-ingress.yaml`
- Prometheus ingress: `kubernetes/infrastructure/base/observability/prometheus-ingress.yaml`
- Dashboards:
  - `dashboard-gitops-flux.yaml`
  - `dashboard-ingress-nginx.yaml`
  - `dashboard-visit-processing-overview.yaml`

## Current Baseline

- Monitoring namespace: `monitoring`
- Grafana exposes browser access through `http://grafana.gitops.local`
- Prometheus exposes browser access through `http://prometheus.gitops.local`
- Loki and Promtail provide cluster log aggregation
- Prometheus Adapter exports custom metrics for workload scaling and dashboards

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

- If Grafana or Prometheus is unreachable, confirm the ingress stage is Ready and the local host entries point at the current NLB addresses.
- If metrics are missing, confirm `kube-prometheus-stack` and `prometheus-adapter` are both Ready.
- If logs are missing, confirm Promtail is running on all nodes and Loki has healthy PVCs.
