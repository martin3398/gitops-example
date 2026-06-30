# Kafka Runbook (Strimzi, Dev)

This repository deploys a platform-owned Kafka baseline with Flux using Strimzi in KRaft mode.

## Components

- Strimzi operator: `kubernetes/platform/dev/core-services/operators/strimzi/`
- Kafka cluster manifests: `kubernetes/platform/dev/data-platform/services/kafka/`
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

## Validation

The standard post-deploy verification path is:

```bash
task pipeline:verify
```

It checks the Kafka custom resource, broker pods, `visits.requested` topic readiness, and the visit demo queue/count flow.

Use a workstation with cluster access:

```bash
kubectl -n data-kafka get pods
kubectl -n data-kafka get kafka
kubectl -n data-kafka get kafkanodepools
kubectl -n data-kafka get pvc -o wide
```

Check Kafka readiness details:

```bash
kubectl -n data-kafka describe kafka kafka
```

The Flux `platform-data-kafka` stage has an explicit health check for the `Kafka/data-kafka/kafka` custom resource. If Strimzi rejects the Kafka version or the cluster is not ready, the stage should not be treated as fully healthy.

If Kafka stays `NotReady`, verify that `spec.kafka.version` is supported by the pinned Strimzi operator chart version.

## Next Iteration (Self-Service)

- Add GitOps templates for `KafkaTopic` and `KafkaUser`
- Add per-team namespace/network policy boundaries
- Add broker and consumer lag alerting
