# Deployment Pipeline Runbook

This runbook describes the current zero-to-100 deployment model for the dev environment.

## Goal

The deployment is intentionally staged:

1. create infrastructure
2. bootstrap Kubernetes without Flux
3. install Flux, core operators, and Ceph
4. deploy monitoring and ingress core platform services
5. bootstrap OpenBao procedurally
6. deploy data services
7. deploy applications

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

Install Flux and deploy core operators and storage:

```bash
task ansible:core
```

Deploy monitoring and ingress core platform services:

```bash
task ansible:core_platform
```

Run the full deployment:

```bash
task pipeline:main
```

Fetch a local kubeconfig after deployment:

```bash
task ansible:get_kubeconfig_public
```

## Taskfile Layout

The root `Taskfile.yml` only contains shared variables and includes. Task groups live under `.taskfiles/`:

```text
Taskfile.yml
.taskfiles/
  ansible.yml
  env.yml
  ingress.yml
  openbao.yml
  pipeline.yml
  tofu.yml
  venv.yml
```

Public task names stay namespaced and stable:

```bash
task tofu:plan
task ansible:core
task ansible:core_platform
task pipeline:main
```

Cross-namespace calls inside included Taskfiles use root-qualified task names, for example `:ansible:core`. Shared variables such as `INVENTORY`, `VENV_ANSIBLE_PLAYBOOK`, and `KUBECONFIG_FILE` stay in the root `Taskfile.yml`.

`KUBECONFIG_FILE` is rooted with `pwd -P` so included Taskfiles do not accidentally resolve it under `.taskfiles/`.

## Full Deployment Sequence

`task pipeline:main` runs:

```text
pipeline:init_cluster
ansible:core
ansible:core_platform
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
9. `ansible:core_platform`
10. `ansible:openbao`
11. `ansible:postgres`
12. `ansible:kafka`
13. `ansible:apps`

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

`ansible:core_platform`:

- waits for `platform-core` and `platform-data-ceph`
- applies `kubernetes/flux/clusters/dev/core-platform/`
- reconciles `platform-observability`
- reconciles `platform-ingress` after monitoring is ready

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
    kustomization-platform-data-ceph.yaml
  core-platform/
    kustomization.yaml
    kustomization-platform-observability.yaml
    kustomization-platform-ingress.yaml
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
```

This avoids unreferenced sibling Kustomization files. A stage folder is applied only when the matching Ansible task runs.

## Stage Mapping

| Stage folder | Applied by | Flux resources |
| --- | --- | --- |
| `core/` | `ansible:core` | `GitRepository/dev-repo`, `platform-core`, `platform-data-ceph` |
| `core-platform/` | `ansible:core_platform` | `platform-observability`, `platform-ingress` |
| `security/` | `ansible:openbao` | `platform-security` |
| `data-postgres/` | `ansible:postgres` | `platform-data-postgres` |
| `data-kafka/` | `ansible:kafka` | `platform-data-kafka` |
| `apps/` | `ansible:apps` | `platform-apps`, `app-visit-web`, `app-visit-processing` |

## Why Staged Flux Application

OpenBao cannot be fully bootstrapped by declarative manifests alone in this lab flow. It requires procedural init/unseal/join/auth/seed steps.

If Postgres or apps are applied before OpenBao and External Secrets are ready, their required Kubernetes Secrets can be missing. The staged Ansible tasks make those dependencies explicit without making every Flux reconciliation depend on every upstream layer forever.

Monitoring and ingress are also staged together because `ingress-nginx` renders a `ServiceMonitor` when metrics are enabled. The `ServiceMonitor` CRD is installed by Prometheus Operator through `kube-prometheus-stack`, so `platform-observability` must be ready before `platform-ingress` installs.

The same CRD race can affect observability subcharts. `loki` depends on `kube-prometheus-stack`, and `promtail` depends on both `kube-prometheus-stack` and `loki`, because Loki and Promtail also render `ServiceMonitor` resources.

## ServiceMonitor Failure Recovery

If ingress was applied before Prometheus Operator CRDs existed, Flux can report an error like:

```text
Helm install failed ... no matches for kind "ServiceMonitor" in version "monitoring.coreos.com/v1"
```

After the refactor, apply the corrected stage order:

```bash
task ansible:core
task ansible:core_platform
```

If the failed Helm release remains stuck, reconcile it after `platform-observability` is ready:

```bash
flux reconcile kustomization platform-observability -n flux-system --timeout=20m
flux reconcile kustomization platform-ingress -n flux-system --timeout=15m
flux reconcile helmrelease ingress-nginx -n flux-system
```

If Helm keeps retrying stale failed release state, uninstall the failed release and let Flux recreate it:

```bash
helm -n ingress-nginx uninstall ingress-nginx-ingress-nginx
flux reconcile helmrelease ingress-nginx -n flux-system
```

## Partial Reruns

Common reruns:

```bash
task ansible:core
task ansible:core_platform
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

Check pipeline order without executing it:

```bash
task --dry pipeline:main
```

Validate Flux stage folders locally:

```bash
kubectl kustomize kubernetes/flux/clusters/dev/core
kubectl kustomize kubernetes/flux/clusters/dev/core-platform
kubectl kustomize kubernetes/flux/clusters/dev/security
kubectl kustomize kubernetes/flux/clusters/dev/data-postgres
kubectl kustomize kubernetes/flux/clusters/dev/data-kafka
kubectl kustomize kubernetes/flux/clusters/dev/apps
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

## Open Work

Platform reliability:
- add `pipeline:verify` for post-deploy checks
- add Velero and restore drills
- add component restore notes for OpenBao, Postgres, Kafka, and Ceph

Secrets and credentials:
- add OpenBao Raft peer verification to automation
- add OpenBao snapshot backup/restore
- decide on auto-unseal for non-dev use
- replace deterministic static Postgres credentials with dynamic OpenBao database credentials
- add secret rotation workflow and narrower per-consumer OpenBao policies

Security:
- review cluster auth/RBAC
- add baseline network policies
- add image scanning in CI
- expand Kyverno policies beyond the current scheduling preference baseline

Application operations:
- add reproducible load generator scenarios
- implement and tune HPA for `visit-processor`
- add Kafka consumer lag and app SLO dashboards/alerts

Maintenance:
- add Renovate or equivalent dependency/workflow update automation
- keep MongoDB as pending/optional unless explicitly brought back into scope

## CI Flow

The GitHub `ansible-run` workflow follows the same staged model after infrastructure apply:

1. generate inventory
2. smoke
3. base
4. runtime
5. cluster bootstrap
6. Flux/core bootstrap
7. core platform deploy
8. OpenBao bootstrap
9. Postgres deploy
10. Kafka deploy
11. apps deploy

The local `task pipeline:main` command is the closest equivalent to the full deployment chain.
