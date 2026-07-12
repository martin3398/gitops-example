# Roadmap

This file tracks the remaining work after the current Phase 1-5 baseline.

## Done

- Phase 1 infrastructure and cluster bootstrap are implemented.
- Phase 2 GitOps platform and workloads are implemented.
- Visit demo delivery, image automation, and the main platform baselines are implemented.
- Loki distributed mode on Ceph object storage is implemented.
- Phase 5 RKE2 migration is implemented.

## Phase 3 - Remaining Hardening

- OpenBao hardening: snapshots, restore, auto-unseal decision, raft peer validation.
- Credential/password hardening: dynamic DB credentials, secret rotation, stronger init material handling, GHCR pull secrets, Flux Git write credentials.
- Cluster authn/authz hardening.

## Phase 4 - Resilience, Backup, Policy & Security

- `Renovate` or similar dependency/update automation.
- Cluster failover and infrastructure update scenarios.
- `Velero` backups and restore drills.
- Grafana dashboard coverage for the current platform and app path.
- Network policy and policy/security expansion.

## Phase 6 - Ingress Platform Migration

- Migrate ingress from `ingress-nginx` to `Cilium` Gateway API.
- Validate `/` and `/api` behavior for the visit app under the new ingress path.

## Notes

- `visit-processor` autoscaling and load generation are already implemented baselines.
- The remaining work is mostly hardening, resilience, and migration scope.
