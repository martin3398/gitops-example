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
The shared local ingress-domain convention is documented in the repository README under `Public Ingress Access`.

1. Resolve the current AWS ingress NLB endpoint:

```bash
nslookup "$(tofu -chdir=infra output -raw ingress_public_endpoint)"
```

2. Add each returned address to `/etc/hosts`:

```text
<nlb-ip-1> gitops.local grafana.gitops.local bao.gitops.local
<nlb-ip-2> gitops.local grafana.gitops.local bao.gitops.local
```

If you are also exposing the other browser UIs locally, use the shared host entry shape from the README so `gitops.local`, `grafana.gitops.local`, and `bao.gitops.local` resolve consistently.

You can print the current entries with:

```bash
task ingress:hosts_entries
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

The same material is stored on the first control plane at:

```text
/root/.secrets/openbao-init.dev.json
```

This file contains unseal keys and the root token. Keep it local only.

Run the full dev bootstrap:

```bash
task ansible:openbao
```

Prerequisites:

- `task ansible:core` has installed Flux, core operators, and Ceph.
- `task ansible:core_platform` has installed monitoring and ingress.
- The `platform-ingress` Flux Kustomization is Ready, so `bao.gitops.local` can route after OpenBao is unsealed.

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
ansible control_plane[0] -i ansible/inventories/dev/hosts.yml -b -m ansible.builtin.file -a 'path=/root/.secrets/openbao-init.dev.json state=absent'
flux resume helmrelease openbao -n flux-system
flux reconcile helmrelease openbao -n flux-system
```

Then run `task ansible:openbao` again.

## Deterministic Dev Secret Seeding

Dev seeding is handled by the OpenBao Ansible bootstrap role.

Override the deterministic seed with `OPENBAO_DEV_SEED` when running the task or playbook.

This writes deterministic secrets for the current Postgres app user and visit demo DB consumers.

## Validation

The standard post-deploy verification path is:

```bash
task pipeline:verify
```

It checks OpenBao pod readiness, initialized/unsealed status, `ClusterSecretStore/openbao`, and synced ExternalSecrets.

Check OpenBao status:

```bash
task openbao:status
```

Check Raft peers from the leader:

```bash
kubectl -n openbao exec openbao-0 -- env BAO_ADDR=http://127.0.0.1:8200 bao operator raft list-peers
```

Check ESO integration:

```bash
kubectl get clustersecretstore openbao
kubectl -n data-postgres get secret app-user
```

## Notes

- No TLS is configured in this phase; this is intentional for current lab scope.
- Rotate dev credentials by changing `OPENBAO_DEV_SEED` and re-running the OpenBao Ansible bootstrap.
- Kafka and Grafana are intentionally not seeded until manifests consume those secrets.
- Auto-unseal is not implemented yet.
- OpenBao snapshot backup/restore is not implemented yet.
- Postgres uses deterministic static dev credentials; dynamic database credentials are not implemented yet.
