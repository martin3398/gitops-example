# Kafka Runbook (Strimzi, Dev)

This repository deploys a platform-owned Kafka baseline with Flux using Strimzi in KRaft mode.

## Components

- Strimzi operator: `kubernetes/platform/dev/core-services/operators/strimzi/`
- Kafka cluster manifests: `kubernetes/platform/dev/data-platform/services/kafka/`
- Namespace: `data-kafka`

## Current Baseline

- 3 broker/controller nodes (single node pool)
- KRaft enabled (no ZooKeeper)
- Internal listeners only:
  - plaintext: `kafka-kafka-bootstrap.data-kafka.svc:9092`
  - TLS: `kafka-kafka-bootstrap.data-kafka.svc:9093`
- Entity Operator enabled (Topic Operator + User Operator)

## Storage

- Persistent volume claims use `local-path` in this dev baseline.
- Plan migration to Ceph-backed storage classes in the Ceph phase.

## Validation

Use a workstation with cluster access:

```bash
kubectl -n data-kafka get pods
kubectl -n data-kafka get kafka
kubectl -n data-kafka get kafkanodepools
```

Check Kafka readiness details:

```bash
kubectl -n data-kafka describe kafka kafka
```

## Next Iteration (Self-Service)

- Add GitOps templates for `KafkaTopic` and `KafkaUser`
- Add per-team namespace/network policy boundaries
- Add broker and consumer lag alerting
