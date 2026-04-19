# OpenTofu Infrastructure (Phase 1)

This directory provisions AWS infrastructure for a 5-node kubeadm cluster:
- 1 VPC
- 1 public subnet for a single NAT gateway
- 3 private subnets for Kubernetes nodes
- 3 control-plane EC2 instances and 2 worker EC2 instances

## Usage

1. Copy example variables:
   - `cp terraform.tfvars.example terraform.tfvars`
2. Update `allowed_admin_cidrs` in `terraform.tfvars`.
3. Keep `enable_ssh_from_admin_cidrs = false` for SSM-only access (recommended).
4. Configure remote state backend (recommended):
   - `cp backend.hcl.example backend.hcl`
   - Edit `backend.hcl` and set your real S3 bucket name.
5. Initialize and plan:
   - `tofu init -backend-config=backend.hcl`
   - `tofu plan`
6. Apply:
   - `tofu apply`

## Notes

- Nodes are private only (no public IPs).
- Outbound internet access is provided by a single NAT gateway.
- IAM instance profile includes AmazonSSMManagedInstanceCore for Session Manager access.
- SSH ingress is optional and disabled by default.
- Outputs include an Ansible-friendly inventory map.
- `backend.hcl` is local-only configuration and should not be committed.
- Security group rules are managed via standalone ingress rule resources.

## Operations

- Plan/apply workflow:
  - `tofu plan -out phase1-infra.tfplan`
  - `tofu apply "phase1-infra.tfplan"`
- Destroy workflow:
  - `tofu plan -destroy -out destroy.tfplan`
  - `tofu apply "destroy.tfplan"`
- Handoff output for bootstrap:
  - `tofu output -json ansible_inventory`
