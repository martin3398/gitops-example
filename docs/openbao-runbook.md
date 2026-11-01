# OpenBao Runbook (HA)

This runbook describes the production-style OpenBao baseline for this repository.

## Topology

- 3 OpenBao server pods
- integrated Raft storage
- ingress host-based exposure via `http://bao.gitops.local` over HTTP

## Kubernetes/Flux Layout

- `kubernetes/infrastructure/base/security/openbao/` - OpenBao namespace, chart, and ClusterSecretStore
- `kubernetes/infrastructure/base/core-services/operators/external-secrets/` - External Secrets Operator
- app secret consumers:
  - `kubernetes/apps/base/visit-web/externalsecret-web.yaml`
  - `kubernetes/apps/base/visit-processing/externalsecret-processing.yaml`
  - `kubernetes/infrastructure/base/data-postgres/externalsecret-app-user.yaml`

## Local Host Entry

OpenBao uses host-based gateway routing with the local-only hostname `bao.gitops.local`.
The shared local host-entry convention is documented in the repository README and the `task ingress:hosts_entries` helper.

1. Generate the `/etc/hosts` entries using the Taskfile helper or OpenTofu output:

```bash
task ingress:hosts_entries
# or manually resolve:
nslookup "$(tofu -chdir=infra output -raw ingress_public_endpoint)"
```

2. Add each returned address to `/etc/hosts`:

```text
<nlb-ip-1> gitops.local grafana.gitops.local prometheus.gitops.local bao.gitops.local
<nlb-ip-2> gitops.local grafana.gitops.local prometheus.gitops.local bao.gitops.local
```

If you are also exposing the other browser UIs locally, use the shared host entry shape from the README so `gitops.local`, `grafana.gitops.local`, `prometheus.gitops.local`, and `bao.gitops.local` resolve consistently.

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

- `task ansible:core` has installed Flux and all Kustomizations.
- The `infrastructure-data-ceph` Flux Kustomization is Ready, which OpenBao depends on.

The task runs the Ansible playbook on `control_plane[0]`. You can run the playbook directly with:

```bash
ANSIBLE_CONFIG=ansible/ansible.cfg \
  .venv/bin/ansible-playbook -i ansible/inventories/dev/hosts.yml ansible/playbooks/openbao-bootstrap.yml
```

The automated sequence does:

1. waits for `infrastructure-data-ceph`
2. waits for `openbao-0`
3. initializes `openbao-0` with `bao operator init -format=json`
4. saves init output to `.secrets/openbao-init.dev.json`
5. unseals `openbao-0`
6. joins `openbao-1` and `openbao-2` to `openbao-0`
7. unseals `openbao-1` and `openbao-2`
8. configures Kubernetes auth for External Secrets Operator
9. seeds current dev DB secrets
10. restarts the External Secrets controller to force a fresh `ClusterSecretStore` reconcile
11. waits for `ClusterSecretStore/openbao` to become Ready

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

This writes deterministic secrets for the current Postgres app user, visit demo DB consumers, and Loki S3 credentials.

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

## Raft Snapshots & Disaster Recovery (TASK-P3-02)

OpenBao state is backed up continuously to local on-prem object storage in Ceph RGW (`s3://openbao-backups/`) using an automated `CronJob` and native Kubernetes authentication.

### Architecture

1. **Storage Provisioning**: An `ObjectBucketClaim` (`openbao-backups`) in `openbao` requests an S3 bucket from Rook-Ceph via the `ceph-bucket` StorageClass.
2. **Least-Privilege Auth**: The `openbao-snapshot` ServiceAccount authenticates to OpenBao using the Kubernetes auth engine (`auth/kubernetes/role/openbao-snapshot`). It is mapped to the `raft-snapshot` policy (`path "sys/storage/raft/snapshot" { capabilities = ["read"] }`) and cannot read any secret data.
3. **Automated Scheduled Snapshots**: `CronJob/openbao-snapshot` runs every 6 hours (`0 */6 * * *`):
   - **Init Container** (`quay.io/openbao/openbao:2.5.5`): Obtains an OpenBao token via Kubernetes auth and dumps the binary snapshot to a shared `emptyDir` volume using `bao operator raft snapshot save`.
   - **Upload Container** (`public.ecr.aws/aws-cli/aws-cli:2.17.53`): Streams the snapshot to Ceph RGW S3, publishing both a timestamped snapshot (`raft-snapshot-<timestamp>.snap`) and updating `raft-snapshot-latest.snap`.

### Operational Commands

Trigger an immediate on-demand snapshot:

```bash
task openbao:snapshot
```

List snapshots stored in Ceph RGW S3:

```bash
task openbao:list_snapshots
```

Restore from the latest snapshot:

```bash
task openbao:restore
```

Restore a specific historical snapshot:

```bash
SNAPSHOT=raft-snapshot-20261027_210000.snap task openbao:restore
```

Or execute the restore playbook directly:

```bash
ANSIBLE_CONFIG=ansible/ansible.cfg \
  .venv/bin/ansible-playbook -i ansible/inventories/dev/hosts.yml ansible/playbooks/openbao-restore.yml \
  -e snapshot_key=raft-snapshot-latest.snap
```

The restore procedure automatically:
1. Downloads the target `.snap` file from Ceph RGW to the control plane.
2. Copies it into `openbao-0` and executes `bao operator raft snapshot restore -force`.
3. Cleans up temporary artifacts.
4. Allows Raft consensus to sync and re-unseals the leader and follower pods if needed.
5. Verifies Raft peer membership and asserts that stored secrets remain intact.

## Dynamic PostgreSQL Secrets Engine (TASK-P3-04)

OpenBao dynamically provisions short-lived, least-privilege PostgreSQL credentials for the application microservices (`visit-gateway` and `visit-processor`), completely eliminating long-lived shared passwords for data plane access.

### Architecture

1. **Database Secrets Engine**: OpenBao's `database` engine is mounted at `database/` using the native `postgresql-database-plugin`. It maintains an administrative connection pool to `postgres-rw.data-postgres.svc:5432/app` as database owner `app` (configured with `CREATEROLE`).
2. **Cilium Network Isolation**: East-west network policies (`openbao-server-isolation` and `postgres-isolation`) explicitly permit TCP port 5432 traffic from OpenBao pods in `openbao` to the PostgreSQL cluster in `data-postgres`.
3. **Dynamic Roles**:
   - `visit-gateway`: Configured with 1-hour default TTL (`max_ttl: 24h`). Creates temporary user `v-token-visit-gat-...` and grants `SELECT, INSERT` on table `visits` and sequences.
   - `visit-processor`: Configured with 1-hour default TTL (`max_ttl: 24h`). Creates temporary user `v-token-visit-proc-...` and grants `SELECT, INSERT` on table `visits` and sequences.
4. **Table Ownership & Drop Safety**:
   - The `visits` table and primary sequence are owned permanently by administrative user `app`.
   - Dynamic users are strictly granted operational DML privileges and never own database objects, allowing OpenBao to cleanly revoke and drop expired roles (`DROP ROLE IF EXISTS ...`) without encountering PostgreSQL dependency lock errors.
5. **Platform ClusterGenerators & Stakater Reloader**:
   - `ClusterGenerator` resources (`generators.external-secrets.io/v1alpha1`) are defined centrally at the platform layer in `kubernetes/infrastructure/base/security/openbao/clustergenerator-postgres.yaml`.
   - ClusterGenerators authenticate to OpenBao via ESO's trusted controller ServiceAccount (`external-secrets-external-secrets` in `external-secrets`) using the scoped `eso-sync` role.
   - Applications (`visit-web` and `visit-processing`) are pure consumers: their `ExternalSecret` simply references the `ClusterGenerator` by name (`openbao-postgres-visit-gateway` / `openbao-postgres-visit-processor`), completely eliminating OpenBao endpoints, auth plumbing, or dummy ServiceAccounts from application overlays.
   - Stakater Reloader is deployed cluster-wide in namespace `reloader`. When ESO updates Secret `visit-demo-db` on its 45-minute cycle, Reloader detects the SHA change and immediately performs a zero-downtime rolling restart of the annotated `visit-gateway` and `visit-processor` deployments before the 1-hour lease expires.
   - Microservices consume both credentials dynamically via `valueFrom: secretKeyRef` in their Helm charts.

### Operational & Diagnostic Commands

1. Check database secrets engine configuration:
```bash
kubectl -n openbao exec openbao-0 -- env BAO_ADDR=http://127.0.0.1:8200 BAO_TOKEN="$BAO_TOKEN" \
  bao read database/config/postgres
```

2. Inspect configured roles:
```bash
kubectl -n openbao exec openbao-0 -- env BAO_ADDR=http://127.0.0.1:8200 BAO_TOKEN="$BAO_TOKEN" \
  bao read database/roles/visit-gateway
```

3. Generate a dynamic test credential:
```bash
kubectl -n openbao exec openbao-0 -- env BAO_ADDR=http://127.0.0.1:8200 BAO_TOKEN="$BAO_TOKEN" \
  bao read database/creds/visit-gateway
```

4. Inspect active leases for database credentials:
```bash
kubectl -n openbao exec openbao-0 -- env BAO_ADDR=http://127.0.0.1:8200 BAO_TOKEN="$BAO_TOKEN" \
  bao list sys/leases/lookup/database/creds/visit-gateway
```

5. Revoke an active lease on demand:
```bash
kubectl -n openbao exec openbao-0 -- env BAO_ADDR=http://127.0.0.1:8200 BAO_TOKEN="$BAO_TOKEN" \
  bao lease revoke -prefix database/creds/visit-gateway
```

6. Verify dynamic roles in PostgreSQL:
```bash
kubectl -n data-postgres exec -c postgres postgres-1 -- \
  psql -U postgres -d app -c "\du v-*"
```

## Notes & Roadmap Hardening Tasks

- **Unsealing Model**: Auto-unseal is omitted for this lab; unsealing and Raft joining are fully automated by the Ansible bootstrap role (`task ansible:openbao`).
- **TLS / HTTPS**: Currently plaintext HTTP; TLS certificate automation is deferred / out of scope per Agents.md.
- **Raft Snapshots & Disaster Recovery (`TASK-P3-02`)**: Complete. Automated CronJob snapshots to S3/Ceph RGW and recovery playbook (`ansible/playbooks/openbao-restore.yml`).
- **Dynamic Database Credentials (`TASK-P3-04`)**: Complete. OpenBao PostgreSQL secrets engine configured with dynamic roles (`visit-gateway`, `visit-processor`), ESO `VaultDynamicSecret` generators, and automated lease revocation.
- **Network Policy Isolation (`TASK-P4-02`)**: Complete. Enforced via `openbao-server-isolation` and `openbao-snapshot-isolation` (`CiliumNetworkPolicy`) restricting port 8200 API access strictly to Envoy Gateway proxy (`reserved:ingress`), External Secrets Operator, and snapshot CronJob; port 8201 restricted to Raft peers; egress restricted to Ceph RGW (`rook-ceph:80`), PostgreSQL (`data-postgres:5432`), and Kubernetes API server.
