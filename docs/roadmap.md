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
│ Status: COMPLETE (Done)  │ Status: COMPLETE (Done)  │ Status: IN PROGRESS        │
├──────────────────────────┼──────────────────────────┼────────────────────────────┤
│ Phase 4: Resilience & Sec│ Phase 5: RKE2 Migration  │ Phase 6: Gateway API Edge  │
│ Status: IN PROGRESS      │ Status: COMPLETE (Done)  │ Status: COMPLETE (Done)    │
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
  - OpenBao & External Secrets (`TASK-P3-02`): HA 3-pod Raft cluster on `ceph-block`, External Secrets Operator syncing to K8s Secrets, and automated scheduled Raft snapshots to Ceph RGW `s3://openbao-backups/` with Ansible restore automation (auto-unseal dropped in favor of procedural Ansible unseal automation).
  - Kafka & Transactional DLQ (`TASK-P3-06`): Strimzi KRaft 3-node HA cluster on `ceph-block` with `visits.requested` topic (30 partitions), `visits.dead-letter` DLQ topic (30 partitions, 14-day retention), fast-path poison pill isolation, 3 explicit retries (4 total attempts) with linear stepped backoff, and isolated PostgreSQL database transactions (`sql.Tx`).
  - Observability: `kube-prometheus-stack`, Promtail, distributed Loki on Ceph S3, Prometheus Kafka Exporter, Prometheus Adapter with HPA on `kafka_consumergroup_lag_sum`.

- **Phase 4 (Resilience, Policy & Security)**:
  - Cilium East-West Network Isolation (`TASK-P4-02`): 2-Tier Defense-in-Depth network policies combining global platform guardrails (`CiliumClusterwideNetworkPolicy`) for CoreDNS, kubelet health probes, and AWS IMDSv1 protection with service-owned least-privilege micro-segmentation (`CiliumNetworkPolicy`) for Postgres, Kafka mTLS, OpenBao API/Raft, and visit microservices.

- **Phase 5 (RKE2 Migration)**:
  - Migrated cluster bootstrap and runtime from kubeadm to RKE2.
- **Phase 6 (Gateway API Edge Migration)**:
  - Migrated edge exposure from `ingress-nginx` to Cilium Gateway API on worker host port 30080 mapped to public AWS NLB.

---

## On-Premise Two-Tier Backup & Disaster Recovery Architecture (Concept)

This platform adheres to an on-premise, cloud-agnostic **Two-Tier (3-2-1) Backup Architecture**. Generic volume backup tools (like Velero) are intentionally omitted in favor of GitOps manifest reconciliation paired with application-consistent object store backups.

> [!NOTE]
> **Lab Architecture Boundary**: For this dev/showcase platform, the in-cluster Rook-Ceph RGW (`s3://`) serves as the central Tier 1 backup destination for all components (PostgreSQL WAL/Base, OpenBao Raft snapshots, Kafka archives, and etcd snapshots). In a multi-site production environment, Tier 2 off-cluster replication (RGW multi-site sync or off-site rclone/rsync to remote MinIO / cloud S3) is layered on top to prevent in-cluster storage single points of failure.

```
┌────────────────────────────────────────────────────────────────────────────────────────┐
│                               TIER 1: LOCAL CLUSTER BACKUPS                            │
├──────────────────────┬────────────────────────────┬────────────────────────────────────┤
│ Component            │ Storage Target             │ Backup Mechanism                   │
├──────────────────────┼────────────────────────────┼────────────────────────────────────┤
│ Cluster Manifests    │ Git Repository             │ Flux v2 GitOps Reconciliation      │
│ PostgreSQL           │ Ceph RGW s3://postgres-backups/ │ CloudNativePG / Barman (WAL + Base)│
│ OpenBao (Vault)      │ Ceph RGW s3://openbao-backups/  │ CronJob Raft Snapshot (.snap)      │
│ Kafka Topics / Events│ Ceph RGW s3://kafka/       │ S3 Archiving Connector / CronJob   │
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
│ TASK-P3-02   │ OpenBao Scheduled Raft Snapshots to Ceph RGW│ COMPLETE (Done)│
│ TASK-P3-04   │ OpenBao Dynamic Postgres Secrets Engine     │ Medium         │
│ TASK-P3-05   │ Kafka Listener Security (mTLS) & ACLs       │ COMPLETE (Done)│
│ TASK-P3-06   │ Dead Letter Queue (DLQ) for Visit Events    │ COMPLETE (Done)│
│ TASK-P3-07   │ Ceph OSD Failure Drill & S3 Retention Policy│ Low            │
│ TASK-P3-08   │ Kafka Event Stream Archiving to Ceph RGW S3 │ Medium         │
└──────────────┴─────────────────────────────────────────────┴────────────────┘
```

---

### `TASK-P3-02`: OpenBao Scheduled Raft Snapshots to Ceph RGW (Complete)

- **Objective**: Implement automated periodic Raft snapshots of OpenBao state with local on-prem object storage in Ceph RGW (`s3://openbao-backups/`).
- **Affected Files**:
  - `kubernetes/infrastructure/base/security/openbao/objectbucketclaim.yaml` (new)
  - `kubernetes/infrastructure/base/security/openbao/cronjob-snapshot.yaml` (new)
  - `kubernetes/infrastructure/base/security/openbao/kustomization.yaml`
  - `ansible/roles/openbao_bootstrap/tasks/main.yml`
  - `ansible/roles/openbao_bootstrap/defaults/main.yml`
  - `ansible/playbooks/openbao-restore.yml` (new)
  - `ansible/playbooks/verify.yml`
  - `.taskfiles/openbao.yml`
  - `.taskfiles/ansible.yml`
  - `docs/openbao-runbook.md`
- **Implementation & Architecture Details**:
  1. **Local Object Storage**: `ObjectBucketClaim` `openbao-backups` dynamically provisions an on-prem S3 bucket in Ceph RGW via `storageClassName: ceph-bucket`, automatically injecting S3 access keys and endpoint details into the `openbao` namespace.
  2. **Least-Privilege Auth**: Configured OpenBao Kubernetes auth role `openbao-snapshot` bound to `ServiceAccount/openbao-snapshot` with dedicated policy `raft-snapshot` (`path "sys/storage/raft/snapshot" { capabilities = ["read"] }`), isolating snapshotting privileges from KV secrets data.
  3. **Two-Stage Snapshot CronJob**: `CronJob/openbao-snapshot` runs every 6 hours (`0 */6 * * *`):
     - *Init Container* (`quay.io/openbao/openbao:2.5.5`): Authenticates via Kubernetes ServiceAccount and writes `bao operator raft snapshot save` to a shared `emptyDir`.
     - *Upload Container* (`public.ecr.aws/aws-cli/aws-cli:2.17.53`): Streams the snapshot to Ceph RGW S3 (`s3://openbao-backups/raft-snapshot-<timestamp>.snap` and `raft-snapshot-latest.snap`).
  4. **Automated Restore Automation**: Playbook `ansible/playbooks/openbao-restore.yml` and `task openbao:restore` downloads specified/latest snapshot from S3, copies to `openbao-0`, executes `bao operator raft snapshot restore -force`, handles unsealing across all cluster pods, and verifies Raft peer health and secret readability.
- **Acceptance Criteria**:
  - Snapshot CronJob runs and successfully saves `.snap` files to Ceph RGW `s3://openbao-backups/`.
  - Restore procedure tested and documented in `docs/openbao-runbook.md`.
  - OBC binding and CronJob verified in `ansible/playbooks/verify.yml`.

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

### `TASK-P3-05`: Kafka Listener Security (mTLS) & ACLs (Complete)

- **Objective**: Secure Strimzi Kafka cluster communications by enabling Mutual TLS (mTLS) client authentication on internal listener port 9093, disabling unauthenticated plaintext traffic, and establishing least-privilege topic and group access via `KafkaUser` ACLs.
- **Affected Files**:
  - `kubernetes/infrastructure/base/data-kafka/kafka.yaml`
  - `kubernetes/infrastructure/base/data-kafka/kafkauser-visit-apps.yaml` (new)
  - `kubernetes/infrastructure/base/data-kafka/clustersecretstore-k8s-kafka.yaml` (new)
  - `kubernetes/infrastructure/base/data-kafka/kustomization.yaml`
  - `kubernetes/infrastructure/base/observability-kafka/prometheus-kafka-exporter.yaml`
  - `apps/visit-demo/visit-gateway/main.go`
  - `apps/visit-demo/visit-gateway/main_test.go` (new)
  - `apps/visit-demo/visit-processor/main.go`
  - `apps/visit-demo/visit-processor/main_test.go`
  - `charts/visit-gateway/templates/deployment.yaml`
  - `charts/visit-gateway/values.yaml`
  - `charts/visit-processor/templates/deployment.yaml`
  - `charts/visit-processor/values.yaml`
  - `kubernetes/apps/base/visit-web/externalsecret-kafka.yaml` (new)
  - `kubernetes/apps/base/visit-processing/externalsecret-kafka.yaml` (new)
  - `ansible/playbooks/verify.yml`
  - `docs/kafka-runbook.md`
- **Implementation & Architecture Details**:
  1. **Listener & Authorization**: Internal listener on port 9093 configured with `authentication.type: tls` and `authorization.type: simple` (`allow.everyone.if.no.acl.found: false`). Unauthenticated port 9092 removed.
  2. **Least-Privilege `KafkaUser` ACLs**:
     - `visit-gateway`: `Write`, `Describe` on topic `visits.requested`; `Describe` on consumer group `visit-processor-v1` (for queue lag reporting).
     - `visit-processor`: `Read`, `Describe` on topic `visits.requested`; `Read`, `Write`, `Describe` on topic `visits.dead-letter`; `Read`, `Describe` on consumer group `visit-processor-v1`.
     - `kafka-exporter`: `Describe` on topics (`*`), consumer groups (`*`), and cluster (`kafka-cluster`).
  3. **Automated Cross-Namespace Secret Projection**: ESO `ClusterSecretStore` (`k8s-data-kafka`) projects Strimzi client TLS secrets into `visit-web` and `visit-processing`.
  4. **Dynamic In-Memory Cert Reload**: Go microservices utilize `crypto/tls.Config.GetClientCertificate` to dynamically reload updated certificates from disk on handshake/reconnect without requiring pod restarts.
- **Acceptance Criteria**:
  - Unauthenticated connections to Kafka are rejected.
  - `visit-gateway` produces to `visits.requested` under mTLS and is denied unauthorized topics/reads.
  - `visit-processor` consumes with group `visit-processor-v1` and routes failures to `visits.dead-letter` under mTLS.
  - Automated verification passes in `ansible/playbooks/verify.yml`.


---

### `TASK-P3-06`: Dead Letter Queue (DLQ) for Visit Event Processing (Complete)

- **Objective**: Implement dead letter queueing in Kafka and transactional error handling in `visit-processor` so unprocessable/poison messages and exhausted database failures are preserved in a dedicated `visits.dead-letter` topic without stalling the consumer group lag.
- **Affected Files**:
  - `kubernetes/infrastructure/base/data-kafka/topic-visits-dlq.yaml` (new)
  - `kubernetes/infrastructure/base/data-kafka/kustomization.yaml`
  - `apps/visit-demo/visit-processor/main.go`
  - `apps/visit-demo/visit-processor/main_test.go` (new)
  - `charts/visit-processor/values.yaml`
  - `charts/visit-processor/templates/deployment.yaml`
  - `ansible/playbooks/verify.yml`
  - `docs/kafka-runbook.md`
  - `docs/visit-demo-runbook.md`
- **Implementation & Architecture Details**:
  1. **Topic Definition**: Strimzi `KafkaTopic` `visits.dead-letter` in `data-kafka` namespace configured with 30 partitions, 3 replicas, `min.insync.replicas: 2`, and 14 days retention (`retention.ms: 1209600000`).
  2. **Database Transactions (`sql.Tx`)**: Every PostgreSQL insertion executes inside an explicit database transaction (`db.BeginTx`) with `sql.LevelReadCommitted` isolation and automatic rollback on error.
  3. **3-Retry Schedule (4 Total Attempts)**: Transient database errors execute 1 initial attempt + 3 retries (4 total attempts) with linear stepped backoff (`RETRY_BACKOFF_MS * attempt`). If all 4 attempts fail, the event is routed to `visits.dead-letter` with `attempt_count: 4` and category `database_failure`.
  4. **Poison Pill Fast-Path**: Corrupt or malformed non-JSON payloads are intercepted during validation and immediately routed to `visits.dead-letter` with `attempt_count: 1` and category `corrupt_payload` without wasting database retries.
  5. **Forensic DLQ Envelope**: Failed events are wrapped in a structured JSON payload containing original topic, partition, offset, key, raw payload, error message, error category, failure timestamp, and attempt count.
  6. **Synchronous Delivery Guarantee**: Offsets on `visits.requested` are committed only after the DLQ producer receives full replica acknowledgment (`RequiredAcks: RequireAll`).
- **Acceptance Criteria**:
  - Malformed non-JSON payloads and exhausted database retry messages are safely published to `visits.dead-letter`.
  - Consumer group offset on `visits.requested` commits cleanly, preventing lag stalls or infinite retry loops.
  - Strimzi validates `visits.dead-letter` topic health in `ansible/playbooks/verify.yml`.


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

### `TASK-P3-08`: Kafka Event Stream Archiving to Ceph RGW S3

- **Objective**: Establish long-term event archiving by exporting committed Kafka topic records (`visits.requested`, `visits.dead-letter`) into Ceph RGW object storage (`s3://kafka-archive/`) to decouple retention from local NVMe OSD disk capacity.
- **Affected Files**:
  - `kubernetes/infrastructure/base/data-kafka/objectbucketclaim.yaml` (new)
  - `kubernetes/infrastructure/base/data-kafka/cronjob-topic-archive.yaml` (new)
  - `kubernetes/infrastructure/base/data-kafka/kustomization.yaml`
  - `docs/kafka-runbook.md`
- **Implementation Steps**:
  1. Define an `ObjectBucketClaim` `kafka-archive` in the `data-kafka` namespace to dynamically provision an S3 bucket in Ceph RGW.
  2. Deploy a lightweight batch consumer CronJob / archiver container connecting to Kafka under mTLS and streaming batch topic segments to `s3://kafka-archive/`.
  3. Document topic replay / recovery procedures from S3 archives in `docs/kafka-runbook.md`.
- **Acceptance Criteria**:
  * Topic messages are periodically archived to Ceph RGW `s3://kafka-archive/`.
  * Archival process operates cleanly without degrading real-time consumer group lag.

---

# Phase 4 - Resilience, Policy & Security Backlog

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            PHASE 4 TASKS                                    │
├──────────────┬─────────────────────────────────────────────┬────────────────┤
│ ID           │ Task Title                                  │ Priority       │
├──────────────┼─────────────────────────────────────────────┼────────────────┤
│ TASK-P4-02   │ Cilium Network Policies (East-West Isolation│ COMPLETE (Done)│
│ TASK-P4-03   │ Pod Security Standards & Kyverno Enforce    │ High           │
│ TASK-P4-05   │ Grafana Platform Dashboards Coverage        │ Medium         │
│ TASK-P4-06   │ RKE2 etcd Automated Snapshots & Retention   │ Medium         │
│ TASK-P4-07   │ Automated Backup & PITR Restore Drill       │ Medium         │
└──────────────┴─────────────────────────────────────────────┴────────────────┘
```

---

### `TASK-P4-02`: Cilium Network Policies (East-West Isolation) (Complete)

- **Objective**: Implement least-privilege network segmentation across all namespaces using a 2-Tier Defense-in-Depth model with `CiliumClusterwideNetworkPolicy` and namespaced `CiliumNetworkPolicy`.
- **Affected Files**:
  - `kubernetes/infrastructure/base/security/policies/cluster-guardrails.yaml` (new)
  - `kubernetes/infrastructure/base/security/kustomization.yaml`
  - `kubernetes/infrastructure/base/data-postgres/cilium-network-policy.yaml` (new)
  - `kubernetes/infrastructure/base/data-postgres/kustomization.yaml`
  - `kubernetes/infrastructure/base/data-kafka/cilium-network-policy.yaml` (new)
  - `kubernetes/infrastructure/base/data-kafka/kustomization.yaml`
  - `kubernetes/infrastructure/base/security/openbao/cilium-network-policy.yaml` (new)
  - `kubernetes/infrastructure/base/security/openbao/kustomization.yaml`
  - `kubernetes/apps/base/visit-web/cilium-network-policy.yaml` (new)
  - `kubernetes/apps/base/visit-web/kustomization.yaml`
  - `kubernetes/apps/base/visit-processing/cilium-network-policy.yaml` (new)
  - `kubernetes/apps/base/visit-processing/kustomization.yaml`
  - `kubernetes/apps/base/visit-loadgen/cilium-network-policy.yaml` (new)
  - `kubernetes/apps/base/visit-loadgen/kustomization.yaml`
  - `ansible/playbooks/verify.yml`
  - `docs/gateway-runbook.md`
- **Implementation & Architecture Details**:
  1. **Tier 1 (Global Platform Guardrails - CCNP)**: Centralized `platform-guardrails` in `infrastructure-security` applies cluster-wide to all tenant namespaces (`data-postgres`, `data-kafka`, `openbao`, `visit-web`, `visit-processing`, `visit-loadgen`). Enables CoreDNS egress (UDP/TCP 53), Kubelet health/readiness probes and port-forwarding (`reserved:host`), Prometheus scraping from `monitoring`, and Kubernetes API server token reviews/leases (`toEntities: [kube-apiserver]`). Blocks cloud provider instance metadata (`169.254.169.254/32` IMDSv1) via `egressDeny`.
  2. **Tier 2 (Service-Owned East-West Isolation - CNP)**:
     - `data-postgres`: Port 5432 ingress restricted to `visit-gateway` and `visit-processor` pods + peer replication; egress restricted to Ceph RGW (`rook-ceph:80`) for Barman WAL and daily base backups.
     - `data-kafka`: Port 9093 mTLS ingress restricted to `visit-gateway`, `visit-processor`, and `prometheus-kafka-exporter`; KRaft peer ports 9090/9091 restricted to internal brokers and Strimzi operator.
     - `openbao`: Port 8200 API ingress allowed from Envoy Gateway (`reserved:ingress`), ESO (`external-secrets`), snapshot CronJob, and host unseal automation; port 8201 Raft restricted to peers; snapshot job egress to Ceph RGW (`rook-ceph:80`).
     - `visit-web`: Port 3000/80 ingress to `visit-ui` from Envoy; port 8080 API ingress to `visit-gateway` from Envoy, `visit-ui`, and `visit-loadgen`; egress strictly limited to Postgres (5432) and Kafka (9093).
     - `visit-processing`: Consumer-only service; egress restricted strictly to Kafka (9093) and Postgres (5432).
     - `visit-loadgen`: Synthetic traffic generator; egress restricted strictly to `visit-gateway` (8080).
  3. **Automated Verification**: `ansible/playbooks/verify.yml` verifies that all CCNP and CNP resources are active, asserts that unauthorized cross-namespace connection attempts (e.g. from `rook-ceph` to `postgres:5432`) are dropped by Cilium eBPF, and validates that end-to-end visit demo event processing and counting operate cleanly.
- **Acceptance Criteria**:
  - Unauthorized cross-namespace traffic is dropped by Cilium eBPF.
  - End-to-end visit demo path (`visit-ui` -> `visit-gateway` -> Kafka -> `visit-processor` -> Postgres) continues to operate cleanly.
  - Automated verification passes in `ansible/playbooks/verify.yml`.

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

### `TASK-P4-06`: RKE2 etcd Automated Snapshots & Retention

- **Objective**: Configure automated scheduled etcd snapshots with retention policy directly in RKE2 server configuration.
- **Affected Files**:
  - `ansible/roles/rke2_control_plane/tasks/main.yml`
  - `ansible/roles/rke2_control_plane/defaults/main.yml`
  - `docs/phase1-infra-runbook.md`
- **Implementation Steps**:
  1. Configure `etcd-snapshot-schedule-cron: "0 * * * *"` and `etcd-snapshot-retention: 24` in RKE2 server configuration template (optional `etcd-s3` target pointed to Ceph RGW).
  2. Update Ansible control-plane role to apply the configuration.
- **Acceptance Criteria**:
  - RKE2 control plane automatically creates hourly snapshots and prunes snapshots beyond the retention window.
  - `rke2 etcd-snapshot list` displays available local snapshots.

---

### `TASK-P4-07`: Automated Backup & PITR Restore Verification Drill

- **Objective**: Automate disaster recovery validation by providing a non-destructive restore drill that tests Point-In-Time Recovery (PITR) against the S3 Barman object store.
- **Affected Files**:
  - `ansible/playbooks/restore-drill.yml` (new)
  - `kubernetes/infrastructure/base/data-postgres/cluster-restore-test.yaml` (new template)
  - `Taskfile.yml`
  - `docs/postgres-runbook.md`
- **Implementation Steps**:
  1. Write an Ansible playbook / Taskfile helper that reads the latest base backup metadata from Ceph RGW `s3://postgres-backups/`.
  2. Deploy a transient `postgres-restored` `Cluster` custom resource in `data-postgres` with `spec.bootstrap.recovery`.
  3. Wait for `postgres-restored` to reach `Ready` and execute SQL assertions comparing record counts in the `visits` table against the live cluster.
  4. Tear down the test restore cluster cleanly.
- **Acceptance Criteria**:
  - Restore drill executes end-to-end without manual intervention.
  - Asserts database consistency and logs backup age and recovery time.

---

# Out of Scope / Dropped Tasks

### `TASK-P6-01`: TLS & Edge Certificate Automation (`cert-manager`) (Dropped)

- **Status**: DROPPED / OUT OF SCOPE
- **Rationale**: Per [Agents.md](file:///home/martin/Dokumente/git/gitops-example/Agents.md), TLS and certificate management are explicitly out of scope for this AWS lab phase (`cert-manager` is deferred; ingress exposure is provided over HTTP via Cilium Gateway API on host port 30080 mapped to the AWS NLB). Implementing public ACME automation against internal `.gitops.local` / `nip.io` domains introduces external DNS delegation overhead with no architectural benefit for the demo workloads.

