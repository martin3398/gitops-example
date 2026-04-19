# Agents.md

## Mission
Build and operate a reproducible Kubernetes platform lab that mirrors on-prem platform engineering practices while running on low-cost AWS EC2.

## Current Objective
Implement a multi-phase learning platform with:
- 3 control plane nodes and 2 worker nodes
- Open-source tooling
- GitOps with Flux
- CI/CD with GitLab

## Assessment
- The selected stack is strong and realistic for a Kubernetes platform engineer path.
- Using EC2 with self-managed Kubernetes is a good compromise between cost and on-prem relevance.
- Flux + GitLab CI/CD is an excellent open-source and GitLab-native combination.
- A phased rollout is the right approach, especially with multiple stateful systems planned.

## Rethink / Scope Control
- Keep strict ownership boundaries:
  - OpenTofu/Terraform: cloud infrastructure only
  - Ansible: host provisioning + Kubernetes bootstrap only
  - Flux: all cluster applications and platform add-ons
- Introduce heavyweight stateful tools one by one (Vault, MongoDB, Kafka, Ceph) to keep troubleshooting tractable.
- Delay backup and policy/security controls until the base platform and app delivery workflow are stable.
- Ceph should be last (or isolated) due to high operational complexity.

## Platform Decisions
- IaC: OpenTofu (Terraform-compatible)
- Config/Bootstrap: Ansible
- Kubernetes: kubeadm
- Runtime: containerd
- CNI: Cilium (fallback Calico if needed)
- GitOps: Flux
- CI/CD: GitLab CI/CD
- Ingress: ingress-nginx
- TLS: cert-manager
- Monitoring: kube-prometheus-stack + Loki + Grafana
- Database: Postgres primary/replica setup in cluster

## Target Topology
- Kubernetes nodes on EC2:
  - 3 control plane nodes
  - 2 worker nodes
- One VPC, private subnets for nodes, restricted administrative ingress.

## Delivery Phases

### Phase 1 - Infrastructure + Kubernetes Foundation
1. Provision AWS base infrastructure with OpenTofu/Terraform:
   - VPC, subnets, security groups, IAM roles, 5 EC2 instances
2. Bootstrap hosts with Ansible:
   - OS packages, hardening baseline, containerd, kubeadm prerequisites
3. Initialize cluster with kubeadm:
   - 3 control planes, 2 workers
   - Install CNI and validate cluster health

Exit criteria:
- All five nodes joined and Ready
- Control plane remains healthy after one-node reboot test

### Phase 2 - GitOps + Platform Services + Sample Workloads
1. Bootstrap Flux and connect to GitLab repository
2. Deploy platform services via Helm (through Flux):
   - ingress-nginx
   - cert-manager
   - monitoring/logging stack
3. Deploy sample application stack:
   - simple frontend
   - 1-2 microservices
   - Postgres primary/replica
4. Add GitLab CI/CD pipelines:
   - test/build images
   - publish images
   - update GitOps manifests/values

Exit criteria:
- Git push triggers CI and Flux reconciliation
- End-to-end app path is reachable through ingress
- Observability dashboards show app and cluster metrics

### Phase 3 - Advanced Tool Labs
Install and operate tools one at a time:
1. Vault
2. MongoDB
3. Kafka
4. Ceph (last)

Exit criteria per tool:
- Install is reproducible
- Basic operations validated
- Troubleshooting notes captured

### Phase 4 - Resilience, Backup, Policy & Security
1. Backups:
   - Velero for cluster resource backup/restore
   - Database-specific backup and restore tests
2. Policy/Security:
   - Kyverno (or OPA Gatekeeper)
   - Trivy image scanning in CI
   - baseline network policies
   - Pod security standards/admission configuration

Exit criteria:
- Restore drill passes
- Policy violations are detectable and actionable

## Operational Rules
- No long-lived manual drift in cluster resources; reconcile through Flux.
- Pin versions for Helm charts and critical platform components.
- Define resource requests/limits for all workloads.
- Document every major component with install, operate, and failure runbook notes.
- Include teardown steps for cost control.
- For this lab workflow, the human operator executes shell/OpenTofu/Ansible commands manually; agents provide plans, file changes, and command guidance only.

## Suggested Repository Layout
- `infra/` OpenTofu/Terraform infrastructure code
- `ansible/` host and cluster bootstrap automation
- `kubernetes/flux/` Flux bootstrap and Kustomizations
- `kubernetes/platform/` ingress, cert-manager, monitoring, logging
- `kubernetes/apps/` frontend, microservices, Postgres
- `kubernetes/labs/` Vault, MongoDB, Kafka, Ceph experiments
- `docs/` architecture, runbooks, troubleshooting, DR tests

## Definition of Done (Global)
- The environment can be created and destroyed from code.
- Flux continuously reconciles desired state from Git.
- CI/CD, observability, and stateful workloads operate as expected.
- Backup/restore and policy/security controls are tested and documented.
