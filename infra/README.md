# OpenTofu Infrastructure (Phase 1)

Use this directory to provision the AWS infra for the dev cluster.

## Setup

1. Copy example variables:
   - `cp terraform.tfvars.example terraform.tfvars`
2. Update `allowed_admin_cidrs` and optional exposure settings in `terraform.tfvars`.
3. Keep `enable_ssh_from_admin_cidrs = false` for SSM-only access.
4. Configure remote state backend:
   - `cp backend.hcl.example backend.hcl`
   - Set your S3 bucket name in `backend.hcl`
5. Run OpenTofu:
   - `tofu init -backend-config=backend.hcl`
   - `tofu plan`
   - `tofu apply`

## Reference

- Infra resources, verification, destroy, and output handoff: `docs/phase1-infra-runbook.md`
- Ansible handoff output: `tofu output -json ansible_inventory`

## Notes

- Nodes are private only.
- NAT provides outbound internet access.
- Session Manager access is enabled via the EC2 instance profile.
- `backend.hcl` is local-only and should not be committed.
- Default environment name is `dev`.
