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

- Kubernetes cluster on EC2 with `3 control planes` and `2 workers`
- Infrastructure provisioning with `OpenTofu/Terraform`
- Host bootstrap and kubeadm automation with `Ansible`
- GitOps delivery with `Flux`
- CI/CD with `GitLab CI/CD`
- Open-source platform tooling deployed with Helm via Flux

## Stack (Planned)

- Provisioning: OpenTofu/Terraform
- Automation: Ansible
- Kubernetes: kubeadm + containerd
- CNI: Cilium (fallback Calico)
- Ingress: ingress-nginx
- TLS: cert-manager
- GitOps: Flux
- CI/CD: GitLab CI/CD
- Monitoring/Logging: kube-prometheus-stack + Loki + Grafana
- Application data: Postgres primary/replica
- Advanced labs: Vault, MongoDB, Kafka, Ceph
- Later-stage reliability/security: Velero, Kyverno/Gatekeeper, Trivy, baseline network policies

## Delivery Plan

### Phase 1 - Infrastructure + Kubernetes Foundation
- Provision VPC, networking, IAM, and 5 EC2 nodes
- Configure hosts with Ansible (container runtime + kubeadm prerequisites)
- Bootstrap kubeadm cluster (3 control planes, 2 workers)
- Validate node health and basic failover behavior

### Phase 2 - GitOps Platform + Workloads
- Bootstrap Flux from GitLab
- Install platform add-ons via Helm through Flux
- Deploy frontend + 1-2 microservices + Postgres primary/replica
- Add GitLab pipelines for build/test/publish and GitOps update flow

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
- `kubernetes/platform/` ingress, cert-manager, monitoring/logging
- `kubernetes/apps/` frontend, services, database
- `kubernetes/labs/` Vault, MongoDB, Kafka, Ceph
- `docs/` architecture and runbooks

## Conventions

- Terraform/OpenTofu manages cloud resources only.
- Ansible manages host setup and cluster bootstrap only.
- Flux manages all long-lived Kubernetes resources.
- Avoid manual drift: persistent changes should be committed to Git.

## Cost and Safety Notes

- A 5-node cluster is not free-tier friendly; keep instance sizes small and monitor cost.
- Add destroy/recreate workflows early to control spend.
- Prefer reproducibility over manual fixes.

## Key Documents

- Agent prompt and project execution guidance: `Agents.md`
- CI pipeline configuration:
  - Root include file: `.gitlab-ci.yml`
  - OpenTofu pipeline: `.gitlab/ci/opentofu.yml`
- Local runner setup: `docker-compose.runner.yml`
- Phase 1 infrastructure runbook: `docs/phase1-infra-runbook.md`
- GitLab runner and CI variables guide: `docs/gitlab-runner-and-ci-vars.md`

## Status

Phase 1 infrastructure baseline is implemented:
- OpenTofu AWS networking + IAM + 5-node EC2 topology
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
