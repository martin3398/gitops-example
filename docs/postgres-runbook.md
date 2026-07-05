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

The visit processor writes through the read-write service endpoint:

```text
postgres-rw.data-postgres.svc:5432
```

## Validation

Use a workstation with cluster access.

```bash
kubectl -n flux-system get kustomizations,helmreleases
kubectl -n data-postgres get pods
kubectl -n data-postgres get cluster
kubectl -n data-postgres get secret app-user
kubectl -n data-postgres get svc
task pipeline:verify
```

## Troubleshooting

- If the cluster is not Ready, inspect the CloudNativePG operator logs in `cnpg-system` or the cluster status in `data-postgres`.
- If the application secret is missing, verify OpenBao and the External Secrets `ClusterSecretStore` are Ready.
- If PVCs stay pending, verify `ceph-block` is the active StorageClass and Ceph is healthy.
