# Phase 1 Infrastructure Runbook

This runbook describes the current OpenTofu-based infrastructure workflow for Phase 1.

## Scope

- Provisions AWS infrastructure only.
- Does not bootstrap Kubernetes yet (Ansible + kubeadm is next).

## What Gets Created

- 1 VPC
- 1 public subnet for NAT
- 3 private subnets for nodes
- 1 Internet Gateway
- 1 NAT Gateway + Elastic IP
- 2 security groups (control plane, workers)
- IAM role + instance profile for EC2 nodes (SSM access)
- 3 control-plane EC2 instances and 2 worker EC2 instances

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
   - `infra/backend.hcl`: real state bucket name
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
- `tofu plan` should be no-op after reconciliation

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
