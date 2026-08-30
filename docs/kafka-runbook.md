# Kafka Runbook (Strimzi, Dev)

This repository deploys a platform-owned Kafka baseline with Flux using Strimzi in KRaft mode.

## Components

- Strimzi operator: `kubernetes/infrastructure/base/core-services/operators/strimzi/`
- Kafka cluster manifests: `kubernetes/infrastructure/base/data-kafka/`
- Namespace: `data-kafka`

## Current Baseline

- 3 broker/controller nodes (single node pool)
- KRaft enabled (no ZooKeeper)
- Strimzi operator chart is pinned to `1.1.0`
- Kafka broker version is pinned to `4.3.0`, which is supported by Strimzi chart `1.1.0`
- Listener: TLS `kafka-kafka-bootstrap.data-kafka.svc:9093` with Mutual TLS (mTLS) client authentication (`authentication: { type: tls }`). Plaintext port 9092 is disabled.
- Authorization: `type: simple` (Kafka ACLs) with `allow.everyone.if.no.acl.found: false`.
- Entity Operator enabled (Topic Operator + User Operator)

## Storage

- Persistent volume claims use `ceph-block` in this dev baseline.

## Security, Access Control (ACLs) & mTLS

All cluster communications require valid X.509 client certificates issued by the Strimzi Clients CA (`kafka-clients-ca`). Workloads are assigned least-privilege ACLs via Strimzi `KafkaUser` custom resources:

| KafkaUser | Authentication | Target Namespace | ACL Permissions | Purpose |
| :--- | :--- | :--- | :--- | :--- |
| `visit-gateway` | Mutual TLS (X.509) | `visit-web` | `Topic: visits.requested` [Write, Describe]<br>`Group: visit-processor-v1` [Describe] | Ingress event publisher + queue lag reader |
| `visit-processor` | Mutual TLS (X.509) | `visit-processing` | `Topic: visits.requested` [Read, Describe]<br>`Topic: visits.dead-letter` [Write, Describe]<br>`Group: visit-processor-v1` [Read, Describe] | Event processor + DLQ routing |
| `kafka-exporter` | Mutual TLS (X.509) | `data-kafka` | `Topic: *` [Describe]<br>`Group: *` [Describe]<br>`Cluster: kafka-cluster` [Describe] | Prometheus metrics collection |

### Cross-Namespace Certificate Propagation
1. Strimzi User Operator creates client TLS Secrets in `data-kafka`.
2. External Secrets Operator (ESO) uses ClusterSecretStore `k8s-data-kafka` to project credentials into tenant namespaces:
   - `data-kafka/visit-gateway` -> `visit-web/visit-gateway-kafka-tls`
   - `data-kafka/visit-processor` -> `visit-processing/visit-processor-kafka-tls`
3. Workload pods mount certificates at `/etc/kafka/tls/` (`ca.crt`, `user.crt`, `user.key`).
4. Microservices dynamically reload certificates on TLS handshake via `crypto/tls.Config.GetClientCertificate`.

### Certificate Rotation & Lifecycle
- **Automatic Renewal**: Strimzi User Operator automatically renews client certificates 30 days before expiration (365-day default validity) without manual intervention.
- **On-Demand Rotation**: To immediately force certificate reissuance:
  ```bash
  kubectl -n data-kafka annotate kafkauser visit-gateway strimzi.io/force-renew="true"
  # Or delete the secret to force regeneration:
  kubectl -n data-kafka delete secret visit-gateway
  ```
- **CA Renewal**: Strimzi maintains dual-CA trust during Clients CA rotation until all client certs have rolled over.

## Topics & Dead Letter Queue (DLQ)

- `visits.requested`: main ingress topic for raw visit requests (30 partitions, 3 replicas).
- `visits.dead-letter`: dead letter queue topic for poison pills, malformed payloads, and failed database retries (30 partitions, 3 replicas, 14-day retention).

### Inspecting DLQ Messages with mTLS

To inspect DLQ messages using `kcat` with the `visit-processor` client certificate:

```bash
# Using a temporary debug container with kcat and mounted client/CA certificates:
kubectl -n data-kafka run kcat-dlq --rm -it --image=edenhill/kcat:1.7.1 \
  --overrides='{
    "spec": {
      "containers": [{
        "name": "kcat-dlq",
        "image": "edenhill/kcat:1.7.1",
        "volumeMounts": [
          {"name": "cluster-ca", "mountPath": "/etc/kafka/ca", "readOnly": true},
          {"name": "user-tls", "mountPath": "/etc/kafka/tls", "readOnly": true}
        ]
      }],
      "volumes": [
        {"name": "cluster-ca", "secret": {"secretName": "kafka-cluster-ca-cert"}},
        {"name": "user-tls", "secret": {"secretName": "visit-processor"}}
      ]
    }
  }' -- \
  -b kafka-kafka-bootstrap.data-kafka.svc:9093 \
  -X security.protocol=SSL \
  -X ssl.ca.location=/etc/kafka/ca/ca.crt \
  -X ssl.certificate.location=/etc/kafka/tls/user.crt \
  -X ssl.key.location=/etc/kafka/tls/user.key \
  -t visits.dead-letter -C -e -o beginning -u
```

DLQ messages contain a structured JSON envelope:
```json
{
  "original_topic": "visits.requested",
  "original_partition": 0,
  "original_offset": 123,
  "original_key": "visit-12345",
  "original_payload": "...",
  "error_message": "invalid JSON payload: ...",
  "error_category": "corrupt_payload",
  "failed_at": "2026-08-28T13:45:00Z",
  "attempt_count": 1
}
```

## Validation

The standard post-deploy verification path is:

```bash
task pipeline:verify
```

It checks the Kafka custom resource, broker pods, `visits.requested` and `visits.dead-letter` topic readiness, `KafkaUser` resources, and the visit demo queue/count flow.

Use a workstation with cluster access:

```bash
kubectl -n data-kafka get pods
kubectl -n data-kafka get kafka
kubectl -n data-kafka get kafkausers
kubectl -n data-kafka get kafkatopics
kubectl -n data-kafka get kafkanodepools
kubectl -n data-kafka get pvc -o wide
```

Check Kafka readiness details:

```bash
kubectl -n data-kafka describe kafka kafka
```

The Flux `infrastructure-data-kafka` stage has an explicit health check for the `Kafka/data-kafka/kafka` custom resource. If Strimzi rejects the Kafka version or the cluster is not ready, the stage should not be treated as fully healthy.

If Kafka stays `NotReady`, verify that `spec.kafka.version` is supported by the pinned Strimzi operator chart version.

## Notes & Roadmap Hardening Tasks

- **Dead Letter Queue (`TASK-P3-06`)**: Implemented (Dedicated `visits.dead-letter` topic and `visit-processor` poison message routing).
- **Listener Security & ACLs (`TASK-P3-05`)**: Implemented (Enforced mTLS client certificate authentication on port 9093, least-privilege `KafkaUser` ACLs, and ESO automated secret projection).
- **Network Policy Isolation (`TASK-P4-02`)**: Restrict Kafka port 9093 access strictly to `visit-gateway`, `visit-processor`, and `prometheus-kafka-exporter`.

