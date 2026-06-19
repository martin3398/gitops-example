# Deployment Pipeline Runbook

This runbook describes the current zero-to-100 deployment model for the dev environment.

## Goal

The deployment is intentionally staged:

1. create infrastructure
2. bootstrap Kubernetes without Flux
3. install Flux and the core platform stage
4. bootstrap OpenBao procedurally
5. deploy data services
6. deploy applications

Flux remains the long-running reconciler for Kubernetes resources. Ansible controls when each stage is introduced to avoid bootstrap races, especially around OpenBao initialization and External Secrets.

## Main Commands

Run local checks:

```bash
task pipeline:check
```

Provision infrastructure and bootstrap Kubernetes without Flux:

```bash
task pipeline:init_cluster
```

Install Flux and deploy the core platform stage:

```bash
task ansible:core
```

Run the full deployment:

```bash
task pipeline:main
```

Fetch a local kubeconfig after deployment:

```bash
task ansible:get_kubeconfig_public
```

## Full Deployment Sequence

`task pipeline:main` runs:

```text
pipeline:init_cluster
ansible:core
ansible:openbao
ansible:postgres
ansible:kafka
ansible:apps
```

Expanded sequence:

1. `tofu:plan`
2. `tofu:apply`
3. `ansible:inventory`
4. `ansible:smoke`
5. `ansible:base`
6. `ansible:runtime`
7. `ansible:bootstrap`
8. `ansible:core`
9. `ansible:openbao`
10. `ansible:postgres`
11. `ansible:kafka`
12. `ansible:apps`

## Task Responsibilities

`pipeline:init_cluster`:

- provisions AWS infrastructure with OpenTofu
- exports OpenTofu outputs
- generates Ansible inventory
- validates SSM connectivity
- prepares hosts
- installs runtime dependencies
- bootstraps kubeadm and Cilium
- stops before Flux is installed

`ansible:core`:

- installs Flux controllers
- creates Flux Git deploy credentials
- applies `kubernetes/flux/clusters/dev/core/`
- waits for `GitRepository/dev-repo`

`ansible:openbao`:

- waits for core, Ceph, and ingress Flux Kustomizations
- applies `kubernetes/flux/clusters/dev/security/`
- initializes OpenBao if needed
- saves init material remotely and fetches it locally to `.secrets/openbao-init.dev.json`
- unseals leader and followers
- configures Kubernetes auth for External Secrets Operator
- seeds deterministic dev secrets
- waits for `ClusterSecretStore/openbao`

`ansible:postgres`:

- waits for core/Ceph and OpenBao `ClusterSecretStore`
- applies `kubernetes/flux/clusters/dev/data-postgres/`
- waits for the `data-postgres/app-user` Secret produced by External Secrets

`ansible:kafka`:

- waits for core/Ceph
- applies `kubernetes/flux/clusters/dev/data-kafka/`

`ansible:apps`:

- applies `kubernetes/flux/clusters/dev/apps/`
- reconciles app policy and visit demo app Flux Kustomizations

## Flux Stage Layout

Flux cluster stage definitions live under:

```text
kubernetes/flux/clusters/dev/
```

The top-level `dev/` directory intentionally contains no YAML manifests. Each stage is a self-contained Kustomize folder with its own `kustomization.yaml`.

```text
kubernetes/flux/clusters/dev/
  core/
    kustomization.yaml
    gitrepository.yaml
    kustomization-platform-core.yaml
    kustomization-platform-ingress.yaml
    kustomization-platform-data-ceph.yaml
  security/
    kustomization.yaml
    kustomization-platform-security.yaml
  data-postgres/
    kustomization.yaml
    kustomization-platform-data-postgres.yaml
  data-kafka/
    kustomization.yaml
    kustomization-platform-data-kafka.yaml
  apps/
    kustomization.yaml
    kustomization-platform-apps.yaml
    kustomization-app-visit-web.yaml
    kustomization-app-visit-processing.yaml
  observability/
    kustomization.yaml
    kustomization-platform-observability.yaml
```

This avoids unreferenced sibling Kustomization files. A stage folder is applied only when the matching Ansible task runs.

## Stage Mapping

| Stage folder | Applied by | Flux resources |
| --- | --- | --- |
| `core/` | `ansible:core` | `GitRepository/dev-repo`, `platform-core`, `platform-ingress`, `platform-data-ceph` |
| `security/` | `ansible:openbao` | `platform-security` |
| `data-postgres/` | `ansible:postgres` | `platform-data-postgres` |
| `data-kafka/` | `ansible:kafka` | `platform-data-kafka` |
| `apps/` | `ansible:apps` | `platform-apps`, `app-visit-web`, `app-visit-processing` |
| `observability/` | not in `pipeline:main` yet | `platform-observability` |

## Why Staged Flux Application

OpenBao cannot be fully bootstrapped by declarative manifests alone in this lab flow. It requires procedural init/unseal/join/auth/seed steps.

If Postgres or apps are applied before OpenBao and External Secrets are ready, their required Kubernetes Secrets can be missing. The staged Ansible tasks make those dependencies explicit without making every Flux reconciliation depend on every upstream layer forever.

## Partial Reruns

Common reruns:

```bash
task ansible:core
task ansible:openbao
task ansible:postgres
task ansible:kafka
task ansible:apps
```

After a full infrastructure destroy/recreate, remove stale local OpenBao init material before rebuilding:

```bash
rm -f .secrets/openbao-init.dev.json
task pipeline:main
```

## Validation

Check task definitions:

```bash
task -l
```

Check Ansible syntax:

```bash
task ansible:lint_inventory
```

Validate Flux stage folders locally:

```bash
kubectl kustomize kubernetes/flux/clusters/dev/core
kubectl kustomize kubernetes/flux/clusters/dev/security
kubectl kustomize kubernetes/flux/clusters/dev/data-postgres
kubectl kustomize kubernetes/flux/clusters/dev/data-kafka
kubectl kustomize kubernetes/flux/clusters/dev/apps
kubectl kustomize kubernetes/flux/clusters/dev/observability
```

After deployment, check Flux:

```bash
kubectl -n flux-system get gitrepositories
kubectl -n flux-system get kustomizations
kubectl -n flux-system get helmreleases
```

Check OpenBao and External Secrets readiness:

```bash
kubectl -n openbao get pods
kubectl get clustersecretstore openbao
kubectl -n data-postgres get secret app-user
```

## CI Flow

The GitHub `ansible-run` workflow follows the same staged model after infrastructure apply:

1. generate inventory
2. smoke
3. base
4. runtime
5. cluster bootstrap
6. Flux/core bootstrap
7. OpenBao bootstrap
8. Postgres deploy
9. Kafka deploy
10. apps deploy

The local `task pipeline:main` command is the closest equivalent to the full deployment chain.
