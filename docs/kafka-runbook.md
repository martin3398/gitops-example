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
- Internal listeners only:
  - plaintext: `kafka-kafka-bootstrap.data-kafka.svc:9092`
  - TLS: `kafka-kafka-bootstrap.data-kafka.svc:9093`
- Entity Operator enabled (Topic Operator + User Operator)

## Storage

- Persistent volume claims use `ceph-block` in this dev baseline.

## Topics & Dead Letter Queue (DLQ)

- `visits.requested`: main ingress topic for raw visit requests (30 partitions, 3 replicas).
- `visits.dead-letter`: dead letter queue topic for poison pills, malformed payloads, and failed database retries (30 partitions, 3 replicas, 14-day retention).

### Inspecting DLQ Messages

To read and inspect poison or failed messages diverted to `visits.dead-letter`:

```bash
# Using a temporary debug container with kcat:
kubectl -n data-kafka run kcat-dlq --rm -it --image=edenhill/kcat:1.7.1 -- \
  -b kafka-kafka-bootstrap.data-kafka.svc:9092 -t visits.dead-letter -C -e -o beginning -u

# Using Strimzi kafka-console-consumer:
kubectl -n data-kafka exec kafka-brokers-0 -c kafka -- \
  bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic visits.dead-letter \
  --from-beginning
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

It checks the Kafka custom resource, broker pods, `visits.requested` and `visits.dead-letter` topic readiness, and the visit demo queue/count flow.

Use a workstation with cluster access:

```bash
kubectl -n data-kafka get pods
kubectl -n data-kafka get kafka
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
- **Listener Security & ACLs (`TASK-P3-05`)**: Enable client mTLS/SASL authentication and define `KafkaUser` resources with least-privilege ACLs for producer/consumer roles.
- **Network Policy Isolation (`TASK-P4-02`)**: Restrict Kafka ports 9092/9093 access strictly to `visit-gateway`, `visit-processor`, and `prometheus-kafka-exporter`.
