# GitOps Example

Kubernetes GitOps example focused on reproducible infrastructure and operations.

## Goal

This project is a practical, end-to-end GitOps example environment.

It uses AWS EC2 for convenience, but follows a self-managed approach designed to mirror on-prem responsibilities:
- infrastructure as code
- automated host and cluster bootstrap
- GitOps-driven platform and workload delivery
- operational practices (observability, secrets, backups, policy/security)

## Architecture Target

- Kubernetes cluster on EC2 with `3 control planes` and `3 workers`
- Infrastructure provisioning with `OpenTofu/Terraform`
- Host bootstrap and kubeadm automation with `Ansible`
- GitOps delivery with `Flux`
- CI/CD with `GitHub Actions`
- Open-source platform tooling deployed with Helm via Flux

## Progress Checklist

Use this as the single source of truth for what is done and what is next.

### Core Technologies

- [x] OpenTofu/Terraform (AWS infra + remote state + CI)
- [x] Ansible (host prep + kubeadm bootstrap + CI stages)
- [x] Kubernetes (kubeadm, 3 control planes + 3 workers)
- [x] containerd runtime
- [x] Cilium CNI
- [x] Flux GitOps bootstrap automation
- [x] ingress-nginx
- [x] cert-manager/HTTPS intentionally excluded from this AWS lab scope
- [x] Monitoring/logging baseline (kube-prometheus-stack + Loki + Grafana)
- [ ] Upgrade Loki to distributed mode on Ceph object storage (after Phase 3 Ceph lab)
- [x] Application GitOps image-update automation (Flux image automation writes GitOps values)
- [x] Application GitOps delivery (visit-web + visit-gateway + visit-processor)
- [x] Postgres baseline: CloudNativePG operator + dev cluster via Flux
- [x] OpenBao baseline: HA Raft cluster + External Secrets integration for platform/app credentials
- [x] Load generator lab: reproducible traffic scenarios for scaling validation
- [ ] Autoscaling lab: HPA for `visit-processor` with measured scaling behavior
- [ ] Cluster authentication hardening (cluster-level authn/authz only; no app-level auth scope)
- [ ] Dependency/update automation baseline (Renovate for app/runtime/workflow updates)
- [ ] Credential management hardening: dynamic DB credentials, secret rotation, stronger OpenBao init material handling, GHCR pull secrets, and Flux Git write credentials
- [x] Kafka baseline: Strimzi operator + 3-broker KRaft cluster via Flux (dev defaults)
- [x] Ceph lab: storage classes and stateful workload migration validation
- [ ] Backups (Velero + restore drill)

Visit demo ownership model:
- each service has its own Helm chart under `charts/visit-ui`, `charts/visit-gateway`, `charts/visit-processor`, and `charts/visit-loadgen`
- Flux deploys each service with its own `HelmRelease` under `kubernetes/apps/base/visit-web/` and `kubernetes/apps/base/visit-processing/`
- Flux reconciles app stages through `kubernetes/clusters/dev/apps/`
- `visit-loadgen` runs as an always-on Kubernetes deployment in `paused` mode by default; switch it to `random` for stochastic load bands
- GitHub Actions builds and pushes each service image via `.github/workflows/apps-build-publish.yml`
- image flow: CI publishes immutable timestamped tags (`YYYYMMDDHHmmSS-<8sha>`) and Flux tracks newest matching tags for automatic updates
- `visit-ui` runtime architecture uses React Router SSR (Node server entry + browser hydration entry); initial count comes from a route loader and queueing uses direct browser calls to `/api/v1/visit-events`
- ingress routing keeps API calls same-origin: `/api` is routed to `visit-gateway` and `/` is routed to `visit-ui`

### Delivery Steps

- [x] Phase 1.1: AWS VPC, IAM, EC2 topology (3 CP + 3 workers)
- [x] Phase 1.2: Ansible inventory + SSM-first connectivity
- [x] Phase 1.3: Base node preparation (kernel/sysctl/swap/packages)
- [x] Phase 1.4: Runtime install (containerd + kubelet/kubeadm/kubectl)
- [x] Phase 1.5: Cluster bootstrap (kubeadm init/join + Cilium + validation)
- [x] Phase 1.6: CI stages for Ansible (manual-gated)
- [x] Phase 2.1: Flux bootstrap and repository structure under `kubernetes/`
- [x] Phase 2.2a: ingress-nginx via Flux
- [x] Phase 2.2b: cert-manager/HTTPS intentionally excluded from this AWS lab scope
- [x] Phase 2.3: Observability baseline via Flux
- [x] Phase 2.4: Flux image automation wired for app image updates
- [ ] Phase 4: Resilience + backup + policy/security hardening

## Stack (Planned)

- Provisioning: OpenTofu/Terraform
- Automation: Ansible
- Kubernetes: kubeadm + containerd
- CNI: Cilium (fallback Calico)
- Ingress: ingress-nginx
- TLS/certificates: out of scope for this AWS lab phase
- GitOps: Flux
- CI/CD: GitHub Actions
- Monitoring/Logging: kube-prometheus-stack + Loki + Grafana
- Application data: Postgres primary/replica
- Advanced labs: OpenBao, MongoDB, Kafka, Ceph
- Later-stage reliability/security: Velero, Kyverno, Trivy, baseline network policies

## Delivery Plan

### Phase 1 - Infrastructure + Kubernetes Foundation
- Provision VPC, networking, IAM, and 6 EC2 nodes
- Configure hosts with Ansible (container runtime + kubeadm prerequisites)
- Bootstrap kubeadm cluster (3 control planes, 3 workers)
- Validate node health and basic failover behavior

### Phase 2 - GitOps Platform + Workloads
- Bootstrap Flux from repository Git source
- Install platform add-ons via Helm through Flux
- Deploy frontend + microservices via Flux with E2E queue/count verification
- Postgres baseline is implemented as a platform data service via Flux
- Add GitHub Actions workflows for build/test/publish and use Flux image automation for GitOps image updates

### Phase 3 - Advanced Tooling
- Add OpenBao, MongoDB, Kafka, and Ceph one by one
- Capture setup and operations notes for each
- Add reproducible load-generation scenarios and document scaling observations
- Implement and tune HPA for `visit-processor` under controlled load

### Phase 4 - Resilience, Backup, Policy & Security
- Add Velero and validate restore drills
- Expand policy/security controls (build on Kyverno baseline; add Trivy and additional network policies)
- Add cluster-level authentication hardening and RBAC review
- Add Renovate automation for dependency and workflow update hygiene

## Repository Roadmap

Planned layout:
- `infra/` infrastructure code
- `ansible/` host and kubeadm automation
- `kubernetes/clusters/` Flux bootstrap entrypoints and cluster Kustomizations
- `kubernetes/infrastructure/` platform components and overlays
- `kubernetes/apps/` application bases and overlays
- `kubernetes/labs/` optional MongoDB and future experiments
- `docs/` architecture and runbooks

## Conventions

- Terraform/OpenTofu manages cloud resources only.
- Ansible manages host setup and cluster bootstrap only.
- Flux manages all long-lived Kubernetes resources.
- Avoid manual drift: persistent changes should be committed to Git.
- Storage default is `ceph-block`; local-path provisioner is not part of the platform baseline.

## Environment Naming

- The active environment name is `dev`.
- Flux cluster stage definitions are under `kubernetes/clusters/dev/`
- Platform overlays are under `kubernetes/infrastructure/overlays/dev/`
- App stage definitions are under `kubernetes/clusters/dev/apps/`

## Direct Kubernetes API Access (Dev Option)

If you want direct `kubectl` access from your workstation (without SSM tunnel), enable the public API NLB in `infra/terraform.tfvars`:

- `enable_public_k8s_api = true`
- `lb_public_subnet_cidrs = ["10.42.101.0/24", "10.42.102.0/24"]`
- keep `allowed_admin_cidrs` restricted to your current public IP (for example `/32`)

When enabled, infrastructure output includes both `kubernetes_api_internal_endpoint` (bootstrap) and `kubernetes_api_endpoint` (external access).
Bootstrap uses the internal endpoint (`kube_api_internal_endpoint`) for kubeadm stability.
The public NLB endpoint is exported as `kubernetes_api_endpoint` and `kube_api_public_endpoint` for external `kubectl` access.

## Public Ingress Access (Dev Option)

Note: HTTPS/certificate management is explicitly out of scope for this AWS-based lab. Public ingress is currently documented and validated for HTTP exposure only.

To expose application ingress publicly through an internet-facing NLB, enable these values in `infra/terraform.tfvars`:

- `enable_public_ingress = true`
- `ingress_allowed_cidrs = ["0.0.0.0/0"]` (or restrict to your IP/CIDR)
- `ingress_nodeport_http = 30080`

When enabled, OpenTofu exports `ingress_public_endpoint`.
Ingress traffic path is NLB (`:80`) -> worker NodePort (`30080`) -> `ingress-nginx` controller.

### Local Ingress Domains

Use local `gitops.local` names for browser-facing dev UIs. Map every returned NLB address to the same hostname set in `/etc/hosts`:

```text
<nlb-ip-1> gitops.local grafana.gitops.local bao.gitops.local
<nlb-ip-2> gitops.local grafana.gitops.local bao.gitops.local
```

Resolve the current NLB addresses with:

```bash
nslookup "$(tofu -chdir=infra output -raw ingress_public_endpoint)"
```

Refresh these entries after a full infrastructure destroy/recreate because NLB IPs can change.

Print the current entries with:

```bash
task ingress:hosts_entries
```

| Application | Intended local URL | Current manifest status |
| --- | --- | --- |
| Visit demo | `http://gitops.local` and the AWS ingress NLB DNS name | The app ingress routes `/` and `/api` for `gitops.local` and keeps a hostless fallback for direct AWS NLB access. |
| Grafana | `http://grafana.gitops.local` | Grafana uses host-based ingress at `/` and is not exposed through the AWS NLB hostname or `gitops.local`. |
| OpenBao | `http://bao.gitops.local` | OpenBao uses host-based ingress for `bao.gitops.local`. |

Grafana is the only browser-facing observability UI. Prometheus, Alertmanager, Loki, Promtail metrics, kube-state-metrics, and node-exporter remain internal cluster services. HTTPS and certificate management remain out of scope for this HTTP-only lab.

## Local Environment Variables

For local Ansible/OpenTofu workflow convenience, keep your shell env in a local `.env` file:

1. Create local env file from template:

```bash
cp .env.example .env
```

2. Update values in `.env` for your account/profile.

3. Load variables in your shell when working in this repo:

```bash
set -a; source .env; set +a
```

Notes:
- `.env` is local-only and gitignored.
- `.env.example` is a safe template tracked in git.

## Task Runner

This repository includes a root `Taskfile.yml` for local execution. The root file holds shared variables and includes task groups from `.taskfiles/`.

Taskfile layout:
- `Taskfile.yml` - shared vars and includes
- `.taskfiles/venv.yml` - Python/Ansible virtualenv tasks
- `.taskfiles/env.yml` - local environment checks
- `.taskfiles/tofu.yml` - OpenTofu tasks
- `.taskfiles/ansible.yml` - Ansible and staged deployment tasks
- `.taskfiles/ingress.yml` - local ingress host entry helper tasks
- `.taskfiles/pipeline.yml` - pipeline orchestration tasks
- `.taskfiles/openbao.yml` - OpenBao helper tasks

Common usage:

```bash
task -l
task env:check
task pipeline:check
task pipeline:init_cluster
task ansible:core
task ansible:openbao
task pipeline:verify
task pipeline:main
task ingress:hosts_entries
task ansible:get_kubeconfig_public
```

Task groups:
- OpenTofu: `tofu:*`
- Ansible: `ansible:*`
- Ingress helpers: `ingress:*`
- Pipeline orchestrators: `pipeline:*`

## Cost and Safety Notes

- A 6-node cluster is not free-tier friendly; keep instance sizes small and monitor cost.
- Add destroy/recreate workflows early to control spend.
- Prefer reproducibility over manual fixes.

## Key Documents

- Agent prompt and project execution guidance: `Agents.md`
- CI/CD workflow configuration:
  - OpenTofu check/plan: `.github/workflows/opentofu-check-plan.yml`
  - OpenTofu manual apply/destroy: `.github/workflows/opentofu-apply-destroy.yml`
  - Ansible check: `.github/workflows/ansible-check.yml`
  - Ansible manual run: `.github/workflows/ansible-run.yml`
  - Apps build/publish: `.github/workflows/apps-build-publish.yml`
- Phase 1 infrastructure runbook: `docs/phase1-infra-runbook.md`
- Kafka baseline runbook: `docs/kafka-runbook.md`
- Kyverno baseline runbook: `docs/kyverno-runbook.md`
- GHCR setup runbook: `docs/ghcr-setup.md`
- Deployment pipeline runbook: `docs/deployment-pipeline-runbook.md`
- GitHub Actions runbook: `docs/github-actions-runbook.md`
- Visit demo app runbook: `docs/visit-demo-runbook.md`
- Ansible CI execution runbook: `docs/ansible-ci-runbook.md`

## Status

Phase 1 infrastructure baseline is implemented:
- OpenTofu AWS networking + IAM + 6-node EC2 topology
- S3 + DynamoDB remote state workflow
- GitHub Actions workflows for OpenTofu (check/plan plus manual apply/destroy)

Ansible Iteration 1 scaffold is now in place:
- SSM-first Ansible configuration
- OpenTofu `ansible_inventory` to Ansible `hosts.yml` generator (includes required SSM transfer bucket var)
- Connectivity smoke playbook and initial Ansible runbook (`ansible/README.md`)

Ansible Iteration 2 base preparation is implemented:
- base playbook and role scaffold (`ansible/playbooks/base.yml`, `ansible/roles/base/`)
- package baseline, kernel modules, Kubernetes sysctl settings, and swap disablement
- runtime validation checks for swap and sysctl values

Ansible Iteration 3 runtime preparation is implemented:
- runtime playbook and role scaffold (`ansible/playbooks/runtime.yml`, `ansible/roles/runtime/`)
- containerd install/config with `SystemdCgroup = true`
- Kubernetes node packages (`kubelet`, `kubeadm`, `kubectl`) with package hold and service checks

Ansible Iteration 4 kubeadm bootstrap automation is implemented:
- cluster bootstrap playbook (`ansible/playbooks/cluster-bootstrap.yml`) covering 4A-4D
- first control-plane init, Cilium installation via Cilium CLI, join workflows for additional control planes/workers
- cluster-level validation checks for node readiness and critical kube-system control-plane pods

Ansible Iteration 5 Flux bootstrap handoff automation is implemented:
- Flux bootstrap playbook (`ansible/playbooks/flux-bootstrap.yml`) and role (`ansible/roles/flux_bootstrap/`)
- installs Flux controllers (including image reflector/automation) and applies the core cluster stage
- validates `GitRepository` readiness; stage-specific readiness is handled by later Ansible tasks

GitHub Actions now includes Ansible automation workflows:
- OpenTofu check/plan workflow and manual apply/destroy workflow
- Ansible syntax-check workflow and manual full-run workflow
- App test/build/publish workflow for GHCR
- runbooks in `docs/github-actions-runbook.md` and `docs/ansible-ci-runbook.md`

GitOps structure and Flux-managed charts are implemented:
- cluster stage definitions: `kubernetes/clusters/dev/`
- infrastructure root: `kubernetes/infrastructure/`
- apps root: `kubernetes/apps/`
- ingress chart: `kubernetes/infrastructure/base/ingress-nginx/`
- app stacks: `kubernetes/apps/base/visit-web/` and `kubernetes/apps/base/visit-processing/`

Flux cluster stages:
- `infrastructure-core` installs the Git source and core operators.
- `infrastructure-scheduling` installs the scheduling policy layer after core.
- `infrastructure-data-ceph` installs Ceph storage resources after scheduling.
- `infrastructure-security` installs OpenBao after Ceph.
- `infrastructure-data-postgres` installs Postgres after Ceph.
- `infrastructure-data-kafka` installs Kafka after Ceph.
- `infrastructure-observability` installs monitoring and logging after Ceph.
- `infrastructure-ingress` installs ingress-nginx after observability.
- `apps` installs app policies and visit demo workloads after the required infra is ready.

Flux image automation for apps is implemented:
- `ImageRepository`, `ImagePolicy`, and `ImageUpdateAutomation` for `visit-web`, `visit-gateway`, and `visit-processor`
- image updates commit directly to `main` and are then reconciled by Flux

Kafka baseline is now implemented via Flux:
- Strimzi operator under `kubernetes/infrastructure/base/core-services/operators/strimzi/`
- 3-broker KRaft Kafka cluster under `kubernetes/infrastructure/base/data-kafka/`
- topic and user operators enabled for later tenant self-service templates

Kyverno baseline is now implemented via Flux:
- Kyverno operator under `kubernetes/infrastructure/base/core-services/operators/kyverno/`
- Active cluster policy set under `kubernetes/clusters/dev/apps/kustomization-app-policies.yaml`
- Current policy: soft non-Ceph node placement preference for general workloads

Established deployment model:
- Git is the desired-state source of truth for cluster resources under `kubernetes/`.
- Container registry is the artifact source (images/charts), not the desired-state source.
- CI builds and publishes images; Flux selects allowed image tags and updates manifests via image automation.
- Post-deploy verification is automated through `task pipeline:verify`, including Flux readiness, Ceph health, OpenBao/ESO, Postgres, Kafka, observability, and the visit demo queue/count path.

Current pipeline gates on `main`:
- manual gate 1: `.github/workflows/opentofu-apply-destroy.yml` with `action=apply` provisions/updates infrastructure
- manual gate 2: `.github/workflows/ansible-run.yml` runs the ordered Ansible deployment chain after apply
- manual gate 3: `.github/workflows/opentofu-apply-destroy.yml` with `action=destroy` tears everything down (requires `destroy_confirm=yes`)

Phase 2 status:
- complete for defined AWS lab scope (HTTP ingress only; cert-manager/HTTPS intentionally excluded)

Phase 3 current status:
- Kafka baseline is implemented via Flux under `kubernetes/infrastructure/base/data-kafka/`
- Postgres baseline is implemented via Flux under `kubernetes/infrastructure/base/data-postgres/` (dev-only defaults)
- Ceph baseline is implemented via Rook, with `ceph-block` as the default StorageClass
- OpenBao baseline is implemented as the secret-management layer for External Secrets
- The current dev baseline is verified end-to-end by `pipeline:verify` after `pipeline:main`
- MongoDB lab remains pending/optional

Current open platform work:
- backup/restore: Velero plus component-specific restore drills
- secrets hardening: OpenBao auto-unseal, snapshots, dynamic DB credentials, secret rotation, and stronger init material custody
- security hardening: cluster auth/RBAC review, network policies, image scanning, and policy expansion
- application operations: load generation, HPA tuning for `visit-processor`, and alerting/lag dashboards
- update hygiene: Renovate or equivalent dependency/workflow update automation
