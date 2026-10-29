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
- Dashboards (`monitoring` namespace, auto-discovered via `grafana_dashboard: "1"` label):
  - `dashboard-gitops-flux.yaml`: Flux GitOps Reconciliation Overview (UID: `flux-overview`)
  - `dashboard-visit-processing-overview.yaml`: Visit Processing Workload & HPA Overview (UID: `visit-processing-overview`)
  - `dashboards/ceph-overview.yaml`: Ceph Storage & OSD Health Overview (UID: `ceph-cluster-overview`)
  - `dashboards/postgres-overview.yaml`: CloudNativePG Cluster Performance & Replication Lag (UID: `postgres-cluster-overview`)
  - `dashboards/kafka-overview.yaml`: Strimzi Kafka Brokers, Topics & Consumer Group Lag (UID: `kafka-cluster-overview`)
  - `dashboards/gateway-overview.yaml`: Cilium Gateway API Traffic, Latency & HTTP Error Rates (UID: `gateway-api-overview`)

## Current Baseline

- Monitoring namespace: `monitoring`
- Grafana exposes browser access through `http://grafana.gitops.local`
- Prometheus exposes browser access through `http://prometheus.gitops.local`
- Loki runs in distributed mode on Ceph-backed S3 object storage, with Loki S3 credentials sourced from OpenBao and bootstrapped through `loki-bootstrap/`
- Promtail provides cluster log aggregation
- Prometheus Adapter exports custom metrics for workload scaling and dashboards
- Cilium Gateway API access logs are available through Hubble Relay; application and platform logs still flow to Loki and Grafana

## Platform Dashboards Inventory

All dashboards are provisioned as Kubernetes ConfigMaps labeled `grafana_dashboard: "1"` in the `monitoring` namespace. The `kube-prometheus-stack` Grafana sidecar automatically detects, imports, and maintains them:

| Dashboard Title | UID | Target Subsystem | Key Monitored Metrics |
| :--- | :--- | :--- | :--- |
| **Flux GitOps Overview** | `flux-overview` | Flux v2 Controllers | Ready reconciliations, failure counts, reconciliation duration p95 |
| **Visit Processing Overview** | `visit-processing-overview` | Visit Processor Workload | Pod replica counts, HPA target vs actual, consumer lag per pod |
| **Ceph Storage & OSD Overview** | `ceph-cluster-overview` | Rook-Ceph Storage Cluster | Cluster health status, OSD up/in count, MON quorum, raw capacity %, pool IOPS/throughput, PG clean/active states, OSD commit/apply latency |
| **PostgreSQL CloudNativePG Overview** | `postgres-cluster-overview` | CloudNativePG Database Cluster | Primary / standby status, replication lag by pod, DB size, TPS (commits/rollbacks), connections vs limit, CRUD throughput, cache hit ratio, WAL archiving activity |
| **Strimzi Kafka Cluster & Topics Overview** | `kafka-cluster-overview` | Strimzi Kafka & Exporter | Active broker count, consumer lag by topic/group, DLQ total message count (`visits.dead-letter`), production rate, partition offset ranges, ISR replica health |
| **Cilium Gateway API Overview** | `gateway-api-overview` | Cilium Gateway API / Envoy | Gateway readiness, HTTP request throughput, 5xx error rate, upstream target distribution, p50/p90/p99 request duration, ingress/egress network bandwidth, active connections |

## Validation

Use a workstation with cluster access.

```bash
kubectl -n flux-system get kustomizations,helmreleases
kubectl -n monitoring get pods
kubectl -n monitoring get svc
kubectl -n monitoring get configmaps -l grafana_dashboard=1
task ingress:hosts_entries
task pipeline:verify
```

## Troubleshooting

- If Grafana or Prometheus is unreachable, confirm the gateway stage is Ready and the local host entries point at the current gateway service address.
- If metrics are missing, confirm `kube-prometheus-stack` and `prometheus-adapter` are both Ready.
- If dashboards do not appear in Grafana, verify the ConfigMap has `grafana_dashboard: "1"` in namespace `monitoring` and check Grafana sidecar logs: `kubectl -n monitoring logs -l app.kubernetes.io/name=grafana -c grafana-sc-dashboard`.
- If application logs are missing, confirm Promtail is running on all nodes, Loki has healthy pods, and the Loki bootstrap resources have created the RGW user and buckets.
- If gateway request visibility is missing, confirm the Cilium gateway pods are healthy and query Hubble Relay for HTTP flows.

## Completed Milestones

- **Platform Dashboard Expansion (`TASK-P4-05`)**: Added auto-discovered Grafana dashboards covering Rook-Ceph storage, CloudNativePG database clusters, Strimzi Kafka topics and consumer groups, and Cilium Gateway API edge traffic.
