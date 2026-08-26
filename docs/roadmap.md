# GitOps Platform Roadmap & Task Backlog

This document is the canonical backlog of implemented baselines and remaining technical tasks.
Each future task is structured with explicit objectives, affected files, implementation steps, and acceptance criteria so a single prompt can grab, execute, and verify it independently.

---

## Current Status Overview

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│                            STATUS SUMMARY BY PHASE                               │
├──────────────────────────┬──────────────────────────┬────────────────────────────┤
│ Phase 1: Infra & RKE2    │ Phase 2: GitOps & Edge   │ Phase 3: Stateful Services │
│ Status: COMPLETE (Done)  │ Status: COMPLETE (Done)  │ Status: BASELINE + WAL DONE│
├──────────────────────────┼──────────────────────────┼────────────────────────────┤
│ Phase 4: Resilience & Sec│ Phase 5: RKE2 Migration  │ Phase 6: Gateway API Edge  │
│ Status: PENDING / OPEN   │ Status: COMPLETE (Done)  │ Status: COMPLETE (Done)    │
└──────────────────────────┴──────────────────────────┴────────────────────────────┘
```

---

## Implemented Work (Done)

- **Phase 1 (Infrastructure & Cluster Foundation)**:
  - OpenTofu: VPC with 3 private node subnets, 1 public NAT subnet, security groups, IAM instance profile for SSM, and 6 EC2 instances (`t3.large`).
  - Ansible: Host OS tuning, kernel modules (`overlay`, `br_netfilter`), sysctl, swap disabled, and SSM inventory generation.
  - RKE2: 3-node HA control plane (embedded etcd) + 3 worker nodes with bundled Cilium CNI.
  - Remote state: OpenTofu remote S3 backend with DynamoDB locking.
- **Phase 2 (GitOps & Workload Delivery)**:
  - Flux v2: Staged Kustomization architecture with dependency tracking (`dependsOn`).
  - GitHub Actions: CI checks and image build/push to GHCR.
  - Flux Image Automation: `ImageRepository`, `ImagePolicy`, and `ImageUpdateAutomation` committing image tags back to Git.
  - Visit Demo Application: `visit-ui` (React Router SSR), `visit-gateway` (Go API), `visit-processor` (Go Kafka consumer/PG writer), `visit-loadgen` (Go load generator).
- **Phase 3 Baselines & Hardening (Stateful Services & Observability)**:
  - Ceph / Rook: 3 worker NVMe OSDs (`/dev/nvme1n1`), `ceph-block` default StorageClass, and Loki RGW S3 object store.
  - Postgres (`TASK-P3-01`): CloudNativePG 3-instance HA cluster on `ceph-block` with continuous WAL archiving and daily base backups (`ScheduledBackup`) to Ceph RGW `s3://postgres-backups/`.
  - Kafka: Strimzi KRaft 3-node HA cluster on `ceph-block` with `visits.requested` topic (30 partitions).
  - OpenBao & External Secrets: HA 3-pod Raft cluster on `ceph-block`, External Secrets Operator syncing to K8s Secrets (auto-unseal dropped in favor of procedural Ansible unseal automation).
  - Observability: `kube-prometheus-stack`, Promtail, distributed Loki on Ceph S3, Prometheus Kafka Exporter, Prometheus Adapter with HPA on `kafka_consumergroup_lag_sum`.

- **Phase 5 (RKE2 Migration)**:
  - Migrated cluster bootstrap and runtime from kubeadm to RKE2.
- **Phase 6 (Gateway API Edge Migration)**:
  - Migrated edge exposure from `ingress-nginx` to Cilium Gateway API on worker host port 30080 mapped to public AWS NLB.

---

## On-Premise Two-Tier Backup & Disaster Recovery Architecture (Concept)

This platform adheres to an on-premise, cloud-agnostic **Two-Tier (3-2-1) Backup Architecture**. Generic volume backup tools (like Velero) are intentionally omitted in favor of GitOps manifest reconciliation paired with application-consistent object store backups.

```
┌────────────────────────────────────────────────────────────────────────────────────────┐
│                               TIER 1: LOCAL CLUSTER BACKUPS                            │
├──────────────────────┬────────────────────────────┬────────────────────────────────────┤
│ Component            │ Storage Target             │ Backup Mechanism                   │
├──────────────────────┼────────────────────────────┼────────────────────────────────────┤
│ Cluster Manifests    │ Git Repository             │ Flux v2 GitOps Reconciliation      │
│ PostgreSQL           │ Ceph RGW s3://postgres/    │ CloudNativePG / Barman (WAL + Base)│
│ OpenBao (Vault)      │ Ceph RGW s3://openbao/     │ CronJob Raft Snapshot (.snap)      │
│ Kafka Topics / Events│ Ceph RGW s3://kafka/       │ S3 Connector / Topic Archiving     │
│ RKE2 Control Plane   │ Host / Ceph RGW s3://etcd/ │ Native RKE2 etcd Automated Snapshot│
└──────────────────────┴────────────────────────────┴────────────────────────────────────┘
                                        │
                                        ▼
┌────────────────────────────────────────────────────────────────────────────────────────┐
│                        TIER 2: OFF-CLUSTER DISASTER RECOVERY (CONCEPT)                 │
├────────────────────────────────────────────────────────────────────────────────────────┤
│ 1. Ceph RGW Multi-Site Sync: Native active-passive S3 bucket replication to secondary  │
│    datacenter or off-site MinIO / object storage cluster.                              │
│ 2. Off-Site Storage Sync: Automated rclone / rsync cron from secure backup bastion to  │
│    remote NAS, tape archive, or cold offsite target.                                   │
│ 3. Ceph RBD Mirroring: Asynchronous block-level pool replication for raw persistent   │
│    volumes via rbd-mirror daemon.                                                      │
└────────────────────────────────────────────────────────────────────────────────────────┘
```

---

# Phase 3 - Stateful & Secrets Hardening Backlog

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            PHASE 3 TASKS                                    │
├──────────────┬─────────────────────────────────────────────┬────────────────┤
│ ID           │ Task Title                                  │ Priority       │
├──────────────┼─────────────────────────────────────────────┼────────────────┤
│ TASK-P3-02   │ OpenBao Scheduled Raft Snapshots to Ceph RGW│ High           │
│ TASK-P3-04   │ OpenBao Dynamic Postgres Secrets Engine     │ Medium         │
│ TASK-P3-05   │ Kafka Listener Security (mTLS/SASL) & ACLs  │ Medium         │
│ TASK-P3-06   │ Dead Letter Queue (DLQ) for Visit Events    │ Medium         │
│ TASK-P3-07   │ Ceph OSD Failure Drill & S3 Retention Policy│ Low            │
└──────────────┴─────────────────────────────────────────────┴────────────────┘
```

---

### `TASK-P3-02`: OpenBao Scheduled Raft Snapshots to Ceph RGW

- **Objective**: Implement automated periodic Raft snapshots of OpenBao state with local on-prem object storage in Ceph RGW (`s3://openbao-backups/`).
- **Affected Files**:
  - `kubernetes/infrastructure/base/security/openbao/objectbucketclaim.yaml` (new)
  - `kubernetes/infrastructure/base/security/openbao/cronjob-snapshot.yaml` (new)
  - `kubernetes/infrastructure/base/security/openbao/kustomization.yaml`
  - `ansible/playbooks/openbao-restore.yml` (new)
  - `docs/openbao-runbook.md`
- **Implementation Steps**:
  1. Create an `ObjectBucketClaim` in the `openbao` namespace for `openbao-backups` using the local Ceph RGW StorageClass.
  2. Create a `CronJob` in `openbao` that executes `bao operator raft snapshot save`, using S3 credentials from the OBC Secret to stream the `.snap` file to `http://rook-ceph-rgw-ceph-objectstore.rook-ceph.svc:80`.
  3. Create an Ansible restore playbook / runbook procedure to restore snapshots into a fresh OpenBao Raft cluster (`bao operator raft snapshot restore`).
- **Acceptance Criteria**:
  - Snapshot CronJob runs and successfully saves `.snap` files to Ceph RGW `s3://openbao-backups/`.
  - Restore procedure tested and documented in `docs/openbao-runbook.md`.

---

### `TASK-P3-04`: OpenBao Dynamic Postgres Secrets Engine

- **Objective**: Replace static Postgres credentials with OpenBao's PostgreSQL secrets engine to issue dynamic, short-lived database credentials for application consumers.
- **Affected Files**:
  - `ansible/roles/openbao_bootstrap/tasks/main.yml`
  - `kubernetes/infrastructure/base/data-postgres/externalsecret-app-user.yaml`
  - `kubernetes/apps/base/visit-web/externalsecret-web.yaml`
  - `kubernetes/apps/base/visit-processing/externalsecret-processing.yaml`
  - `docs/openbao-runbook.md`
- **Implementation Steps**:
  1. Configure OpenBao `database` secrets engine with Postgres plugin connection details.
  2. Define database roles for `visit-gateway` (read-write) and `visit-processor` (read-write) with configurable TTL (e.g., 1 hour) and automatic rotation.
  3. Update External Secrets or application deployment to consume dynamically issued DB credentials.
- **Acceptance Criteria**:
  - OpenBao dynamically provisions temporary Postgres users and revokes them upon TTL expiry.
  - `visit-gateway` and `visit-processor` connect cleanly with dynamic credentials.

---

### `TASK-P3-05`: Kafka Listener Security (mTLS/SASL) & ACLs

- **Objective**: Secure Strimzi Kafka cluster communications by enabling authentication on internal listeners and restricting topics via `KafkaUser` ACLs.
- **Affected Files**:
  - `kubernetes/infrastructure/base/data-kafka/kafka.yaml`
  - `kubernetes/infrastructure/base/data-kafka/kafkauser-visit-apps.yaml` (new)
  - `charts/visit-gateway/templates/deployment.yaml`
  - `charts/visit-processor/templates/deployment.yaml`
  - `docs/kafka-runbook.md`
- **Implementation Steps**:
  1. Configure TLS listener with TLS client authentication (mTLS) or SASL SCRAM-SHA-512 in `kafka.yaml`.
  2. Create `KafkaUser` custom resources for `visit-gateway` (producer only on `visits.requested`) and `visit-processor` (consumer only on `visits.requested`).
  3. Mount client certificates or SASL secrets in the application deployments.
- **Acceptance Criteria**:
  - Unauthenticated clients cannot produce or consume messages.
  - `visit-gateway` can produce to `visits.requested` but cannot read.
  - `visit-processor` can consume with consumer group `visit-processor-v1`.

---

### `TASK-P3-06`: Dead Letter Queue (DLQ) for Visit Event Processing

- **Objective**: Implement dead letter queueing in Kafka and `visit-processor` so unprocessable/poison messages are preserved for inspection rather than silently dropped.
- **Affected Files**:
  - `kubernetes/infrastructure/base/data-kafka/topic-visits-dlq.yaml` (new)
  - `apps/visit-demo/visit-processor/main.go`
  - `docs/kafka-runbook.md`
  - `docs/visit-demo-runbook.md`
- **Implementation Steps**:
  1. Define `KafkaTopic` for `visits.dead-letter`.
  2. In `visit-processor/main.go`, add error routing: if JSON decoding or database insertion fails after max retries, produce the payload and error metadata to `visits.dead-letter` before committing offset.
- **Acceptance Criteria**:
  - Invalid messages posted to `visits.requested` are safely routed to `visits.dead-letter` without stalling the processor consumer group.

---

### `TASK-P3-07`: Ceph OSD Failure Drill & S3 Retention Policy

- **Objective**: Hardening drill for Ceph storage and lifecycle management for Loki log chunk objects.
- **Affected Files**:
  - `kubernetes/infrastructure/base/data-ceph/cephobjectstore-loki.yaml`
  - `docs/ceph-runbook.md`
- **Implementation Steps**:
  1. Configure RGW bucket lifecycle rules on `loki` bucket (e.g. expire log chunks after 14 days).
  2. Document and test the step-by-step procedure for replacing a failed NVMe EBS OSD disk without data loss.
- **Acceptance Criteria**:
  - RGW bucket lifecycle policy active and verifiable via `s3cmd` / `aws s3api`.
  - OSD removal and re-addition procedure validated.

---

# Phase 4 - Resilience, Policy & Security Backlog

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            PHASE 4 TASKS                                    │
├──────────────┬─────────────────────────────────────────────┬────────────────┤
│ ID           │ Task Title                                  │ Priority       │
├──────────────┼─────────────────────────────────────────────┼────────────────┤
│ TASK-P4-02   │ Cilium Network Policies (East-West Isolation│ High           │
│ TASK-P4-03   │ Pod Security Standards & Kyverno Enforce    │ High           │
│ TASK-P4-04   │ Renovate Dependency Automation Setup        │ Medium         │
│ TASK-P4-05   │ Grafana Platform Dashboards Coverage        │ Medium         │
│ TASK-P4-06   │ RKE2 etcd Automated Snapshots Retention     │ Medium         │
└──────────────┴─────────────────────────────────────────────┴────────────────┘
```

---

### `TASK-P4-02`: Cilium Network Policies (East-West Isolation)

- **Objective**: Implement least-privilege network segmentation across all namespaces using `CiliumNetworkPolicy`.
- **Affected Files**:
  - `kubernetes/infrastructure/base/security/policies/cilium-network-policies.yaml` (new)
  - `kubernetes/infrastructure/base/security/kustomization.yaml`
  - `docs/gateway-runbook.md`
- **Implementation Steps**:
  1. Define default-deny ingress/egress policies for tenant namespaces (`data-postgres`, `data-kafka`, `openbao`, `visit-web`, `visit-processing`).
  2. Allow explicit ingress:
     - `data-postgres`: port 5432 only from `visit-gateway` and `visit-processor` pods.
     - `data-kafka`: port 9092/9093 only from `visit-gateway`, `visit-processor`, and `prometheus-kafka-exporter`.
     - `openbao`: port 8200 only from `external-secrets` operator and gateway route.
     - `visit-web`: HTTP only from Cilium Gateway proxy.
  3. Validate traffic flows with `cilium hubble observe`.
- **Acceptance Criteria**:
  - Unauthorized cross-namespace traffic is dropped by Cilium.
  - End-to-end visit demo path (`visit-ui` -> `visit-gateway` -> Kafka -> `visit-processor` -> Postgres) continues to operate cleanly.

---

### `TASK-P4-03`: Pod Security Standards & Kyverno Enforce Mode

- **Objective**: Enforce Kubernetes Pod Security Standards (`baseline` / `restricted`) and switch Kyverno policies to `Enforce` mode.
- **Affected Files**:
  - All `namespace.yaml` files under `kubernetes/apps/` and `kubernetes/infrastructure/`
  - `kubernetes/infrastructure/base/scheduling/policies/clusterpolicy-tiny-pod-requests.yaml`
  - `kubernetes/apps/overlays/dev/policies/clusterpolicy-prefer-non-ceph-nodes.yaml`
  - `docs/kyverno-runbook.md`
- **Implementation Steps**:
  1. Label application namespaces (`visit-web`, `visit-processing`, `visit-loadgen`) with:
     - `pod-security.kubernetes.io/enforce: restricted`
     - `pod-security.kubernetes.io/enforce-version: latest`
  2. Label platform namespaces with `pod-security.kubernetes.io/enforce: baseline`.
  3. Switch Kyverno policies from `validationFailureAction: Audit` to `validationFailureAction: Enforce`.
  4. Ensure workload security contexts specify non-root user, read-only root filesystems, and drop all capabilities.
- **Acceptance Criteria**:
  - Non-compliant pods violating PSS or Kyverno policies are rejected by the admission webhook.
  - All existing platform and application workloads remain healthy.

---

### `TASK-P4-04`: Renovate Automated Dependency Updates

- **Objective**: Automate dependency updates across Helm charts, Docker base images, Go modules, npm packages, and GitHub Actions.
- **Affected Files**:
  - `renovate.json` (new)
  - `docs/github-actions-runbook.md`
- **Implementation Steps**:
  1. Add `renovate.json` configuring package managers (`helmv3`, `dockerfile`, `gomod`, `npm`, `github-actions`).
  2. Set up automerge rules for patch updates and schedule PR creation during designated maintenance windows.
  3. Pin dependency version constraints.
- **Acceptance Criteria**:
  - Renovate scans the repository and opens automated PRs with changelogs for outdated Helm charts and packages.

---

### `TASK-P4-05`: Grafana Platform Dashboards Coverage

- **Objective**: Expand Grafana dashboard coverage to provide deep visibility into Ceph storage, CloudNativePG, Strimzi Kafka, and Gateway API traffic.
- **Affected Files**:
  - `kubernetes/infrastructure/base/observability/dashboards/` (new dashboard configmaps)
  - `kubernetes/infrastructure/base/observability/kustomization.yaml`
  - `docs/observability-runbook.md`
- **Implementation Steps**:
  1. Add dashboard ConfigMaps for:
     - Ceph Cluster & OSD Health (Rook metrics)
     - CloudNativePG Cluster Performance & Replication Lag
     - Strimzi Kafka Brokers, Topics & Consumer Group Lag
     - Cilium Gateway API Traffic, Latency & HTTP Error Rates
  2. Label ConfigMaps with `grafana_dashboard: "1"` for Grafana sidecar auto-discovery.
- **Acceptance Criteria**:
  - Dashboards automatically appear in Grafana under `http://grafana.gitops.local` without manual imports.

---

### `TASK-P4-06`: RKE2 etcd Automated Snapshots Retention

- **Objective**: Configure automated scheduled etcd snapshots with retention policy directly in RKE2 server configuration.
- **Affected Files**:
  - `ansible/roles/rke2_control_plane/templates/config.yaml.j2`
  - `ansible/roles/rke2_control_plane/tasks/main.yml`
  - `docs/phase1-infra-runbook.md`
- **Implementation Steps**:
  1. Configure `etcd-snapshot-schedule-cron: "0 * * * *"` and `etcd-snapshot-retention: 24` in RKE2 server configuration template (optional `etcd-s3` target pointed to Ceph RGW).
  2. Update Ansible control-plane role to apply the configuration.
- **Acceptance Criteria**:
  - RKE2 control plane automatically creates hourly snapshots and prunes snapshots beyond the retention window.
  - `rke2 etcd-snapshot list` displays available local snapshots.

---

# Future Platform Expansion (Phase 6+)

### `TASK-P6-01`: TLS & Edge Certificate Automation (`cert-manager`)

- **Objective**: Introduce `cert-manager` to automate Let's Encrypt certificates and configure HTTPS listeners on the Cilium Gateway.
- **Affected Files**:
  - `kubernetes/infrastructure/base/core-services/cert-manager/` (new)
  - `kubernetes/infrastructure/base/gateway/gateway.yaml`
  - `docs/gateway-runbook.md`
- **Implementation Steps**:
  1. Deploy `cert-manager` HelmRelease via Flux in `infrastructure-core`.
  2. Configure `ClusterIssuer` for Let's Encrypt (DNS-01 route53 or HTTP-01).
  3. Add HTTPS listener (`port: 443`, `protocol: HTTPS`) to `dev-gateway` referencing the TLS Secret.
- **Acceptance Criteria**:
  - Gateways serve valid TLS certificates; HTTP routes automatically redirect to HTTPS.

