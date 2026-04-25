# Phase 1 Infrastructure Runbook

This runbook describes the current OpenTofu-based infrastructure workflow for Phase 1.

## Scope

- Provisions AWS infrastructure only.
- Kubernetes bootstrap is handled in `ansible/` and documented in `ansible/README.md`.

## What Gets Created

- 1 VPC
- 1 public subnet for NAT
- 3 private subnets for nodes
- 1 Internet Gateway
- 1 NAT Gateway + Elastic IP
- 2 security groups (control plane, workers)
- IAM role + instance profile for EC2 nodes (SSM access)
- 3 control-plane EC2 instances and 2 worker EC2 instances
- Optional: internet-facing Kubernetes API NLB on `:6443` when `enable_public_k8s_api=true`

## Files

- `infra/providers.tf`: provider and backend declarations
- `infra/variables.tf`: input variables
- `infra/locals.tf`: naming, tags, node maps
- `infra/network.tf`: VPC/subnets/routes/NAT
- `infra/security.tf`: security groups and ingress rules
- `infra/iam.tf`: role/profile for nodes
- `infra/compute.tf`: EC2 fleet
- `infra/outputs.tf`: handoff outputs (`ansible_inventory`)

## Prerequisites

- OpenTofu installed locally
- AWS credentials configured (for example via `AWS_PROFILE`)
- S3 backend bucket + DynamoDB lock table created

## Local Workflow

1. Prepare variable files:
   - `cp infra/terraform.tfvars.example infra/terraform.tfvars`
   - `cp infra/backend.hcl.example infra/backend.hcl`
2. Update values:
    - `infra/terraform.tfvars`: `allowed_admin_cidrs`, optional instance sizes
    - `infra/backend.hcl`: real state bucket name and `key` path (default is `gitops-showcase/dev/infra.tfstate`)
3. Run OpenTofu:
   - `cd infra`
   - `tofu init -migrate-state -backend-config=backend.hcl` (first remote-state migration only)
   - `tofu init -backend-config=backend.hcl`
   - `tofu fmt -recursive`
   - `tofu validate`
   - `tofu plan -out phase1-infra.tfplan`
   - `tofu apply "phase1-infra.tfplan"`

## Verification

- `tofu output`
- `tofu output -json ansible_inventory`
- `tofu output kubernetes_api_endpoint`
- `tofu output kubernetes_api_internal_endpoint`
- `tofu plan` should be no-op after reconciliation

If public API is enabled:

- ensure `kubernetes_api_endpoint` resolves to NLB DNS
- ensure your current public IP is present in `allowed_admin_cidrs`
- verify `nc -vz <nlb-dns-name> 6443` from your workstation

## Destroy

- `tofu plan -destroy -out destroy.tfplan`
- `tofu apply "destroy.tfplan"`

## Known SG Rule Migration Note

When transitioning security group rule modeling, AWS may already contain equivalent rules and OpenTofu may fail with `InvalidPermission.Duplicate` for self-referencing rules.

Preferred resolution is importing those existing rule IDs into state instead of recreating all infrastructure.

## Exit Criteria (Phase 1 Infra Step)

- VPC and subnets are in place.
- NAT egress works for private node subnets.
- 3 control-plane and 2 worker instances exist.
- State is remote (S3) and locking is active (DynamoDB).
- Outputs provide clean handoff data for Ansible bootstrap.

## Current Completion Status

- [x] Infra provisioning workflow implemented
- [x] Remote state + locking implemented
- [x] CI plan/apply/destroy flow implemented
- [x] Handoff output (`ansible_inventory`) implemented
