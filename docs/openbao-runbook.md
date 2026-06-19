# OpenBao Runbook (HA)

This runbook describes the production-style OpenBao baseline for this repository.

## Topology

- 3 OpenBao server pods
- integrated Raft storage
- ingress host-based exposure via `http://bao.gitops.local` over HTTP

## Kubernetes/Flux Layout

- `kubernetes/platform/dev/security/openbao/` - OpenBao namespace, chart, and ClusterSecretStore
- `kubernetes/platform/dev/core-services/operators/external-secrets/` - External Secrets Operator
- app secret consumers:
  - `kubernetes/apps/dev/visit-web/externalsecret-web.yaml`
  - `kubernetes/apps/dev/visit-processing/externalsecret-processing.yaml`
  - `kubernetes/platform/dev/data-platform/services/postgres/externalsecret-app-user.yaml`

## Local Host Entry

OpenBao uses host-based ingress with the local-only hostname `bao.gitops.local`.

1. Resolve the current AWS ingress NLB endpoint:

```bash
nslookup "$(tofu -chdir=infra output -raw ingress_public_endpoint)"
```

2. Add each returned address to `/etc/hosts`:

```text
<nlb-ip-1> bao.gitops.local
<nlb-ip-2> bao.gitops.local
```

You can print the current entries with:

```bash
task openbao:hosts_entries
```

3. Open OpenBao at `http://bao.gitops.local`.

Refresh these entries after a full infrastructure destroy/recreate because NLB IPs can change.

## Bootstrap (fresh cluster)

For this rebuild-heavy lab, OpenBao init material is stored locally in a gitignored file and then parsed to automate Raft join, unseal, auth bootstrap, and initial dev seeding.

Automation lives in:

- `ansible/playbooks/openbao-bootstrap.yml`
- `ansible/roles/openbao_bootstrap/`

The init file path is in the repository root:

```text
.secrets/openbao-init.dev.json
```

This file contains unseal keys and the root token. Keep it local only.

Run the full dev bootstrap:

```bash
task ansible:openbao
```

The task runs the Ansible playbook on `control_plane[0]`. You can run the playbook directly with:

```bash
ANSIBLE_CONFIG=ansible/ansible.cfg \
  .venv/bin/ansible-playbook -i ansible/inventories/dev/hosts.yml ansible/playbooks/openbao-bootstrap.yml
```

The automated sequence does:

1. waits for `openbao-0`
2. initializes `openbao-0` with `bao operator init -format=json`
3. saves init output to `.secrets/openbao-init.dev.json`
4. unseals `openbao-0`
5. joins `openbao-1` and `openbao-2` to `openbao-0`
6. unseals `openbao-1` and `openbao-2`
7. configures Kubernetes auth for External Secrets Operator
8. seeds current dev DB secrets

Manual Raft join command, for reference:

```bash
kubectl -n openbao exec -ti openbao-1 -- bao operator raft join http://openbao-0.openbao-internal:8200
```

Expected final state:

```text
openbao-0      1/1 Running
openbao-1      1/1 Running
openbao-2      1/1 Running
Initialized    true
Sealed         false
```

## Reset Broken Raft State

If a follower is stuck or has stale Raft data, reset OpenBao state destructively and retry from a clean cluster:

```bash
flux suspend helmrelease openbao -n flux-system
helm -n openbao uninstall openbao
kubectl -n openbao delete pvc -l app.kubernetes.io/instance=openbao
rm -f .secrets/openbao-init.dev.json
flux resume helmrelease openbao -n flux-system
flux reconcile helmrelease openbao -n flux-system
```

Then run `task ansible:openbao` again.

## Deterministic Dev Secret Seeding

Dev seeding is handled by the OpenBao Ansible bootstrap role.

Override the deterministic seed with `OPENBAO_DEV_SEED` when running the task or playbook.

This writes deterministic secrets for the current Postgres app user and visit demo DB consumers.

## Notes

- No TLS is configured in this phase; this is intentional for current lab scope.
- Rotate dev credentials by changing `OPENBAO_DEV_SEED` and re-running the OpenBao Ansible bootstrap.
- Kafka and Grafana are intentionally not seeded until manifests consume those secrets.
