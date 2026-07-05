# Roadmap

This file tracks the remaining work after the current Phase 1-3 baseline.

## Done

- Phase 1 infrastructure and cluster bootstrap are implemented.
- Phase 2 GitOps platform and workloads are implemented.
- Visit demo delivery, image automation, and the main platform baselines are implemented.

## Phase 3 - Remaining Hardening

- Loki distributed mode on Ceph object storage.
- OpenBao hardening: snapshots, restore, auto-unseal decision, raft peer validation.
- Credential/password hardening: dynamic DB credentials, secret rotation, stronger init material handling, GHCR pull secrets, Flux Git write credentials.
- Cluster authn/authz hardening.

## Phase 4 - Resilience, Backup, Policy & Security

- `Renovate` or similar dependency/update automation.
- Cluster failover and infrastructure update scenarios.
- `Velero` backups and restore drills.
- Grafana dashboard coverage for the current platform and app path.
- Network policy and policy/security expansion.

## Phase 5 - RKE2 Migration / Regulatory Hardening

- Migrate the cluster runtime to `RKE2`.
- Revalidate bootstrap, CNI, observability, storage, and workloads on the new runtime.

## Phase 6 - Ingress Platform Migration

- Migrate ingress from `ingress-nginx` to `Cilium` Gateway API.
- Validate `/` and `/api` behavior for the visit app under the new ingress path.

## Notes

- `visit-processor` autoscaling and load generation are already implemented baselines.
- The remaining work is mostly hardening, resilience, and migration scope.
