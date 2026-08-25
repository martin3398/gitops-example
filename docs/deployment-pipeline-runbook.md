# Deployment Pipeline Runbook

This runbook describes the current dev deployment flow.

## Main Commands

```bash
task pipeline:check
task pipeline:init_cluster
task ansible:core
task ansible:openbao
task pipeline:verify
task pipeline:main
```

## Pipeline Shape

`task pipeline:main` runs:

```text
pipeline:init_cluster
ansible:core
ansible:openbao
pipeline:verify
```

Expanded sequence:

1. `tofu:plan`
2. `tofu:apply`
3. `ansible:inventory`
4. `ansible:smoke`
5. `ansible:base`
6. `ansible:runtime`
7. `ansible:bootstrap`
8. `ansible:core`
9. `ansible:openbao`
10. `pipeline:verify`

## Task Responsibilities

`pipeline:init_cluster`:

- provisions AWS infrastructure with OpenTofu
- exports OpenTofu outputs
- generates Ansible inventory
- validates SSM connectivity
- prepares hosts
- installs runtime dependencies
- bootstraps RKE2 and bundled Cilium

`ansible:core`:

- installs Flux controllers
- creates Flux Git deploy credentials
- applies `kubernetes/clusters/dev/`
- waits for `GitRepository/dev-repo`

`ansible:openbao`:

- waits for `infrastructure-data-ceph`
- initializes OpenBao if needed
- saves init material remotely and fetches it locally to `.secrets/openbao-init.dev.json`
- unseals leader and followers
- configures Kubernetes auth for External Secrets Operator
- seeds deterministic dev secrets, including Loki S3 credentials
- waits for `ClusterSecretStore/openbao`

`pipeline:verify`:

- verifies node readiness and the expected 6-node topology
- verifies Flux source, Kustomizations, and HelmReleases are Ready
- verifies Ceph health, block pool readiness, and PVC binding
- verifies monitoring/logging workload readiness
- verifies OpenBao is initialized and unsealed
- verifies Postgres and Kafka custom resources and pods are Ready
- verifies visit demo deployments and the gateway queue/count path

## Flux Stage Layout

Flux cluster stage definitions live under:

```text
kubernetes/clusters/dev/
```

The stage files are:

```text
gitrepository.yaml
kustomization-infrastructure-core.yaml
kustomization-infrastructure-scheduling.yaml
kustomization-infrastructure-data-ceph.yaml
kustomization-infrastructure-gateway.yaml
kustomization-infrastructure-security.yaml
kustomization-infrastructure-data-postgres.yaml
kustomization-infrastructure-data-kafka.yaml
kustomization-infrastructure-observability.yaml
apps/
```

Each stage uses `wait: true` and `prune: true`. `infrastructure-core`, `infrastructure-gateway`, `infrastructure-data-kafka`, and the app stages also carry explicit health checks.

## Why It Is Staged

OpenBao cannot be fully bootstrapped by declarative manifests alone in this lab flow. It requires procedural init, unseal, auth, and seed steps.

`infrastructure-data-postgres`, `infrastructure-observability` Loki bootstrap, the Gateway API routes, and the visit workloads depend on the OpenBao-backed `ClusterSecretStore`, so they are verified only after OpenBao bootstrap completes.

Monitoring and the public gateway are staged separately because Loki, Promtail, and the monitoring routes can render `ServiceMonitor` resources that need Prometheus Operator CRDs from `kube-prometheus-stack`.

## Recovery

If a Helm release fails because a CRD is missing, reconcile the upstream stage after the dependency is Ready:

```bash
flux reconcile kustomization infrastructure-observability -n flux-system --timeout=20m
flux reconcile kustomization infrastructure-gateway -n flux-system --timeout=15m
```

If OpenBao gets stuck in a bad Raft state, reset the init material and re-run `task ansible:openbao`.

## Validation

```bash
task pipeline:verify
kubectl -n flux-system get gitrepositories
kubectl -n flux-system get kustomizations
kubectl -n flux-system get helmreleases
kubectl -n openbao get pods
kubectl get clustersecretstore openbao
```
