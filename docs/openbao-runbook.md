# OpenBao Runbook (HA)

This runbook describes the production-style OpenBao baseline for this repository.

## Topology

- 3 OpenBao server pods
- integrated Raft storage
- worker-node-only scheduling
- ingress path-based exposure via `http://<ingress_public_endpoint>/bao` over HTTP

## Kubernetes/Flux Layout

- `kubernetes/platform/dev/security/openbao/` - OpenBao namespace, chart, and ClusterSecretStore
- `kubernetes/platform/dev/core-services/operators/external-secrets/` - External Secrets Operator
- app secret consumers:
  - `kubernetes/apps/dev/visit-web/externalsecret-web.yaml`
  - `kubernetes/apps/dev/visit-processing/externalsecret-processing.yaml`
  - `kubernetes/platform/dev/data-platform/services/postgres/externalsecret-app-user.yaml`

## Ingress Endpoint

1. Ensure infra output `ingress_public_endpoint` is available.
2. OpenBao is exposed at `http://<ingress_public_endpoint>/bao`.
3. Ingress rewrites `/bao/...` to `/...` at the OpenBao service.

## Bootstrap (fresh cluster)

1. Reconcile Flux and wait until `openbao` pods are ready.
2. Initialize OpenBao once:

```bash
kubectl -n openbao exec -it openbao-0 -- bao operator init -key-shares=5 -key-threshold=3
```

3. Store unseal keys and root token in your secure operator vault (never in Git).
4. Unseal all OpenBao pods:

```bash
kubectl -n openbao exec -it openbao-0 -- bao operator unseal
kubectl -n openbao exec -it openbao-1 -- bao operator unseal
kubectl -n openbao exec -it openbao-2 -- bao operator unseal
```

5. Enable Kubernetes auth and ESO role:

```bash
kubectl -n openbao exec -it openbao-0 -- sh
export OPENBAO_ADDR=http://127.0.0.1:8200
export OPENBAO_TOKEN=<root-token>
bao auth enable kubernetes || true
bao write auth/kubernetes/config kubernetes_host="https://kubernetes.default.svc"
bao policy write eso-sync - <<'EOF'
path "secret/data/dev/*" {
  capabilities = ["read"]
}
EOF
bao write auth/kubernetes/role/eso-sync \
  bound_service_account_names="external-secrets" \
  bound_service_account_namespaces="external-secrets" \
  policies="eso-sync" \
  ttl="1h"
exit
```

## Deterministic Dev Secret Seeding

Use the seed helper after bootstrap for tear-down/recreate consistency:

```bash
chmod +x scripts/openbao/seed-dev-secrets.sh
OPENBAO_TOKEN=<operator-token> OPENBAO_DEV_SEED=<stable-seed> ./scripts/openbao/seed-dev-secrets.sh
```

This writes deterministic secrets for Postgres/Kafka/Grafana and app DB consumers.

## Notes

- No TLS is configured in this phase; this is intentional for current lab scope.
- Rotate credentials by changing `OPENBAO_DEV_SEED` and re-running seed script.
- Grafana credentials are pre-seeded for later integration milestone.
