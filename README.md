# gitops-showcase

Hands-on Kubernetes platform engineering lab focused on GitOps, reproducible infrastructure, and on-prem transferable operations.

## Goal

This project is a practical training environment for a platform engineer role.

It uses AWS EC2 for convenience, but follows a self-managed approach designed to mirror on-prem responsibilities:
- infrastructure as code
- automated host and cluster bootstrap
- GitOps-driven platform and workload delivery
- operational practices (observability, backups, policy/security)

## Architecture Target

- Kubernetes cluster on EC2 with `3 control planes` and `3 workers`
- Infrastructure provisioning with `OpenTofu/Terraform`
- Host bootstrap and kubeadm automation with `Ansible`
- GitOps delivery with `Flux`
- CI/CD with `GitLab CI/CD`
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
- [ ] cert-manager (optional for public HTTPS/custom domains)
- [ ] Monitoring/logging baseline (kube-prometheus-stack + Loki + Grafana)
- [x] Application GitOps image-update automation (Flux image automation writes GitOps values)
- [ ] Application GitOps delivery (sample app + Postgres)
- [ ] Backups (Velero + restore drill)
- [ ] Policy/security controls (Kyverno/Gatekeeper, Trivy, network policies)

### Delivery Steps

- [x] Phase 1.1: AWS VPC, IAM, EC2 topology (3 CP + 3 workers)
- [x] Phase 1.2: Ansible inventory + SSM-first connectivity
- [x] Phase 1.3: Base node preparation (kernel/sysctl/swap/packages)
- [x] Phase 1.4: Runtime install (containerd + kubelet/kubeadm/kubectl)
- [x] Phase 1.5: Cluster bootstrap (kubeadm init/join + Cilium + validation)
- [x] Phase 1.6: GitLab CI stages for Ansible (manual-gated)
- [x] Phase 2.1: Flux bootstrap and repository structure under `kubernetes/`
- [x] Phase 2.2a: ingress-nginx via Flux
- [ ] Phase 2.2b (optional): cert-manager via Flux
- [ ] Phase 2.3: Observability baseline via Flux
- [x] Phase 2.4 (partial): Flux image automation wired for app image updates
- [ ] Phase 3: Advanced tooling labs (Vault, MongoDB, Kafka, Ceph)
- [ ] Phase 4: Resilience + backup + policy/security hardening

## Stack (Planned)

- Provisioning: OpenTofu/Terraform
- Automation: Ansible
- Kubernetes: kubeadm + containerd
- CNI: Cilium (fallback Calico)
- Ingress: ingress-nginx
- TLS: optional cert-manager
- GitOps: Flux
- CI/CD: GitLab CI/CD
- Monitoring/Logging: kube-prometheus-stack + Loki + Grafana
- Application data: Postgres primary/replica
- Advanced labs: Vault, MongoDB, Kafka, Ceph
- Later-stage reliability/security: Velero, Kyverno/Gatekeeper, Trivy, baseline network policies

## Delivery Plan

### Phase 1 - Infrastructure + Kubernetes Foundation
- Provision VPC, networking, IAM, and 6 EC2 nodes
- Configure hosts with Ansible (container runtime + kubeadm prerequisites)
- Bootstrap kubeadm cluster (3 control planes, 3 workers)
- Validate node health and basic failover behavior

### Phase 2 - GitOps Platform + Workloads
- Bootstrap Flux from GitLab
- Install platform add-ons via Helm through Flux
- Deploy frontend + 1-2 microservices + Postgres primary/replica
- Add GitLab pipelines for build/test/publish and use Flux image automation for GitOps image updates

### Phase 3 - Advanced Tooling
- Add Vault, MongoDB, Kafka, and Ceph one by one
- Capture setup and operations notes for each

### Phase 4 - Resilience, Backup, Policy & Security
- Add Velero and validate restore drills
- Add policy/security controls (Kyverno/Gatekeeper, Trivy, network policies)

## Repository Roadmap

Planned layout:
- `infra/` infrastructure code
- `ansible/` host and kubeadm automation
- `kubernetes/flux/` Flux bootstrap and Kustomizations
- `kubernetes/platform/` ingress, optional cert-manager, monitoring/logging
- `kubernetes/apps/` frontend, services, database
- `kubernetes/labs/` Vault, MongoDB, Kafka, Ceph
- `docs/` architecture and runbooks

## Conventions

- Terraform/OpenTofu manages cloud resources only.
- Ansible manages host setup and cluster bootstrap only.
- Flux manages all long-lived Kubernetes resources.
- Avoid manual drift: persistent changes should be committed to Git.

## Environment Naming

- The active environment name is `dev`.
- Flux cluster entrypoint is `kubernetes/flux/clusters/dev/`.
- Platform and app overlays are under `kubernetes/platform/dev/` and `kubernetes/apps/dev/`.

## Direct Kubernetes API Access (Dev Option)

If you want direct `kubectl` access from your workstation (without SSM tunnel), enable the public API NLB in `infra/terraform.tfvars`:

- `enable_public_k8s_api = true`
- `lb_public_subnet_cidrs = ["10.42.101.0/24", "10.42.102.0/24"]`
- keep `allowed_admin_cidrs` restricted to your current public IP (for example `/32`)

When enabled, infrastructure output includes both `kubernetes_api_internal_endpoint` (bootstrap) and `kubernetes_api_endpoint` (external access).
Bootstrap uses the internal endpoint (`kube_api_internal_endpoint`) for kubeadm stability.
The public NLB endpoint is exported as `kubernetes_api_endpoint` and `kube_api_public_endpoint` for external `kubectl` access.

## Public Ingress Access (Dev Option)

To expose application ingress publicly through an internet-facing NLB, enable these values in `infra/terraform.tfvars`:

- `enable_public_ingress = true`
- `ingress_allowed_cidrs = ["0.0.0.0/0"]` (or restrict to your IP/CIDR)
- `ingress_nodeport_http = 30080`
- `ingress_nodeport_https = 30443`

When enabled, OpenTofu exports `ingress_public_endpoint`.
Ingress traffic path is NLB (`:80/:443`) -> worker NodePorts (`30080/30443`) -> `ingress-nginx` controller.

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

This repository now includes a root `Taskfile.yml` that mirrors the current CI jobs for local execution.

Common usage:

```bash
task -l
task env:check
task pipeline:check
task tofu:plan
task tofu:apply
task ansible:all
task ansible:get_kubeconfig_public
```

Task groups:
- OpenTofu: `tofu:*`
- Ansible: `ansible:*`
- Pipeline orchestrators: `pipeline:*`

## Cost and Safety Notes

- A 6-node cluster is not free-tier friendly; keep instance sizes small and monitor cost.
- Add destroy/recreate workflows early to control spend.
- Prefer reproducibility over manual fixes.

## Key Documents

- Agent prompt and project execution guidance: `Agents.md`
- CI pipeline configuration:
  - Root include file: `.gitlab-ci.yml`
  - OpenTofu pipeline: `.gitlab/ci/opentofu.yml`
  - Ansible pipeline: `.gitlab/ci/ansible.yml`
- Local runner setup: `docker-compose.runner.yml`
- Phase 1 infrastructure runbook: `docs/phase1-infra-runbook.md`
- GitLab runner and CI variables guide: `docs/gitlab-runner-and-ci-vars.md`
- Ansible CI execution runbook: `docs/ansible-ci-runbook.md`

## Status

Phase 1 infrastructure baseline is implemented:
- OpenTofu AWS networking + IAM + 6-node EC2 topology
- S3 + DynamoDB remote state workflow
- GitLab CI pipeline for OpenTofu (check/plan/manual apply/manual destroy)
- Local Docker Compose GitLab runner path for early testing

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
- installs Flux controllers (including image reflector/automation) and applies Git source + cluster Kustomizations
- validates `GitRepository` and Kustomization readiness for `platform` and `apps`

GitLab CI now includes Ansible automation stages:
- inventory generation from OpenTofu apply artifact
- smoke, base, runtime, bootstrap, and Flux bootstrap jobs (auto-sequenced after manual `tofu:apply` on `main`)
- runbooks in `docs/gitlab-runner-and-ci-vars.md` and `docs/ansible-ci-runbook.md`

GitOps structure and Flux-managed charts are implemented:
- cluster entrypoint and source chain: `kubernetes/flux/clusters/dev/`
- platform root: `kubernetes/platform/dev/`
- apps root: `kubernetes/apps/dev/`
- platform ingress chart: `kubernetes/platform/dev/ingress-nginx/`
- smoke Helm chart: `kubernetes/apps/dev/podinfo/`

Flux image automation for apps is implemented:
- `ImageRepository`, `ImagePolicy` (stable semver `>=6.0.0 <7.0.0`), and `ImageUpdateAutomation` for podinfo
- image updates commit directly to `main` and are then reconciled by Flux

Established deployment model:
- Git is the desired-state source of truth for cluster resources under `kubernetes/`.
- Container registry is the artifact source (images/charts), not the desired-state source.
- CI builds and publishes images; Flux selects allowed image tags and updates manifests via image automation.

Current pipeline gates on `main`:
- manual gate 1: `tofu:apply` provisions infrastructure and then runs the full Ansible sequence
- manual gate 2: `tofu:destroy` tears everything down (requires `DESTROY_CONFIRM=yes`)

Phase 2 remaining work:
- platform add-ons (optional cert-manager, observability) via Flux are still pending
- sample application stack beyond podinfo smoke deployment is still pending
