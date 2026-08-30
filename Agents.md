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
- Kubernetes: RKE2
- Runtime: containerd
- CNI: Cilium (fallback Calico if needed)
- GitOps: Flux
- CI/CD: GitHub Actions
- Edge: Cilium Gateway API
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
   - OS packages, hardening baseline, and runtime prerequisites
3. Initialize cluster with RKE2:
   - 3 control planes, 3 workers
   - Install CNI and validate cluster health

Exit criteria:
- All six nodes joined and Ready
- Control plane remains healthy after one-node reboot test

### Phase 2 - GitOps + Platform Services + Sample Workloads
1. Bootstrap Flux and connect to repository Git source
2. Deploy platform services via Helm (through Flux):
   - Cilium Gateway API
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

### Phase 3 - Stateful & Secrets Hardening
Current state:
1. OpenBao baseline implemented (3 pods, Raft on Ceph, ESO synced; unsealing automated via Ansible).
2. Kafka baseline implemented with Strimzi in KRaft mode (3 brokers on Ceph), Transactional Dead Letter Queue (`TASK-P3-06` completed with `visits.dead-letter`, fast-path poison isolation, `sql.Tx` DB transactions, and 3-retry schedule), and Mutual TLS Listener Security & ACLs (`TASK-P3-05` completed with X.509 mTLS on port 9093, least-privilege `KafkaUser` ACLs, and ESO automated secret projection).
3. Ceph baseline implemented with `ceph-block` as default StorageClass.
4. Postgres implemented with CloudNativePG (3 instances on Ceph) + continuous WAL archiving and daily base backups to Ceph RGW (`TASK-P3-01` completed).
5. Distributed Loki on Ceph S3 RGW and custom metric autoscaling on Kafka lag.

Phase 3 hardening backlog (see `docs/roadmap.md` for task specs):
- `TASK-P3-02`: OpenBao scheduled Raft snapshots to Ceph RGW S3.
- `TASK-P3-04`: OpenBao dynamic PostgreSQL secrets engine and rotation.
- `TASK-P3-07`: Ceph OSD failure drills and Loki S3 retention lifecycle rules.


Exit criteria per tool:

- Install is reproducible from code.
- Basic operations and failure scenarios validated.
- Operational runbooks updated with disaster recovery steps.

### Phase 4 - Resilience, Policy & Security
Phase 4 backlog (see `docs/roadmap.md` for task specs):
1. **Backups & DR (Ceph-Centric)**:
   - `TASK-P4-06`: RKE2 etcd automated snapshots retention and Ceph/local rotation.
2. **Policy & Security**:
   - `TASK-P4-02`: CiliumNetworkPolicy least-privilege east-west isolation.
   - `TASK-P4-03`: Pod Security Standards (`baseline`/`restricted`) & Kyverno `Enforce` mode.
3. **Observability & Maintenance**:
   - `TASK-P4-04`: Renovate automation for automated dependency and chart updates.
   - `TASK-P4-05`: Grafana dashboard expansion (Ceph, Postgres, Kafka, Gateway API).

Exit criteria:
- Policy violations are blocked in admission.
- East-west network policies isolate data platform components without breaking app flow.
- Backup and recovery procedures validated for Ceph-backed stateful services.

### Phase 5 - RKE2 Migration (Implemented)
1. Migrated the cluster runtime and control plane to RKE2.
2. Revalidated bootstrap, CNI, observability, storage, and workloads on RKE2.

Exit criteria:
- Cluster bootstrap is reproducible on RKE2.
- Core add-ons and workloads reconcile cleanly.

### Phase 6 - Gateway Platform Migration (Implemented)
1. Migrated edge exposure from ingress-nginx to Cilium Gateway API on port 30080.
2. Revalidated `/` and `/api` behavior for the visit app.

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

1. `pipeline:init_cluster` - infrastructure, inventory, host prep, RKE2, Cilium
2. `ansible:core` - Flux, core operators, Ceph
3. `ansible:openbao` - OpenBao deploy/init/unseal/auth/seed
4. `pipeline:verify` - full post-deploy verification

Use `task pipeline:main` for the full ordered chain.

## Active Follow-ups

- Keep docs aligned with the implemented stack; see `docs/roadmap.md` for the prioritized task backlog.
- Maintain and extend post-deploy verification automation (`pipeline:verify`) before adding more platform components.
- Execute Phase 3 and Phase 4 tasks from `docs/roadmap.md` sequentially.

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
