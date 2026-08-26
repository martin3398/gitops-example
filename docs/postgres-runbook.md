# Postgres Runbook (CloudNativePG, Dev)

This runbook covers the Postgres baseline deployed by Flux.

## Components

- CloudNativePG operator: `kubernetes/infrastructure/base/core-services/operators/cloudnative-pg/`
- Postgres cluster manifests: `kubernetes/infrastructure/base/data-postgres/`
- Namespace: `data-postgres`

## Current Baseline

- 3-instance CloudNativePG cluster named `postgres`
- storage class: `ceph-block`
- volume size: `1Gi`
- application bootstrap secret: `app-user` from OpenBao via External Secrets
- app database: `app`
- app user: `app`
- S3 Barman Object Store: continuous WAL archiving and base backups to Ceph RGW `s3://postgres-backups/` (`http://rook-ceph-rgw-loki.rook-ceph.svc:80`)
- ScheduledBackup: daily base backup `postgres-daily-backup` running at `00:00 UTC`

The visit processor writes through the read-write service endpoint:

```text
postgres-rw.data-postgres.svc:5432
```

## Backup & WAL Archiving Architecture

1. **Credentials Lifecycle**: OpenBao seeds `dev/platform/postgres/s3` with deterministic access and secret keys during cluster bootstrap. External Secrets Operator synchronizes these into Secret `postgres-s3-credentials` in `data-postgres` and `rook-ceph`.
2. **Ceph Object Store User & Bucket**: `CephObjectStoreUser` `postgres` is created in `rook-ceph` bound to RGW store `loki`. Job `postgres-create-bucket` in `data-postgres` idempotently ensures bucket `postgres-backups` exists on Ceph RGW.
3. **WAL Archiving**: PostgreSQL continuously archives WAL segments using `spec.backup.barmanObjectStore` with gzip compression.
4. **Base Backups**: `ScheduledBackup` `postgres-daily-backup` executes daily base backups (with `immediate: true` for initial bootstrap).

## Validation

Use a workstation with cluster access:

```bash
kubectl -n flux-system get kustomizations,helmreleases
kubectl -n data-postgres get pods
kubectl -n data-postgres get cluster
kubectl -n data-postgres get secret app-user postgres-s3-credentials
kubectl -n data-postgres get scheduledbackups,backups
kubectl -n data-postgres get svc
task pipeline:verify
```

### Inspecting Backup and WAL Status

1. Check cluster backup and WAL archiving status:
```bash
kubectl -n data-postgres describe cluster postgres
```
Verify `First Status Backup` and `Last Successful WAL Archive` show recent timestamps.

2. Check backup custom resources:
```bash
kubectl -n data-postgres get backups -o wide
```

3. Inspect backup files in Ceph RGW via toolbox:
```bash
kubectl -n rook-ceph exec deploy/rook-ceph-tools -- radosgw-admin bucket list
kubectl -n rook-ceph exec deploy/rook-ceph-tools -- radosgw-admin bucket stats --bucket=postgres-backups
```

## Point-in-Time Recovery (PITR) & Restore Procedure

To restore a database cluster from the Barman S3 backup:

### 1. Full Restore to a New Cluster

Create a recovery cluster manifest (e.g. `cluster-restore.yaml`):

```yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: postgres-restored
  namespace: data-postgres
spec:
  instances: 3
  storage:
    size: 1Gi
    storageClass: ceph-block
  bootstrap:
    recovery:
      source: postgres
  externalClusters:
    - name: postgres
      barmanObjectStore:
        destinationPath: s3://postgres-backups/
        endpointURL: http://rook-ceph-rgw-loki.rook-ceph.svc:80
        s3Credentials:
          accessKeyId:
            name: postgres-s3-credentials
            key: AWS_ACCESS_KEY_ID
          secretAccessKey:
            name: postgres-s3-credentials
            key: AWS_SECRET_ACCESS_KEY
        wal:
          compression: gzip
```

### 2. Point-in-Time Recovery (PITR) Target

To restore to a specific point in time or target transaction ID, add `recoveryTarget` to `spec.bootstrap.recovery`:

```yaml
spec:
  bootstrap:
    recovery:
      source: postgres
      recoveryTarget:
        targetTime: "2026-08-26 12:00:00.000000+00"
```

Apply the restore manifest:
```bash
kubectl apply -f cluster-restore.yaml
kubectl -n data-postgres wait cluster/postgres-restored --for=condition=Ready --timeout=15m
```

## Troubleshooting

- If WAL archiving reports failures, verify the RGW endpoint (`http://rook-ceph-rgw-loki.rook-ceph.svc:80`) is reachable and Secret `postgres-s3-credentials` contains valid keys.
- If `ScheduledBackup` fails to start, verify the Job `postgres-create-bucket` succeeded and the `postgres-backups` bucket exists.
- If the application secret is missing, verify OpenBao and the External Secrets `ClusterSecretStore` are Ready.
- If PVCs stay pending, verify `ceph-block` is the active StorageClass and Ceph is healthy.

## Roadmap & Next Hardening Steps

- **S3 / RGW WAL Archiving (`TASK-P3-01`)**: Implemented (Continuous WAL archiving and daily scheduled base backups to Ceph RGW).
- **Dynamic Database Credentials (`TASK-P3-04`)**: Integrate with OpenBao PostgreSQL secrets engine to replace static `app-user` credentials with rotating short-lived roles.
- **Network Isolation (`TASK-P4-02`)**: Restrict port 5432 ingress to authorized application pods (`visit-gateway`, `visit-processor`) via `CiliumNetworkPolicy`.

