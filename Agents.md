# Agents.md

## Mission
Build and operate a reproducible Kubernetes GitOps example on low-cost AWS EC2.

## Current Objective
Implement a multi-phase learning platform with:
- 3 control plane nodes and 3 worker nodes
- Open-source tooling
- GitOps with Flux
- CI/CD with GitHub Actions

## Rethink / Scope Control
- Keep strict ownership boundaries:
  - OpenTofu/Terraform: cloud infrastructure only
  - Ansible: host provisioning + Kubernetes bootstrap only
  - Flux: all cluster applications and platform add-ons
- Introduce heavyweight stateful tools one by one (OpenBao, Kafka, Ceph) to keep troubleshooting tractable.
- Backup and policy/security controls are next-phase work now that the base platform and app delivery workflow are stable.
- Treat stateful systems as implemented baselines that still need operational hardening, especially backup/restore and credential rotation.

## Platform Decisions
- IaC: OpenTofu (Terraform-compatible)
- Config/Bootstrap: Ansible
- Kubernetes: kubeadm
- Runtime: containerd
- CNI: Cilium (fallback Calico if needed)
- GitOps: Flux
- CI/CD: GitHub Actions
- Ingress: ingress-nginx
- TLS/certificate management: out of scope for this AWS lab phase
- Monitoring: kube-prometheus-stack + Loki + Grafana
- Data platform baselines: Postgres (CloudNativePG) and Kafka (Strimzi)
- Secrets: OpenBao + External Secrets Operator

## Target Topology
- Kubernetes nodes on EC2:
  - 3 control plane nodes
  - 3 worker nodes
- One VPC, private subnets for nodes, restricted administrative ingress.

## Delivery Phases

### Phase 1 - Infrastructure + Kubernetes Foundation
1. Provision AWS base infrastructure with OpenTofu/Terraform:
   - VPC, subnets, security groups, IAM roles, 6 EC2 instances
2. Bootstrap hosts with Ansible:
   - OS packages, hardening baseline, containerd, kubeadm prerequisites
3. Initialize cluster with kubeadm:
   - 3 control planes, 3 workers
   - Install CNI and validate cluster health

Exit criteria:
- All six nodes joined and Ready
- Control plane remains healthy after one-node reboot test

### Phase 2 - GitOps + Platform Services + Sample Workloads
1. Bootstrap Flux and connect to repository Git source
2. Deploy platform services via Helm (through Flux):
   - ingress-nginx
   - cert-manager is deferred (HTTPS/certificates out of scope)
   - monitoring/logging stack
3. Deploy sample application stack:
   - simple frontend
   - 1-2 microservices
   - consume platform-provided data services
4. Add CI/CD pipelines:
   - test/build images
   - publish images
   - use Flux image automation to update GitOps manifests/values

Exit criteria:
- Git push triggers CI and Flux reconciliation
- End-to-end app path is reachable through ingress
- Observability dashboards show app and cluster metrics

### Phase 3 - Advanced Tooling
Current state:
1. OpenBao baseline implemented for platform/app secrets
2. Kafka baseline implemented with Strimzi in KRaft mode
3. Ceph baseline implemented with `ceph-block` as default StorageClass
Phase 3 hardening:
- OpenBao hardening: snapshots, restore, auto-unseal, dynamic DB credentials, rotation
- credential management hardening: stronger OpenBao init material handling, GHCR pull secrets, and Flux Git write credentials
- Ceph hardening: restore drills and stateful workload failure testing
- Traffic validation: reproducible load scenarios and queue/throughput observations
- Alerting and dashboards: queue depth and lag visibility for the visit path

Exit criteria per tool:
- Install is reproducible
- Basic operations validated
- Troubleshooting notes captured

### Phase 4 - Resilience, Backup, Policy & Security
1. Backups:
    - Velero for cluster resource backup/restore
    - Database-specific backup and restore tests
2. Policy/Security:
   - Kyverno (or OPA Gatekeeper)
   - Trivy image scanning in CI
   - baseline network policies
   - Pod security standards/admission configuration
3. Observability:
   - Grafana dashboard coverage for the current platform and app path
4. Update hygiene:
     - Renovate automation for dependency and workflow updates

Exit criteria:
- Restore drill passes
- Policy violations are detectable and actionable

### Phase 5 - RKE2 Migration / Regulatory Hardening
1. Migrate the cluster runtime to RKE2.
2. Revalidate bootstrap, CNI, observability, storage, and workloads on the new runtime.

Exit criteria:
- Cluster bootstrap is reproducible on RKE2.
- Core add-ons and workloads still reconcile cleanly.

### Phase 6 - Ingress Platform Migration
1. Migrate ingress from ingress-nginx to Cilium Gateway API.
2. Revalidate `/` and `/api` behavior for the visit app.

Exit criteria:
- App exposure works through the new Gateway API path.
- Visit app routing remains same-origin and functional.

## Operational Rules
- No long-lived manual drift in cluster resources; reconcile through Flux.
- Pin versions for Helm charts and critical platform components.
- Define resource requests/limits for all workloads.
- Document every major component with install, operate, and failure runbook notes.
- Include teardown steps for cost control.
- Agents may run safe local inspection and validation commands when needed.
- Destructive or cost-impacting infrastructure actions, especially `tofu apply`, `tofu destroy`, cluster rebuilds, and secret resets, require explicit human direction.
- OpenTofu state backend for this lab uses S3 with DynamoDB locking; avoid local-state-only workflows once backend is initialized.
- Infrastructure apply/destroy operations in CI remain manual-gated.

## Current Deployment Model

The full local deployment is staged:

1. `pipeline:init_cluster` - infrastructure, inventory, host prep, kubeadm, Cilium
2. `ansible:core` - Flux, core operators, Ceph
3. `ansible:openbao` - OpenBao deploy/init/unseal/auth/seed
4. `pipeline:verify` - full post-deploy verification

Use `task pipeline:main` for the full ordered chain.

## Active Follow-ups

- Keep docs aligned with the implemented stack; do not reintroduce Vault terminology unless a separate HashiCorp Vault lab is explicitly requested.
- Maintain and extend post-deploy verification automation (`pipeline:verify`) before adding more platform components.
- Execute Phase 4 hardening work: `Renovate`, failover drills, backup/restore, dashboards, and secret hardening.
- Prepare the RKE2 migration as a separate phase with a clear cutover and rollback path.
- Prepare the Gateway API ingress migration as a separate phase after cluster/runtime stability is proven.

## Suggested Repository Layout
- `infra/` OpenTofu/Terraform infrastructure code
- `ansible/` host and cluster bootstrap automation
- `kubernetes/clusters/` Flux bootstrap and Kustomizations
- `kubernetes/infrastructure/` platform components and overlays
- `kubernetes/apps/` frontend, microservices, Postgres
- `docs/` architecture, runbooks, troubleshooting, DR tests

## Definition of Done (Global)
- The environment can be created and destroyed from code.
- Flux continuously reconciles desired state from Git.
- CI/CD, observability, and stateful workloads operate as expected.
- Backup/restore and policy/security controls are tested and documented.
